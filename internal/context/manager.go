package context

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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

var msgSeparator = strings.Repeat("-", 50)

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
		// Read line by line
		line := scanner.Text()
		if line == msgSeparator {
			continue
		}

		var role provider.ContextRole
		switch {
		case strings.Contains(line, fmt.Sprintf("%s: ", provider.RoleUser)):
			role = provider.RoleUser
		case strings.Contains(line, fmt.Sprintf("%s: ", provider.RoleAssistant)):
			role = provider.RoleAssistant
		case strings.Contains(line, fmt.Sprintf("%s: ", provider.RoleSystem)):
			role = provider.RoleSystem
		default:
			fmt.Printf("Unknown role in line: %s\n", line)
			continue
		}

		msgBody := strings.TrimPrefix(line, fmt.Sprintf("%s: ", role))

		// Read the next lines until the separator to get the full message body
		for scanner.Scan() {
			nextLine := scanner.Text()

			// Break if we reach the separator - end of message
			if nextLine == msgSeparator {
				break
			}

			// Append to message body
			msgBody += "\n" + nextLine
		}

		// We reached the end of the message, create the message struct
		msg := provider.Message{
			Role:    role,
			Content: msgBody,
		}
		m.messages = append(m.messages, msg)
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

	// Check if the directory exists, create it if not
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create conversation directory: %w", err)
		}
	}

	// If file exists, remove it (so it will be recreated)
	if _, err := os.Stat(filePath); err == nil {
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove existing conversation file: %w", err)
		}
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer file.Close()

	// Write all messages to the file
	for _, msg := range m.messages {
		_, err := file.WriteString(string(msg.Role) + ": " + msg.Content + "\n" + msgSeparator + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}
