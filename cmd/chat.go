package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KooQix/term-ai/internal/config"
	ctxmanager "github.com/KooQix/term-ai/internal/context"
	"github.com/KooQix/term-ai/internal/fileprocessor"
	"github.com/KooQix/term-ai/internal/provider"
	"github.com/KooQix/term-ai/internal/ui"
	"github.com/KooQix/term-ai/internal/utils"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const availableCommands = `Available commands:
  /exit or /quit - Exit chat
  /clear - Clear conversation context
  /profile - Show current profile info
  /attach <file> [...] - Attach one or more files
  /files - Show currently attached files
  /clear-files - Clear all attached files
  /context - Show context files from directory
  /context-add <file> [...] - Add files to context
  /context-remove <file> - Remove file from context
  /save <name> -d <optional-directory> - Save conversation
  /load <path> - Load conversation from file
  /help - Show this help`

var (
	chatFilePaths []string
	contextDir    string

	chatCmd = &cobra.Command{
		Use:   "chat",
		Short: "Start interactive chat session",
		Long:  `Start an interactive chat session with conversation context.`,
		RunE:  runChat,
	}

	// Available chat commands for auto-completion
	chatCommands = []string{"/help", "/exit", "/quit", "/clear", "/profile", "/attach", "/files", "/clear-files", "/context", "/context-add", "/context-remove", "/save", "/load"}
)

func init() {
	chatCmd.Flags().StringArrayVarP(&chatFilePaths, "file", "f", []string{}, "File(s) to attach (can be used multiple times)")
	chatCmd.Flags().StringVarP(&contextDir, "dir", "d", "", "Directory to use as context (scans for supported files)")
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
}

func NewChatModel(cfg *config.Config, ta textarea.Model, vp viewport.Model, prov provider.Provider, profile *config.Profile) chatModel {
	m := chatModel{
		textarea:   ta,
		viewport:   vp,
		messages:   []string{},
		ctxManager: ctxmanager.NewManager(),
		provider:   prov,
		profile:    profile,
	}

	// Add system config if defined in config
	if cfg.SystemContext != "" {
		m.ctxManager.AddSystemMessage(cfg.SystemContext)
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

func runChat(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get profile
	var profile *config.Profile
	if profileName != "" {
		profile, err = cfg.GetProfile(profileName)
	} else {
		profile, err = cfg.GetDefaultProfile()
	}
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	// Check for placeholder API key
	if profile.APIKey == "" || profile.APIKey == "your-abacus-api-key" || profile.APIKey == "your-openai-api-key" {
		return fmt.Errorf("please set a valid API key for profile '%s' in your config file\nEdit config with: termai config edit", profile.Name)
	}

	// Create provider
	prov := provider.NewOpenAICompatible(
		profile.Endpoint,
		profile.APIKey,
		profile.Model,
		profile.Temperature,
		profile.MaxTokens,
		profile.TopP,
	)

	// Create chat model
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Alt+Enter or Ctrl+Enter to send)"
	ta.Focus()
	ta.CharLimit = 5000
	ta.SetWidth(80)
	ta.SetHeight(3)

	vp := viewport.New(80, 20)

	m := chatModel{
		textarea:   ta,
		viewport:   vp,
		messages:   []string{},
		ctxManager: ctxmanager.NewManager(),
		provider:   prov,
		profile:    profile,
	}

	// Add welcome message
	welcome := fmt.Sprintf("Welcome to TermAI Interactive Chat!\nUsing profile: %s (%s)\n\n", profile.Name, profile.Model)
	welcome += availableCommands

	// Process initial files if provided
	if len(chatFilePaths) > 0 {
		fmt.Print("Processing initial files... ")
		attachments, err := fileprocessor.ProcessFiles(chatFilePaths)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			m.attachedFiles = attachments
			fmt.Printf("✓ %d file(s) attached\n", len(attachments))
			welcome += fmt.Sprintf("📎 %d file(s) attached and ready\n", len(attachments))
		}
	}

	// Process directory context if provided
	if contextDir != "" {
		fmt.Print("Scanning directory context... ")
		contextFiles, err := scanDirectory(contextDir)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			m.contextFiles = contextFiles
			m.contextDirPath = contextDir
			fmt.Printf("✓ %d file(s) in context\n", len(contextFiles))
			welcome += fmt.Sprintf("📁 Context: %s (%d files)\n", contextDir, len(contextFiles))
		}
	}

	welcome += "\n" + ui.FormatSeparator() + "\n"
	m.messages = append(m.messages, welcome)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// scanDirectory scans a directory for supported files
