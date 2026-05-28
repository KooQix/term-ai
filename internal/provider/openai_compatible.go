package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/KooQix/term-ai/internal/config"
	"github.com/KooQix/term-ai/internal/tools"
)

// OpenAICompatible implements the Provider interface for OpenAI-compatible APIs
type OpenAICompatible struct {
	Endpoint    string
	APIKey      string
	Model       string
	Temperature float64
	MaxTokens   int
	TopP        float64
}

type chatRequest struct {
	Model       string        `json:"model"`
	Tools       []tools.Tool  `json:"tools,omitempty"`
	Messages    []interface{} `json:"messages"` // Can be Message or messageWithContent
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
	Stream      bool          `json:"stream"`
}

// messageWithContent is used when images are present
type messageWithContent struct {
	Role    ContextRole   `json:"role"`
	Content []contentPart `json:"content"`
}

// contentPart represents a part of message content (text or image)
type contentPart struct {
	Type     string    `json:"type"` // "text" or "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

// imageURL wraps the image URL
type imageURL struct {
	URL string `json:"url"`
}

type chatResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
}

// NewOpenAICompatible creates a new OpenAI-compatible provider
func NewOpenAICompatible(endpoint, apiKey, model string, temperature float64, maxTokens int, topP float64) *OpenAICompatible {
	return &OpenAICompatible{
		Endpoint:    endpoint,
		APIKey:      apiKey,
		Model:       model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		TopP:        topP,
	}
}

// formatMessages converts Message structs to the appropriate format for the API
func formatMessages(messages []Message) []interface{} {
	formatted := make([]interface{}, len(messages))

	for i, msg := range messages {
		// If the message has images, use the content array format
		if len(msg.Images) > 0 {
			content := []contentPart{
				{
					Type: "text",
					Text: msg.Content,
				},
			}

			// Add images
			for _, imgURL := range msg.Images {
				content = append(content, contentPart{
					Type: "image_url",
					ImageURL: &imageURL{
						URL: imgURL,
					},
				})
			}

			formatted[i] = messageWithContent{
				Role:    msg.Role,
				Content: content,
			}
		} else {
			// No images, use simple message format
			formatted[i] = msg
		}
	}

	return formatted
}

// request sends the chat request to the API and returns the raw HTTP response
func (p *OpenAICompatible) request(ctx context.Context, chatReq chatRequest) (*http.Response, error) {
	jsonData, _ := json.Marshal(chatReq)
	url := strings.TrimSuffix(p.Endpoint, "/") + "/chat/completions"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API %d: %s", resp.StatusCode, body)
	}

	return resp, nil
}

func (p *OpenAICompatible) chatMessage(messages []Message, stream bool) chatRequest {
	return chatRequest{
		Model:       p.Model,
		Messages:    formatMessages(messages),
		Tools:       tools.AvailableTools(),
		Temperature: p.Temperature,
		MaxTokens:   p.MaxTokens,
		TopP:        p.TopP,
		Stream:      stream,
	}
}

// send dispatches the chat request and returns the raw response
func (p *OpenAICompatible) send(ctx context.Context, messages []Message, stream bool) (*http.Response, error) {
	chatReq := p.chatMessage(messages, stream)
	return p.request(ctx, chatReq)
}

// parse the API response into chatResponse struct
func (p *OpenAICompatible) parse(resp *http.Response) (chatResponse, error) {
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		resp.Body.Close()
		return chatResponse{}, err
	}
	resp.Body.Close()

	if len(cr.Choices) == 0 {
		return chatResponse{}, fmt.Errorf("no choices")
	}

	return cr, nil
}

