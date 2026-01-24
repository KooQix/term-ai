package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/KooQix/term-ai/internal/chat"
	"github.com/KooQix/term-ai/internal/config"
	"github.com/KooQix/term-ai/internal/fileprocessor"
	"github.com/KooQix/term-ai/internal/provider"
	"github.com/KooQix/term-ai/internal/ui"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
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
  /cp   - Copy the last assistant response to clipboard
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
	chatCommands = []string{"/help", "/exit", "/quit", "/clear", "/profile", "/attach", "/files", "/clear-files", "/context", "/context-add", "/context-remove", "/save", "/load", "/cp"}
)

func init() {
	chatCmd.Flags().StringArrayVarP(&chatFilePaths, "file", "f", []string{}, "File(s) to attach (can be used multiple times)")
	chatCmd.Flags().StringVarP(&contextDir, "dir", "d", "", "Directory to use as context (scans for supported files)")
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
	ta.CharLimit = 0 // No limit
	ta.SetWidth(80)
	ta.SetHeight(3)

	vp := viewport.New(150, 80)

	m := chat.NewChatModel(cfg, ta, vp, prov, profile, chat.ChatCommands{
		Commands:  chatCommands,
		Available: availableCommands,
	})

	// Add welcome message
	welcome := fmt.Sprintf("Context: %s\n\n", wordwrap.String(*profile.SystemContext, vp.Width-2))
	welcome += "Type /help to see available commands.\n"

	// Process initial files if provided
	if len(chatFilePaths) > 0 {
		fmt.Print("Processing initial files... ")
		attachments, err := fileprocessor.ProcessFiles(chatFilePaths)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			m.AttachFiles(attachments)
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
			m.AddContextFiles(contextFiles)
			m.SetContextDir(contextDir)
			fmt.Printf("✓ %d file(s) in context\n", len(contextFiles))
			welcome += fmt.Sprintf("📁 Context: %s (%d files)\n", contextDir, len(contextFiles))
		}
	}

	welcome += "\n" + ui.FormatSeparator() + "\n"
	m.AddMessage(welcome)

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
