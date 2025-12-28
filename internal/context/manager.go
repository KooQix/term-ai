package context

import "github.com/KooQix/term-ai/internal/provider"

// Manager handles conversation context
type Manager struct {
	messages []provider.Message
}

// NewManager creates a new context manager
func NewManager() *Manager {
	return &Manager{
		messages: make([]provider.Message, 0),
	}
}

// AddUserMessage adds a user message to the context
func (m *Manager) AddUserMessage(content string) {
	m.messages = append(m.messages, provider.Message{
		Role:    "user",
		Content: content,
	})
}

// AddAssistantMessage adds an assistant message to the context
func (m *Manager) AddAssistantMessage(content string) {
	m.messages = append(m.messages, provider.Message{
		Role:    "assistant",
		Content: content,
	})
}

// GetMessages returns all messages
func (m *Manager) GetMessages() []provider.Message {
	return m.messages
}

// Clear clears all messages
func (m *Manager) Clear() {
	m.messages = make([]provider.Message, 0)
}

// IsEmpty returns true if there are no messages
func (m *Manager) IsEmpty() bool {
	return len(m.messages) == 0
}
