package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/KooQix/term-ai/internal/config"
	"github.com/KooQix/term-ai/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage TermAI configuration file and settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	RunE:  runConfigPath,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration in default editor",
	RunE:  runConfigEdit,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configEditCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Copy the config to replace all API Keys with ****
	maskedCfg := *cfg
	for i, profile := range maskedCfg.Profiles {
		if profile.APIKey != "" {
			maskedCfg.Profiles[i].APIKey = "****"
		}
	}

	data, err := yaml.Marshal(maskedCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Println(ui.InfoStyle.Render("Current Configuration:"))
	fmt.Println(ui.FormatSeparator())
	fmt.Println(string(data))
	fmt.Println(ui.FormatSeparator())

	configPath, _ := config.GetConfigPath()
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Config file: %s", configPath)))

	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	configPath, err := config.GetConfigPath()
	if err != nil {
		return err
	}
	fmt.Println(configPath)
	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	// Ensure config exists
	if _, err := config.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	configPath, err := config.GetConfigPath()
	if err != nil {
		return err
	}

	// Get editor from environment or use default
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // fallback to vi
	}

	// Open editor
	cmd2 := exec.Command(editor, configPath)
	cmd2.Stdin = os.Stdin
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr

	if err := cmd2.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	fmt.Println(ui.FormatSuccess("Configuration updated"))
	return nil
}
