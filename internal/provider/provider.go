package provider

import (
	"context"

	"github.com/KooQix/term-ai/internal/tools"
)

type ContextRole string

const (
	RoleUser      ContextRole = "user"
	RoleAssistant ContextRole = "assistant"
	RoleSystem    ContextRole = "system"
	RoleTool      ContextRole = "tool"
)

// Message represents a chat message
type Message struct {
	Role       ContextRole      `json:"role"`
	Content    string           `json:"content"`
	Images     []string         `json:"images,omitempty"` // base64 data URLs for images
	ToolCalls  []tools.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"` // for role="tool"
	Name       string           `json:"name,omitempty"`         // optional, tool name
}

// StreamChunk represents a chunk of streamed response
type StreamChunk struct {
	Content  string
	Thinking string
	ToolCall *tools.ToolCallEvent // notify UI a tool is running
	Done     bool
	Error    error
}

type streamResponse struct {
	Choices []struct {
		Delta struct {
			Content   string           `json:"content"`
			Thinking  string           `json:"thinking,omitempty"`
			ToolCalls []tools.ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// Provider defines the interface for AI providers
type Provider interface {
	// Stream sends a chat request and returns a channel of streaming chunks
	Stream(ctx context.Context, messages []Message) (<-chan StreamChunk, error)

	// Complete sends a chat request and returns the complete response
	Complete(ctx context.Context, messages []Message) (string, error)

	CompleteWithTools(ctx context.Context, messages []Message) (string, []Message, error)
}
