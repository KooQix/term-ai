package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/KooQix/term-ai/internal/config"
	ctxmanager "github.com/KooQix/term-ai/internal/context"
	"github.com/KooQix/term-ai/internal/fileprocessor"
	"github.com/KooQix/term-ai/internal/provider"
	"github.com/KooQix/term-ai/internal/ui"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
)

type ChatCommands struct {
	Commands  []string
	Available string
}

type chatModel struct {
	textarea           textarea.Model
	viewport           viewport.Model
	messages           []string
	ctxManager         *ctxmanager.Manager
	provider           provider.Provider
	profile            *config.Profile
	streaming          bool
	currentResp        string
	streamChan         <-chan provider.StreamChunk
	err                error
	ready              bool
	suggestions        []string
	selectedSuggestion int
	showSuggestions    bool
	// File attachments
	attachedFiles  []*fileprocessor.FileAttachment
	contextFiles   []*fileprocessor.FileAttachment
	contextDirPath string

	chatPath string // Path to save/load conversation (only set when saving/loading)

	commands        ChatCommands
	commandsHandler *commandHandler
}

func NewChatModel(cfg *config.Config, ta textarea.Model, vp viewport.Model, prov provider.Provider, profile *config.Profile, commands ChatCommands) chatModel {
	m := chatModel{
		textarea:   ta,
		viewport:   vp,
		messages:   []string{},
		ctxManager: ctxmanager.NewManager(),
		provider:   prov,
		profile:    profile,

		commands: commands,
	}

	// Add the command handler
	m.commandsHandler = newCommandHandler(&m)

	// Add system config if defined in config
	// Get from the profile (can be nil, empty, or non-empty)
	// Profile context set by cfg.GetProfile
	if *profile.SystemContext != "" {
		m.ctxManager.AddSystemMessage(*profile.SystemContext)
	}

	return m
}

