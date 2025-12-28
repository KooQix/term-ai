package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/KooQix/term-ai/internal/config"
	"github.com/KooQix/term-ai/internal/ui"
	"github.com/spf13/cobra"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Manage AI provider profiles",
	Long:  `Manage AI provider profiles for different models and configurations.`,
}

var profilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE:  runProfilesList,
}

var profilesAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new profile",
	RunE:  runProfilesAdd,
}

var profilesRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfilesRemove,
}

var profilesSetDefaultCmd = &cobra.Command{
	Use:   "set-default [name]",
	Short: "Set default profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfilesSetDefault,
}

var profilesShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show profile details",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfilesShow,
}

func init() {
	profilesCmd.AddCommand(profilesListCmd)
	profilesCmd.AddCommand(profilesAddCmd)
	profilesCmd.AddCommand(profilesRemoveCmd)
	profilesCmd.AddCommand(profilesSetDefaultCmd)
	profilesCmd.AddCommand(profilesShowCmd)
}

func runProfilesList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		fmt.Println(ui.InfoStyle.Render("No profiles configured"))
		return nil
	}

	fmt.Println(ui.InfoStyle.Render("Available Profiles:"))
	fmt.Println(ui.FormatSeparator())

	for _, profile := range cfg.Profiles {
		isDefault := ""
		if profile.Name == cfg.DefaultProfile {
			isDefault = ui.SuccessStyle.Render(" (default)")
		}
		fmt.Printf("%s%s\n", ui.AssistantStyle.Render(profile.Name), isDefault)
		fmt.Printf("  Provider: %s\n", profile.Provider)
		fmt.Printf("  Model: %s\n", profile.Model)
		fmt.Printf("  Endpoint: %s\n", profile.Endpoint)
		fmt.Println()
	}

	return nil
}

func runProfilesAdd(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(ui.InfoStyle.Render("Add New Profile"))
	fmt.Println(ui.FormatSeparator())

	// Get profile details
	fmt.Print("Profile name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Provider (openai/claude/abacus/ollama/custom): ")
	provider, _ := reader.ReadString('\n')
	provider = strings.TrimSpace(provider)

	fmt.Print("API Endpoint: ")
	endpoint, _ := reader.ReadString('\n')
	endpoint = strings.TrimSpace(endpoint)

	fmt.Print("API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	fmt.Print("Model name: ")
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)

	fmt.Print("Temperature (0.0-1.0, default 0.7): ")
	tempStr, _ := reader.ReadString('\n')
	tempStr = strings.TrimSpace(tempStr)
	temperature := 0.7
	if tempStr != "" {
		if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
			temperature = temp
		}
	}

	fmt.Print("Max tokens (default 2000): ")
	maxTokensStr, _ := reader.ReadString('\n')
	maxTokensStr = strings.TrimSpace(maxTokensStr)
	maxTokens := 2000
	if maxTokensStr != "" {
		if tokens, err := strconv.Atoi(maxTokensStr); err == nil {
			maxTokens = tokens
		}
	}

	// Load config and add profile
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	profile := config.Profile{
		Name:        name,
		Provider:    provider,
		Endpoint:    endpoint,
		APIKey:      apiKey,
		Model:       model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	if err := cfg.AddProfile(profile); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.FormatSuccess(fmt.Sprintf("Profile '%s' added successfully", name)))
	return nil
}

func runProfilesRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.RemoveProfile(name); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(ui.FormatSuccess(fmt.Sprintf("Profile '%s' removed", name)))
	return nil
}

func runProfilesSetDefault(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if profile exists
	if _, err := cfg.GetProfile(name); err != nil {
		return err
	}

	cfg.DefaultProfile = name

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(ui.FormatSuccess(fmt.Sprintf("Default profile set to '%s'", name)))
	return nil
}

func runProfilesShow(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	profile, err := cfg.GetProfile(name)
	if err != nil {
		return err
	}

	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Profile: %s", profile.Name)))
	fmt.Println(ui.FormatSeparator())
	fmt.Printf("Provider:    %s\n", profile.Provider)
	fmt.Printf("Model:       %s\n", profile.Model)
	fmt.Printf("Endpoint:    %s\n", profile.Endpoint)
	fmt.Printf("Temperature: %.1f\n", profile.Temperature)
	fmt.Printf("Max Tokens:  %d\n", profile.MaxTokens)
	if profile.TopP > 0 {
		fmt.Printf("Top P:       %.2f\n", profile.TopP)
	}

	return nil
}