func (p *OpenAICompatible) streamOnce(
	ctx context.Context,
	messages []Message,
	out chan<- StreamChunk,
) (string, []tools.ToolCall, error) {

	resp, err := p.send(ctx, messages, true)
	if err != nil {
		return "", nil, err
	}

	// Accumulator: index -> partial ToolCall
	toolAcc := map[int]*tools.ToolCall{}
	finishReason := ""

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, err
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 || !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}

		var sr streamResponse
		if err := json.Unmarshal(data, &sr); err != nil {
			continue
		}
		if len(sr.Choices) == 0 {
			continue
		}
		ch := sr.Choices[0]

		// Text delta
		if ch.Delta.Content != "" || ch.Delta.Thinking != "" {
			out <- StreamChunk{
				Content:  ch.Delta.Content,
				Thinking: ch.Delta.Thinking,
			}
		}

		// Tool call deltas — accumulate
		for _, tcDelta := range ch.Delta.ToolCalls {
			acc, ok := toolAcc[tcDelta.Index]
			if !ok {
				acc = &tools.ToolCall{Type: "function"}
				toolAcc[tcDelta.Index] = acc
			}
			if tcDelta.ID != "" {
				acc.ID = tcDelta.ID
			}
			if tcDelta.Function.Name != "" {
				acc.Function.Name = tcDelta.Function.Name
			}
			// Arguments stream in fragments — concatenate
			acc.Function.Arguments += tcDelta.Function.Arguments
		}

		if ch.FinishReason != nil {
			finishReason = *ch.FinishReason
		}
	}

	// Flatten accumulator into ordered slice
	calls := make([]tools.ToolCall, 0, len(toolAcc))
	for i := 0; i < len(toolAcc); i++ {
		if tc, ok := toolAcc[i]; ok {
			calls = append(calls, *tc)
		}
	}

	return finishReason, calls, nil
}

func (p *OpenAICompatible) Stream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	out := make(chan StreamChunk)

	go func() {
		defer close(out)

		for iter := 0; iter < config.AppConfig.ToolConfigs.MaxIter; iter++ {
			finished, toolCalls, err := p.streamOnce(ctx, messages, out)
			if err != nil {
				out <- StreamChunk{Error: err}
				return
			}

			if finished == "stop" || finished == "" {
				out <- StreamChunk{Done: true}
				return
			}

			if finished != "tool_calls" || len(toolCalls) == 0 {
				out <- StreamChunk{Done: true}
				return
			}

			// Append assistant message with tool_calls
			messages = append(messages, Message{
				Role:      RoleAssistant,
				Content:   "",
				ToolCalls: toolCalls,
			})

			// Execute each tool and append result
			for _, tc := range toolCalls {
				out <- StreamChunk{ToolCall: &tools.ToolCallEvent{
					Name: tc.Function.Name,
					Args: tc.Function.Arguments,
				}}

				result, err := tools.ExecuteTool(tc.Function.Name, tc.Function.Arguments)
				if err != nil {
					result = fmt.Sprintf(`{"error": %q}`, err.Error())
				}

				out <- StreamChunk{ToolCall: &tools.ToolCallEvent{
					Name:   tc.Function.Name,
					Args:   tc.Function.Arguments,
					Result: result,
				}}

				messages = append(messages, Message{
					Role:       RoleTool,
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
					Content:    result,
				})
			}
			// Loop again — model now has tool results
		}

		out <- StreamChunk{Error: fmt.Errorf("max tool iterations reached")}
	}()

	return out, nil
}

// Complete implements non-streaming chat completion
func (p *OpenAICompatible) Complete(ctx context.Context, messages []Message) (string, error) {

	res, err := p.send(ctx, messages, false)
	if err != nil {
		return "", err
	}

	chatResp, err := p.parse(res)
	if err != nil {
		return "", err
	}

	return chatResp.Choices[0].Message.Content, nil
}

// CompleteWithTools runs the full tool-execution loop.
func (p *OpenAICompatible) CompleteWithTools(ctx context.Context, messages []Message) (string, []Message, error) {
	for i := 0; i < config.AppConfig.ToolConfigs.MaxIter; i++ {
		res, err := p.send(ctx, messages, false)
		if err != nil {
			return "", messages, err
		}

		chatResp, err := p.parse(res)
		if err != nil {
			return "", messages, err
		}

		choice := chatResp.Choices[0]
		assistantMsg := choice.Message
		assistantMsg.Role = RoleAssistant
		messages = append(messages, assistantMsg)

		// Done?
		if choice.FinishReason != "tool_calls" || len(assistantMsg.ToolCalls) == 0 {
			return assistantMsg.Content, messages, nil
		}

		// Execute each tool call, append a role:"tool" message per call
		for _, tc := range assistantMsg.ToolCalls {
			result, err := tools.ExecuteTool(tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf(`{"error": %q}`, err.Error())
			}

			messages = append(messages, Message{
				Role:       RoleTool,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result,
			})
		}
	}
	return "", messages, fmt.Errorf("tool loop exceeded %d iterations", config.AppConfig.ToolConfigs.MaxIter)
}