type streamMsg struct {
	chunk   provider.StreamChunk
	channel <-chan provider.StreamChunk
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

func (m chatModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	// Check if this is a paste event for optimization
	isPaste := false
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		isPaste = keyMsg.Paste
	}

	// Update textarea first (always needed for input)
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	// Skip expensive operations during paste for better performance
	if !isPaste {
		// Update suggestions based on input (only when not pasting)
		m.updateSuggestions()
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 6 // Compact layout for maximum viewport space
		m.textarea.SetWidth(msg.Width - 7) // Account for "You ▸ " prompt
		m.viewport.SetContent(wordwrap.String(m.messages[len(m.messages)-1], m.viewport.Width))
		m.ready = true
		m.updateViewport()

	case tea.KeyMsg:
		// Handle auto-completion navigation
		if m.showSuggestions && len(m.suggestions) > 0 {
			switch msg.Type {
			case tea.KeyTab, tea.KeyDown:
				// Move to next suggestion
				m.selectedSuggestion = (m.selectedSuggestion + 1) % len(m.suggestions)
				return m, nil
			case tea.KeyShiftTab, tea.KeyUp:
				// Move to previous suggestion
				m.selectedSuggestion--
				if m.selectedSuggestion < 0 {
					m.selectedSuggestion = len(m.suggestions) - 1
				}
				return m, nil
			case tea.KeyEnter:
				// If suggestions are showing and Enter is pressed, accept the selected suggestion
				m.textarea.SetValue(m.suggestions[m.selectedSuggestion])
				m.showSuggestions = false
				m.suggestions = nil
				m.selectedSuggestion = 0
				return m, nil
			}
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			// Check for Alt+Enter or Ctrl+Enter to send message
			if msg.Alt || strings.Contains(msg.String(), "ctrl+enter") {
				// Send message with Alt+Enter or Ctrl+Enter
				if m.streaming {
					return m, nil
				}
				userMsg := strings.TrimSpace(m.textarea.Value())
				if userMsg == "" {
					return m, nil
				}

				// Handle commands
				if strings.HasPrefix(userMsg, "/") {
					return m.handleCommand(userMsg)
				}

				// Add user message
				m.messages = append(m.messages, "")
				m.messages = append(m.messages, ui.FormatUserMessage(userMsg))
				m.messages = append(m.messages, "")
				m.ctxManager.AddUserMessage(userMsg)
				m.textarea.Reset()
				m.updateViewport()

				// Start streaming
				m.streaming = true
				m.currentResp = ""
				m.messages = append(m.messages, ui.AssistantStyle.Render("Assistant: "))
				m.updateViewport()

				return m, m.streamResponse()
			}
			// Regular Enter without modifiers - let textarea handle it (adds newline)
			// Fall through to default textarea behavior
		}

	case streamMsg:
		// Store or update the channel
		if msg.channel != nil {
			m.streamChan = msg.channel
		}

		if msg.chunk.Error != nil {
			m.err = msg.chunk.Error
			m.streaming = false
			m.streamChan = nil
			m.messages = append(m.messages, ui.FormatError(msg.chunk.Error))
			m.updateViewport()
			return m, nil
		}

		if msg.chunk.Content != "" {
			m.currentResp += msg.chunk.Content
			// Update last message with accumulated content
			if len(m.messages) > 0 {
				m.messages[len(m.messages)-1] = ui.AssistantStyle.Render("Assistant: ") + m.currentResp
			}
			m.updateViewport()
		}

		if msg.chunk.Done {
			m.streaming = false
			m.streamChan = nil
			m.ctxManager.AddAssistantMessage(m.currentResp)

			// Clear attached files after successful send
			m.attachedFiles = nil

			// Format the complete response with syntax highlighting
			formatted, err := ui.FormatResponse(m.currentResp)
			if err != nil {
				// If formatting fails, use the original response
				formatted = m.currentResp
			}

			// Replace the last message with formatted version
			if len(m.messages) > 0 {
				m.messages[len(m.messages)-1] = ui.AssistantStyle.Render("Assistant:\n") + formatted
			}

			m.messages = append(m.messages, "")
			m.messages = append(m.messages, ui.FormatSeparator())
			m.updateViewport()
			return m, nil
		}

		// Continue reading from stream
		if m.streamChan != nil {
			return m, subscribeToStream(m.streamChan)
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		m.streaming = false
		m.messages = append(m.messages, ui.FormatError(msg.err))
		m.updateViewport()
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m chatModel) handleCommand(cmd string) (tea.Model, tea.Cmd) {
	chatModel, errCmd := m.commandsHandler.handle(cmd)

	if errCmd != nil {
		return chatModel, errCmd
	}

	m.textarea.Reset()
	m.updateViewport()
	return chatModel, nil
}

func (m chatModel) streamResponse() tea.Cmd {
	// Start streaming
	return func() tea.Msg {
		ctx := context.Background()

		// Get messages from context manager
		messages := m.ctxManager.GetMessages()

		// If we have attached or context files, modify the last user message
		if len(m.attachedFiles) > 0 || len(m.contextFiles) > 0 {
			if len(messages) > 0 {
				lastMsg := &messages[len(messages)-1]

				// Combine attached and context files
				allFiles := append([]*fileprocessor.FileAttachment{}, m.attachedFiles...)
				allFiles = append(allFiles, m.contextFiles...)

				// Separate images from text content
				var images []string
				var textContent strings.Builder
				textContent.WriteString(lastMsg.Content)

				for _, file := range allFiles {
					switch file.Type {
					case "image":
						images = append(images, file.Content)
					case "pdf", "text", "code":
						textContent.WriteString(fmt.Sprintf("\n\n--- Content from %s ---\n%s\n--- End of %s ---",
							file.Name, file.Content, file.Name))
					}
				}

				// Update the message
				lastMsg.Content = textContent.String()
				if len(images) > 0 {
					lastMsg.Images = images
				}
			}
		}

		chunkChan, err := m.provider.Stream(ctx, messages)
		if err != nil {
			return errMsg{err}
		}

		// Read first chunk
		chunk, ok := <-chunkChan
		if !ok {
			return streamMsg{
				chunk:   provider.StreamChunk{Done: true},
				channel: nil,
			}
		}

		return streamMsg{
			chunk:   chunk,
			channel: chunkChan,
		}
	}
}

func subscribeToStream(chunkChan <-chan provider.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		// Read next chunk
		chunk, ok := <-chunkChan
		if !ok {
			return streamMsg{
				chunk:   provider.StreamChunk{Done: true},
				channel: nil,
			}
		}
		return streamMsg{
			chunk:   chunk,
			channel: chunkChan,
		}
	}
}

func (m *chatModel) loadConversation(path string) error {
	// Load the conversation into the context manager
	if err := m.ctxManager.Load(path); err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}

	// Attach all the context messages to the chat view
	numMessages := 0
	for _, msg := range m.ctxManager.GetMessages() {
		numMessages++
		if msg.Role == provider.RoleSystem {
			continue // Skip system messages in chat view
		}

		var formatted string
		if msg.Role == provider.RoleUser {
			formatted = ui.FormatUserMessage(msg.Content)
		} else if msg.Role == provider.RoleAssistant {
			// Format the response with syntax highlighting
			resp, err := ui.FormatResponse(msg.Content)
			if err != nil {
				// If formatting fails, use the original response
				resp = msg.Content
			}
			// Add the "Assistant:" prefix after formatting
			formatted = ui.AssistantStyle.Render("Assistant:\n") + resp
		}

		m.messages = append(m.messages, formatted)
		m.messages = append(m.messages, "")
		m.messages = append(m.messages, ui.FormatSeparator())
	}

	m.messages = append(m.messages, ui.FormatSuccess(fmt.Sprintf("Conversation loaded from '%s', %d messages", path, numMessages)))

	m.updateViewport()

	return nil
}

func (m *chatModel) AttachFiles(files []*fileprocessor.FileAttachment) {
	m.attachedFiles = append(m.attachedFiles, files...)
}

func (m *chatModel) AddContextFiles(files []*fileprocessor.FileAttachment) {
	m.contextFiles = append(m.contextFiles, files...)
}

func (m *chatModel) SetContextDir(dir string) {
	m.contextDirPath = dir
}

func (m *chatModel) AddMessage(message string) {
	m.messages = append(m.messages, message)
	m.messages = append(m.messages, "")
}
