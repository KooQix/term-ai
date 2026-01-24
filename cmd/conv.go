package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KooQix/term-ai/internal/config"
	"github.com/spf13/cobra"
)

var convCmd = &cobra.Command{
	Use:   "conv",
	Short: "Manage conversations",
	Long: `Manage your saved conversations with TermAI.

Commands:
  list        List all saved conversations
  delete      Delete a specific conversation
  export      Export a conversation to a readable format

Examples:
  termai conv list
  termai conv list <project_name>
  termai conv delete <conversation_id>
  termai conv export <conversation_id> -o output.txt`,
}

func init() {
	convCmd.AddCommand(convListCmd)
	convCmd.AddCommand(convDeleteCmd)
	convCmd.AddCommand(convExportCmd)

	convExportCmd.Flags().StringP("output", "o", "", "Output file path")
}

var convListCmd = &cobra.Command{
	Use:   "list [project_name]",
	Short: "List all saved conversations",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runConvList,
}

var convDeleteCmd = &cobra.Command{
	Use:   "delete [conversation_id]",
	Short: "Delete a specific conversation",
	Args:  cobra.ExactArgs(1),
	RunE:  runConvDelete,
}

var convExportCmd = &cobra.Command{
	Use:   "export [conversation_id]",
	Short: "Export a conversation to a readable format",
	Args:  cobra.ExactArgs(1),
	RunE:  runConvExport,
}

func runConvList(cmd *cobra.Command, args []string) error {
	convPath, err := config.GetDefaultConversationsPath()
	if err != nil {
		return err
	}

	// If project name is provided, list contents of that project
	if len(args) > 0 {
		projectPath := filepath.Join(convPath, args[0])
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

		fmt.Printf("Conversations in project '%s':\n\n", args[0])
		return listProjectContents(projectPath, "")
	}

	// List top-level conversations and folders
	entries, err := os.ReadDir(convPath)
	if err != nil {
		return fmt.Errorf("failed to read conversations directory: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No conversations found.")
		return nil
	}

	fmt.Println("Conversations:\n")
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

func runConvDelete(cmd *cobra.Command, args []string) error {
	convPath, err := config.GetDefaultConversationsPath()
	if err != nil {
		return err
	}

	conversationID := args[0]

	// Try to find the conversation (could be a file or in a subdirectory)
	targetPath, err := findConversation(convPath, conversationID)
	if err != nil {
		return err
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete '%s'? (y/N): ", config.GetDisplayPath(conversationID))
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Deletion cancelled.")
		return nil
	}

	if err := os.RemoveAll(targetPath); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	fmt.Printf("Conversation '%s' deleted successfully.\n", config.GetDisplayPath(conversationID))
	return nil
}

func runConvExport(cmd *cobra.Command, args []string) error {
	convPath, err := config.GetDefaultConversationsPath()
	if err != nil {
		return err
	}

	conversationID := args[0]
	outputPath, _ := cmd.Flags().GetString("output")

	// Find the conversation file
	targetPath, err := findConversation(convPath, conversationID)
	if err != nil {
		return err
	}

	// Read the conversation file
	content, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read conversation: %w", err)
	}

	// If no output path specified, print to stdout
	if outputPath == "" {
		fmt.Println(string(content))
		return nil
	}

	// Write to output file
	if err := os.WriteFile(outputPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write to output file: %w", err)
	}

	fmt.Printf("Conversation exported to '%s'\n", outputPath)
	return nil
}

// findConversation searches for a conversation file in the conversations directory
// It checks both the root level and subdirectories
func findConversation(convPath, conversationID string) (string, error) {
	// First, try direct path
	directPath := filepath.Join(convPath, conversationID)
	if _, err := os.Stat(directPath); err == nil {
		return directPath, nil
	}

	// Try with common extensions if not found
	extensions := []string{".json", ".txt", ".md"}
	for _, ext := range extensions {
		if !strings.HasSuffix(conversationID, ext) {
			testPath := filepath.Join(convPath, conversationID+ext)
			if _, err := os.Stat(testPath); err == nil {
				return testPath, nil
			}
		}
	}

	// Search in subdirectories
	var foundPath string
	err := filepath.Walk(convPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			baseName := filepath.Base(path)
			displayName := config.GetDisplayPath(baseName)

			if baseName == conversationID || displayName == conversationID {
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
		return "", fmt.Errorf("conversation '%s' not found", conversationID)
	}

	return foundPath, nil
}
