package context

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/KooQix/term-ai/internal/config"
	"github.com/KooQix/term-ai/internal/provider"
)

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
		Role:    provider.RoleUser,
		Content: content,
	})
}

// AddAssistantMessage adds an assistant message to the context
func (m *Manager) AddAssistantMessage(content string) {
	m.messages = append(m.messages, provider.Message{
		Role:    provider.RoleAssistant,
		Content: content,
	})
}

func (m *Manager) AddSystemMessage(content string) {
	m.messages = append(m.messages, provider.Message{
		Role:    provider.RoleSystem,
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

// Retrieve the last message of the conversation
func (m *Manager) GetLastMessage() *provider.Message {
	if len(m.messages) == 0 {
		return nil
	}
	return &m.messages[len(m.messages)-1]
}

//////////////////// Saving and loading conversations \\\\\\\\\\\\\\\\\\\\

var msgSeparator = strings.Repeat("-", 50) + "\n"

// Load loads the conversation from a file
func (m *Manager) Load(filePath string) error {
	if !strings.HasSuffix(filePath, config.ConversationFileExt) {
		return fmt.Errorf("Invalid file path: %s. Conversation must be a valid file, and including the %s extension", filePath, config.ConversationFileExt)
	}

	// Now check that the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("Conversation file does not exist: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	m.messages = make([]provider.Message, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == msgSeparator {
			continue
		}

		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			// Build the entire message content (in case of multiple lines)
			msgBody := parts[1]
			for scanner.Scan() {
				nextLine := scanner.Text()
				if nextLine == msgSeparator {
					break
				}
				msgBody += "\n" + nextLine
			}

			msg := provider.Message{
				Role:    provider.ContextRole(parts[0]),
				Content: msgBody,
			}
			m.messages = append(m.messages, msg)
		}
	}

	return scanner.Err()
}

// Save appends the conversation to an existing file
// This assumes the path exists, and the filePath is valid and absolute (use utils.GetAbsolutePath helper)
func (m *Manager) Save(filePath string) error {
	// Filepath is path/to/conversation/conversation-name.termai.md
	// Ensure the conversation name has the correct extension
	if !strings.HasSuffix(filePath, config.ConversationFileExt) {
		filePath += config.ConversationFileExt
	}

	var file *os.File

	// If file doesn't exist, create it
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err = os.Create(filePath)
		if err != nil {
			return err
		}
	} else {
		file, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}

	}

	defer file.Close()

	for _, msg := range m.messages {
		_, err := file.WriteString(string(msg.Role) + ": " + msg.Content + "\n" + msgSeparator)
		if err != nil {
			return err
		}
	}

	return nil
}
