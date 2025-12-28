package provider

import "context"

// Message represents a chat message
type Message struct {
        Role    string   `json:"role"`
        Content string   `json:"content"`
        Images  []string `json:"images,omitempty"` // base64 data URLs for images
}

// StreamChunk represents a chunk of streamed response
type StreamChunk struct {
        Content  string
        Thinking string
        Done     bool
        Error    error
}

// Provider defines the interface for AI providers
type Provider interface {
        // Stream sends a chat request and returns a channel of streaming chunks
        Stream(ctx context.Context, messages []Message) (<-chan StreamChunk, error)
        
        // Complete sends a chat request and returns the complete response
        Complete(ctx context.Context, messages []Message) (string, error)
}
