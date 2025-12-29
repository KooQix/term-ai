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
		Message Message `json:"message"`
	} `json:"choices"`
}

type streamResponse struct {
	Choices []struct {
		Delta struct {
			Content  string `json:"content"`
			Thinking string `json:"thinking,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
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
			formatted[i] = Message{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}
	}

	return formatted
}

// Stream implements streaming chat completion
func (p *OpenAICompatible) Stream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	chatReq := chatRequest{
		Model:       p.Model,
		Messages:    formatMessages(messages),
		Temperature: p.Temperature,
		MaxTokens:   p.MaxTokens,
		TopP:        p.TopP,
		Stream:      true,
	}

	jsonData, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(p.Endpoint, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	chunkChan := make(chan StreamChunk)

	go func() {
		defer resp.Body.Close()
		defer close(chunkChan)

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					chunkChan <- StreamChunk{Error: err}
				}
				return
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			if !bytes.HasPrefix(line, []byte("data: ")) {
				continue
			}

			data := bytes.TrimPrefix(line, []byte("data: "))
			if bytes.Equal(data, []byte("[DONE]")) {
				chunkChan <- StreamChunk{Done: true}
				return
			}

			var streamResp streamResponse
			if err := json.Unmarshal(data, &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				delta := streamResp.Choices[0].Delta
				if delta.Content != "" || delta.Thinking != "" {
					chunkChan <- StreamChunk{
						Content:  delta.Content,
						Thinking: delta.Thinking,
					}
				}
				if streamResp.Choices[0].FinishReason != nil {
					chunkChan <- StreamChunk{Done: true}
					return
				}
			}
		}
	}()

	return chunkChan, nil
}

// Complete implements non-streaming chat completion
func (p *OpenAICompatible) Complete(ctx context.Context, messages []Message) (string, error) {
	chatReq := chatRequest{
		Model:       p.Model,
		Messages:    formatMessages(messages),
		Temperature: p.Temperature,
		MaxTokens:   p.MaxTokens,
		TopP:        p.TopP,
		Stream:      false,
	}

	jsonData, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(p.Endpoint, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return chatResp.Choices[0].Message.Content, nil
}
