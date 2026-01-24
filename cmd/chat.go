package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	chatPath      string

	chatCmd = &cobra.Command{
		Use:   "chat",
		Short: "Start interactive chat session",
		Long:  `Start an interactive chat session with conversation context.`,
		RunE:  runChat,
	}

	// Available chat commands for auto-completion
	chatCommands = []string{"/help", "/exit", "/quit", "/clear", "/profile", "/attach", "/files", "/clear-files", "/context", "/context-add", "/context-remove", "/save", "/load", "/cp"}
)

var chatListCmd = &cobra.Command{
	Use:   "list [project_name]",
	Short: "List all saved chats",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runChatList,
}

var chatDeleteCmd = &cobra.Command{
	Use:   "delete [chat_id]",
	Short: "Delete a specific chat",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatDelete,
}

func init() {
	chatCmd.Flags().StringArrayVarP(&chatFilePaths, "file", "f", []string{}, "File(s) to attach (can be used multiple times)")
	chatCmd.Flags().StringVarP(&contextDir, "dir", "d", "", "Directory to use as context (scans for supported files)")
	chatCmd.Flags().StringVarP(&chatPath, "load-chat", "c", "", "Load a saved chat conversation from file")

	chatCmd.AddCommand(chatListCmd)
	chatCmd.AddCommand(chatDeleteCmd)
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
	welcome := "Type /help to see available commands.\n\n"

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
		contextFiles, err := fileprocessor.ScanDirectory(contextDir)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			m.AddContextFiles(contextFiles)
			m.SetContextDir(contextDir)
			fmt.Printf("✓ %d file(s) in context\n", len(contextFiles))
			welcome += fmt.Sprintf("📁 Context: %s (%d files)\n", contextDir, len(contextFiles))
		}
	}

	welcome += ui.FormatSeparator()
	m.AddMessage(welcome)

	// Check if loading a chat (and will load its context as well)
	if chatPath != "" {
		if err := m.LoadChatHandler(chatPath); err != nil {
			return fmt.Errorf("failed to load conversation: %w", err)
		}
	} else {
		// If not loading a chat and starting fresh, then load the context from profile if any
		if m.Profile.SystemContext != nil && *m.Profile.SystemContext != "" {
			m.AddMessage(ui.FormatSystemMessage(wordwrap.String(*m.Profile.SystemContext, vp.Width-5)))
		}
	}

	// Start Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

//////////////////// Chat CLI commands \\\\\\\\\\\\\\\\\\\\

func runChatList(cmd *cobra.Command, args []string) error {
	chatPath, err := config.GetDefaultChatsPath()
	if err != nil {
		return err
	}

	// If project name is provided, list contents of that project
	if len(args) > 0 {
		projectPath := filepath.Join(chatPath, args[0])
		if _, err := os.Stat(projectPath); os.IsNotExist(err) {
			return fmt.Errorf("project '%s' not found", args[0])
		}

		info, err := os.Stat(projectPath)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return fmt.Errorf("'%s' is not a project/folder", args[0])
		}

		fmt.Printf("Chats in project '%s':\n\n", args[0])
		return listProjectContents(projectPath, "")
	}

	// List top-level chats and folders
	entries, err := os.ReadDir(chatPath)
	if err != nil {
		return fmt.Errorf("failed to read chats directory: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No chats found.")
		return nil
	}

	fmt.Println("Chats:")
	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Printf("  📁 %s/ (project)\n", entry.Name())
		} else {
			displayName := config.GetDisplayPath(entry.Name())
			fmt.Printf("  📄 %s\n", displayName)
		}
	}

	return nil
}

func listProjectContents(projectPath string, indent string) error {
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return fmt.Errorf("failed to read project directory: %w", err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(projectPath, entry.Name())

		if entry.IsDir() {
			fmt.Printf("%s  📁 %s/\n", indent, entry.Name())
			if err := listProjectContents(fullPath, indent+"    "); err != nil {
				return err
			}
		} else {
			displayName := config.GetDisplayPath(entry.Name())
			fmt.Printf("%s  📄 %s\n", indent, displayName)
		}
	}

	return nil
}

func runChatDelete(cmd *cobra.Command, args []string) error {
	chatPath, err := config.GetDefaultChatsPath()
	if err != nil {
		return err
	}

	chatID := args[0]

	// Try to find the conversation (could be a file or in a subdirectory)
	targetPath, err := findChat(chatPath, chatID)
	if err != nil {
		return err
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete '%s'? (y/N): ", config.GetDisplayPath(chatID))
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Deletion cancelled.")
		return nil
	}

	if err := os.RemoveAll(targetPath); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	fmt.Printf("Conversation '%s' deleted successfully.\n", config.GetDisplayPath(chatID))
	return nil
}

// findChat searches for a chat file in the conversations directory
// It checks both the root level and subdirectories
func findChat(chatPath, chatID string) (string, error) {
	// First, try direct path
	directPath := filepath.Join(chatPath, chatID)
	if _, err := os.Stat(directPath); err == nil {
		return directPath, nil
	}

	// Try with common extensions if not found
	extensions := []string{".json", ".txt", ".md"}
	for _, ext := range extensions {
		if !strings.HasSuffix(chatID, ext) {
			testPath := filepath.Join(chatPath, chatID+ext)
			if _, err := os.Stat(testPath); err == nil {
				return testPath, nil
			}
		}
	}

	// Search in subdirectories
	var foundPath string
	err := filepath.Walk(chatPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			baseName := filepath.Base(path)
			displayName := config.GetDisplayPath(baseName)

			if baseName == chatID || displayName == chatID {
				foundPath = path
				return filepath.SkipAll
			}
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error searching for conversation: %w", err)
	}

	if foundPath == "" {
		return "", fmt.Errorf("conversation '%s' not found", chatID)
	}

	return foundPath, nil
}
