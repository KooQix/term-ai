package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/KooQix/term-ai/internal/config"
	"github.com/KooQix/term-ai/internal/fileprocessor"
	"github.com/KooQix/term-ai/internal/provider"
	"github.com/KooQix/term-ai/internal/ui"
	"github.com/spf13/cobra"
)

var (
	profileName string
	filePaths   []string
	version     = "1.0.0"
)

var rootCmd = &cobra.Command{
	Use:   "termai [prompt]",
	Short: "TermAI - Your AI assistant in the terminal",
	Long: `TermAI is a powerful AI assistant CLI tool that supports multiple AI providers
including OpenAI, Claude, Abacus.AI, Ollama, and any OpenAI-compatible API.

Examples:
  termai "Explain quantum computing"
  termai --profile ollama "What is the capital of France?"
  termai -file doc.pdf -file image.png "Analyze these files"
  termai -f report.md "Summarize this document"
  termai chat
  termai profiles list
  termai config show`,
	Version: version,
	Args:    cobra.MaximumNArgs(1),
	RunE:    runPrompt,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&profileName, "profile", "p", "", "Profile to use")
	rootCmd.PersistentFlags().StringArrayVarP(&filePaths, "file", "f", []string{}, "File(s) to attach (can be used multiple times)")

	// Add subcommands
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(profilesCmd)
	rootCmd.AddCommand(convCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

// runPrompt handles one-line prompt mode
func runPrompt(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	prompt := args[0]

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

	// Setup context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Process files if provided
	var attachments []*fileprocessor.FileAttachment
	if len(filePaths) > 0 {
		ui.ShowSpinner("Processing files")
		var err error
		attachments, err = fileprocessor.ProcessFiles(filePaths)
		ui.ClearSpinner()
		if err != nil {
			return fmt.Errorf("failed to process files: %w", err)
		}
		if len(attachments) > 0 {
			fmt.Printf("✓ Processed %d file(s)\n", len(attachments))
		}
	}

	// Build message content
	messageContent := prompt
	var images []string

	// Add file contents to the message
	for _, attachment := range attachments {
		switch attachment.Type {
		case "image":
			// Add image as base64 data URL
			images = append(images, attachment.Content)
			fmt.Printf("  • Image: %s\n", attachment.Name)
		case "pdf", "text", "code":
			// Append text content to the prompt
			messageContent += fmt.Sprintf("\n\n--- Content from %s ---\n%s\n--- End of %s ---",
				attachment.Name, attachment.Content, attachment.Name)
			fmt.Printf("  • %s: %s\n", strings.Title(attachment.Type), attachment.Name)
		}
	}

	// Create messages
	messages := []provider.Message{
		{
			Role:    "user",
			Content: messageContent,
			Images:  images,
		},
	}

	// Add system context if defined in config
	if cfg.SystemContext != "" {
		systemMessage := provider.Message{
			Role:    "system",
			Content: cfg.SystemContext,
		}
		messages = append([]provider.Message{systemMessage}, messages...)
	}

	// Show header
	fmt.Println()
	fmt.Println(ui.FormatUserMessage(prompt))
	if len(attachments) > 0 {
		fmt.Printf("(with %d attachment(s))\n", len(attachments))
	}
	fmt.Println(ui.FormatSeparator())

	// Assistant header
	fmt.Println(ui.AssistantStyle.Render("Assistant:"))
	fmt.Println()

	// Stream response
	ui.ShowSpinner("Thinking")
	chunkChan, err := prov.Stream(ctx, messages)
	if err != nil {
		ui.ClearSpinner()
		return fmt.Errorf("failed to get response: %w", err)
	}

	// ui.ClearSpinner()

	// Stream raw content in real-time while accumulating
	writer := ui.NewStreamWriter()
	for chunk := range chunkChan {
		if chunk.Error != nil {
			return fmt.Errorf("stream error: %w", chunk.Error)
		}
		if chunk.Thinking != "" {
			// Print and accumulate thinking
			writer.WriteThinking(chunk.Thinking)
		}
		if chunk.Content != "" {
			// Print raw content immediately for real-time feedback
			writer.WriteContent(chunk.Content)
		}
		if chunk.Done {
			break
		}
	}

	// fmt.Println() // Add newline after raw streaming
	ui.ClearSpinner() // Clear spinner after streaming is done
	// Display separator line
	// fmt.Println(ui.FormatSeparator())

	// Now format and display the complete response with syntax highlighting
	rawContent := writer.GetContent()
	formatted, err := ui.FormatResponse(rawContent)
	if err != nil {
		// If formatting fails, raw content already shown above
		fmt.Println()
	} else {
		// Show formatted version
		// TODO: right now we disable the raw stream display to avoid duplication
		// fmt.Println(ui.AssistantStyle.Render("Formatted:"))
		// fmt.Println()
		fmt.Println(formatted)
	}

	fmt.Println()

	return nil
}