func scanDirectory(dirPath string) ([]*fileprocessor.FileAttachment, error) {
	// Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dirPath)
	}

	var filePaths []string

	// Walk the directory (only top level by default for safety)
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip subdirectories (only process top level)
			if path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file is supported
		if fileprocessor.IsSupported(path) {
			filePaths = append(filePaths, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error scanning directory: %w", err)
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no supported files found in directory")
	}

	// Process all found files
	return fileprocessor.ProcessFiles(filePaths)
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

func (m chatModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var sb strings.Builder

	// Render header (compact, single-line)
	sb.WriteString(m.renderHeader())
	sb.WriteString("\n")

	// Render viewport (chat messages) - maximize this area
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	// Render input separator (no extra spacing)
	sb.WriteString(ui.FormatSeparator())
	sb.WriteString("\n")

	// Render input area (compact)
	sb.WriteString(m.renderInputArea())

	// Render suggestions if showing (inline, no extra newline)
	if m.showSuggestions && len(m.suggestions) > 0 {
		sb.WriteString("\n")
		sb.WriteString(m.renderSuggestions())
	}

	// Render footer (compact, single-line)
	sb.WriteString("\n")
	sb.WriteString(m.renderFooter())

	return sb.String()
}

func (m chatModel) renderHeader() string {
	// Status indicator
	status := "Ready"
	statusColor := "#00FF00" // Green
	if m.streaming {
		status = "Streaming..."
		statusColor = "#FFAA00" // Orange
	}
	if m.err != nil {
		status = "Error"
		statusColor = "#FF0000" // Red
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(statusColor))

	profileStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#5C4D7B"))

	contextStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#4A90E2"))

	header := headerStyle.Render(" ✨ TermAI ") + " " +
		profileStyle.Render(" "+m.profile.Name+" ") + " " +
		profileStyle.Render(" "+m.profile.Model+" ")

	// Add context information if present
	if m.contextDirPath != "" && len(m.contextFiles) > 0 {
		contextInfo := fmt.Sprintf(" 📁 %s (%d files) ", filepath.Base(m.contextDirPath), len(m.contextFiles))
		header += " " + contextStyle.Render(contextInfo)
	}

	header += " " + statusStyle.Render(" ● "+status+" ")

	return header
}

func (m chatModel) renderInputArea() string {
	var sb strings.Builder

	// Show attached files if any
	if len(m.attachedFiles) > 0 {
		attachStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Background(lipgloss.Color("#1A1A1A"))

		fileNames := make([]string, 0, len(m.attachedFiles))
		for _, file := range m.attachedFiles {
			fileType := ""
			switch file.Type {
			case "image":
				fileType = "📷"
			case "pdf":
				fileType = "📄"
			case "code":
				fileType = "💻"
			case "text":
				fileType = "📝"
			}
			fileNames = append(fileNames, fmt.Sprintf("%s %s", fileType, file.Name))
		}

		attachInfo := fmt.Sprintf("📎 Attached: %s", strings.Join(fileNames, ", "))
		sb.WriteString(attachStyle.Render(attachInfo))
		sb.WriteString("\n")
	}

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00D9FF")).
		Bold(true)

	sb.WriteString(promptStyle.Render("You ▸ "))
	sb.WriteString(m.textarea.View())

	return sb.String()
}

func (m chatModel) renderSuggestions() string {
	suggestionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Background(lipgloss.Color("#1A1A1A"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true)

	var parts []string
	for i, suggestion := range m.suggestions {
		if i == m.selectedSuggestion {
			parts = append(parts, selectedStyle.Render(" "+suggestion+" "))
		} else {
			parts = append(parts, suggestionStyle.Render(" "+suggestion+" "))
		}
	}

	return suggestionStyle.Render(" ▸ ") + strings.Join(parts, " ")
}

func (m chatModel) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Background(lipgloss.Color("#1A1A1A"))

	hints := footerStyle.Render(" /help /exit /clear /profile ")
	shortcuts := footerStyle.Render(" Alt+Enter or Ctrl+Enter to send | Enter for new line | Ctrl+C=quit ")

	// Calculate spacing
	totalWidth := m.viewport.Width
	hintsWidth := lipgloss.Width(hints)
	shortcutsWidth := lipgloss.Width(shortcuts)
	spacing := max(totalWidth-hintsWidth-shortcutsWidth, 0)

	return hints + strings.Repeat(" ", spacing) + shortcuts
}

func (m *chatModel) updateViewport() {
	content := strings.Join(m.messages, "\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *chatModel) updateSuggestions() {
	input := m.textarea.Value()

	// Only show suggestions if input starts with "/"
	if !strings.HasPrefix(input, "/") {
		m.showSuggestions = false
		m.suggestions = nil
		m.selectedSuggestion = 0
		return
	}

	// Filter commands that match the input
	var matches []string
	for _, cmd := range chatCommands {
		if strings.HasPrefix(cmd, input) {
			matches = append(matches, cmd)
		}
	}

	// Update suggestions
	if len(matches) > 0 {
		m.showSuggestions = true
		m.suggestions = matches
		// Keep selected suggestion in bounds
		if m.selectedSuggestion >= len(matches) {
			m.selectedSuggestion = 0
		}
	} else {
		m.showSuggestions = false
		m.suggestions = nil
		m.selectedSuggestion = 0
	}
}

func (m chatModel) handleCommand(cmd string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return m, nil
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "/exit", "/quit":
		return m, tea.Quit
	case "/clear":
		m.ctxManager.Clear()
		m.messages = append(m.messages, ui.FormatSuccess("Conversation context cleared"))
		m.messages = append(m.messages, "")
	case "/profile":
		info := fmt.Sprintf("Current Profile: %s\n", m.profile.Name)
		info += fmt.Sprintf("Provider: %s\n", m.profile.Provider)
		info += fmt.Sprintf("Model: %s\n", m.profile.Model)
		info += fmt.Sprintf("Endpoint: %s\n", m.profile.Endpoint)
		info += fmt.Sprintf("Temperature: %.2f\n", m.profile.Temperature)
		info += fmt.Sprintf("Max Tokens: %d\n", m.profile.MaxTokens)
		m.messages = append(m.messages, ui.InfoStyle.Render(info))
		m.messages = append(m.messages, "")
	case "/attach":
		if len(args) == 0 {
			m.messages = append(m.messages, ui.FormatError(fmt.Errorf("/attach requires at least one file path")))
			m.messages = append(m.messages, "")
		} else {
			// Process files
			attachments, err := fileprocessor.ProcessFiles(args)
			if err != nil {
				m.messages = append(m.messages, ui.FormatError(err))
			} else {
				m.attachedFiles = append(m.attachedFiles, attachments...)
				m.messages = append(m.messages, ui.FormatSuccess(fmt.Sprintf("Attached %d file(s)", len(attachments))))
			}
			m.messages = append(m.messages, "")
		}
	case "/files":
		if len(m.attachedFiles) == 0 {
			m.messages = append(m.messages, ui.InfoStyle.Render("No files currently attached"))
		} else {
			info := fmt.Sprintf("Attached files (%d):\n", len(m.attachedFiles))
			for _, file := range m.attachedFiles {
				info += fmt.Sprintf("  • %s (%s)\n", file.Name, file.Type)
			}
			m.messages = append(m.messages, ui.InfoStyle.Render(info))
		}
		m.messages = append(m.messages, "")
	case "/clear-files":
		count := len(m.attachedFiles)
		m.attachedFiles = nil
		m.messages = append(m.messages, ui.FormatSuccess(fmt.Sprintf("Cleared %d attached file(s)", count)))
		m.messages = append(m.messages, "")
	case "/context":
		if len(m.contextFiles) == 0 {
			m.messages = append(m.messages, ui.InfoStyle.Render("No context files loaded"))
		} else {
			info := fmt.Sprintf("Context: %s (%d files)\n", m.contextDirPath, len(m.contextFiles))
			info += "Files:\n"
			for _, file := range m.contextFiles {
				info += fmt.Sprintf("  • %s (%s)\n", file.Name, file.Type)
			}
			m.messages = append(m.messages, ui.InfoStyle.Render(info))
		}
		m.messages = append(m.messages, "")
	case "/context-add":
		if len(args) == 0 {
			m.messages = append(m.messages, ui.FormatError(fmt.Errorf("/context-add requires at least one file path")))
			m.messages = append(m.messages, "")
		} else {
			// Process files and add to context
			attachments, err := fileprocessor.ProcessFiles(args)
			if err != nil {
				m.messages = append(m.messages, ui.FormatError(err))
			} else {
				m.contextFiles = append(m.contextFiles, attachments...)
				m.messages = append(m.messages, ui.FormatSuccess(fmt.Sprintf("Added %d file(s) to context", len(attachments))))
			}
			m.messages = append(m.messages, "")
		}
	case "/context-remove":
		if len(args) == 0 {
			m.messages = append(m.messages, ui.FormatError(fmt.Errorf("/context-remove requires a filename")))
			m.messages = append(m.messages, "")
		} else {
			filename := strings.Join(args, " ")
			removed := false
			for i, file := range m.contextFiles {
				if file.Name == filename || file.Path == filename {
					m.contextFiles = append(m.contextFiles[:i], m.contextFiles[i+1:]...)
					removed = true
					break
				}
			}
			if removed {
				m.messages = append(m.messages, ui.FormatSuccess(fmt.Sprintf("Removed '%s' from context", filename)))
			} else {
				m.messages = append(m.messages, ui.FormatError(fmt.Errorf("file '%s' not found in context", filename)))
			}
			m.messages = append(m.messages, "")
		}

	case "/save":
		// If the chatPath is already set (an no name/path is provided), use it as default
		if m.chatPath != "" && len(args) == 0 {
			// Save conversation to existing path
			if err := m.ctxManager.Save(m.chatPath); err != nil {
				m.messages = append(m.messages, ui.FormatError(fmt.Errorf("failed to save conversation: %v", err)))
			} else {
				m.messages = append(m.messages, ui.FormatSuccess(fmt.Sprintf("Conversation saved successfully to '%s'", m.chatPath)))
			}
			m.messages = append(m.messages, "")
			break
		}

		// Otherwise, expect a name and optional directory
		if len(args) == 0 {
			m.messages = append(m.messages, ui.FormatError(fmt.Errorf("/save requires a conversation name")))
			m.messages = append(m.messages, "")
		} else {
			name := args[0]
			dir, err := config.GetDefaultConversationsPath()
			if err != nil {
				m.messages = append(m.messages, ui.FormatError(fmt.Errorf("failed to get default conversations path: %w", err)))
				m.messages = append(m.messages, "")
				return m, nil
			}
			if len(args) > 2 && args[1] == "-d" {
				if absDir, err := utils.GetAbsolutePath(args[2]); err != nil {
					m.messages = append(m.messages, ui.FormatError(fmt.Errorf("invalid directory path: %w", err)))
					m.messages = append(m.messages, "")
					return m, nil
				} else {
					dir = absDir
				}
			}

			// Save conversation
			if err := m.ctxManager.Save(filepath.Join(dir, name)); err != nil {
				m.messages = append(m.messages, ui.FormatError(fmt.Errorf("failed to save conversation: %v", err)))
			} else {
				m.messages = append(m.messages, ui.FormatSuccess(fmt.Sprintf("Conversation '%s' saved successfully", name)))
			}
			m.messages = append(m.messages, "")
		}
	case "/load":
		if len(args) == 0 {
			m.messages = append(m.messages, ui.FormatError(fmt.Errorf("/load requires a conversation file path")))
			m.messages = append(m.messages, "")
		} else {
			path := args[0] + config.ConversationFileExt

			// Path can be only a filename - look in default conversations dir
			if !filepath.IsAbs(path) && !strings.Contains(path, string(os.PathSeparator)) {
				defaultDir, err := config.GetDefaultConversationsPath()
				if err != nil {
					m.messages = append(m.messages, ui.FormatError(fmt.Errorf("failed to get default conversations path: %w", err)))
					m.messages = append(m.messages, "")
					return m, nil
				}
				path = filepath.Join(defaultDir, path)
			}

			// Load conversation
			if absPath, err := utils.GetAbsolutePath(path); err != nil {
				m.messages = append(m.messages, ui.FormatError(fmt.Errorf("invalid file path: %w", err)))
				m.messages = append(m.messages, "")
			} else {
				err := m.loadConversation(absPath)
				if err != nil {
					m.messages = append(m.messages, ui.FormatError(fmt.Errorf("failed to load conversation: %v", err)))
					m.messages = append(m.messages, "")
				} else {
					// Successfully loaded, set chatPath
					m.chatPath = absPath
				}
				// Success message added in loadConversation
			}
		}
	case "/help":
		m.messages = append(m.messages, ui.InfoStyle.Render(availableCommands))
		m.messages = append(m.messages, "")
	default:
		m.messages = append(m.messages, ui.FormatError(fmt.Errorf("unknown command: %s", command)))
		m.messages = append(m.messages, "")
	}
	m.textarea.Reset()
	m.updateViewport()
	return m, nil
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
