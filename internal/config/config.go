package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Profile struct {
	Name        string  `yaml:"name"`
	Provider    string  `yaml:"provider"`
	Endpoint    string  `yaml:"endpoint"`
	APIKey      string  `yaml:"api_key" sensitive:"true"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
	TopP        float64 `yaml:"top_p,omitempty"`

	SystemContext *string `yaml:"system_context"` // nil means use global system context || empty string means no system context
}

type UIConfig struct {
	Theme        string `yaml:"theme"`         // Theme name: dracula, monokai, github, solarized-dark, solarized-light, auto
	ColorOutput  bool   `yaml:"color_output"`  // Enable/disable colored output
	ShowThinking bool   `yaml:"show_thinking"` // Show thinking/reasoning in output
}

type FileConfig struct {
	MaxFileSize              int64 `yaml:"max_file_size"`                // Maximum file size in bytes (default: 10MB)
	AutoClearAfterSend       bool  `yaml:"auto_clear_after_send"`        // Clear attached files after sending message
	IncludeContextInEveryMsg bool  `yaml:"include_context_in_every_msg"` // Include context files in every message
}

type ToolsConfig struct {
	MaxIter int                       `yaml:"max_iter"`     // Default max iterations for tools that support it (can be overridden by specific tool config)
	Config  map[string]map[string]any `yaml:"tool_configs"` // Tool-specific configurations, keyed by tool name (e.g. "web_search": {"api_key
}

type Config struct {
	DefaultProfile string     `yaml:"default_profile"`
	Profiles       []Profile  `yaml:"profiles"`
	UI             UIConfig   `yaml:"ui,omitempty"`
	Files          FileConfig `yaml:"files,omitempty"`
	SystemContext  string     `yaml:"system_context,omitempty"`

	ToolConfigs ToolsConfig `yaml:"tool_configs"`
}

const (
	ConfigDirName  = ".termai"
	ConfigFileName = "config.yaml"

	ConversationsDirectory = "conversations"
	ChatFileExt            = ".termai.md"
)

var AppConfig *Config

func init() {
	if _, err := Load(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
	}
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ConfigDirName, ConfigFileName), nil
}

func GetDefaultChatsPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	conversationsPath := filepath.Join(configDir, ConversationsDirectory)

	// If conversations directory doesn't exist, create it
	if _, err := os.Stat(conversationsPath); os.IsNotExist(err) {
		if err := os.MkdirAll(conversationsPath, 0o700); err != nil {
			return "", fmt.Errorf("failed to create conversations directory: %w", err)
		}
	}

	return conversationsPath, nil
}

// GetConfigDir returns the path to the config directory
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ConfigDirName), nil
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(configDir, 0o700)
}

// Load reads and parses the config file
func Load() (*Config, error) {
	if AppConfig != nil {
		return AppConfig, nil
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		if err := createDefaultConfig(); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := getDefaultConfig() // start with default config

	// Unmarshal user config on top of default config to fill in any missing fields with defaults
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	AppConfig = cfg

	return cfg, nil
}

// Save writes the config to file
func (c *Config) Save() error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetProfile returns a profile by name
func (c *Config) GetProfile(name string) (*Profile, error) {
	for _, p := range c.Profiles {
		if p.Name == name {

			// Load the correct context if needed
			if p.SystemContext == nil {
				// Use global system context — copy the value so a caller mutating
				// *profile.SystemContext can't silently write through to the global
				// config struct.
				sysCtx := c.SystemContext
				p.SystemContext = &sysCtx
			} // else, use the profile-specific context (even if it's an empty string)

			return &p, nil
		}
	}
	return nil, fmt.Errorf("profile '%s' not found", name)
}

// GetDefaultProfile returns the default profile
func (c *Config) GetDefaultProfile() (*Profile, error) {
	if c.DefaultProfile == "" && len(c.Profiles) > 0 {
		return &c.Profiles[0], nil
	}
	return c.GetProfile(c.DefaultProfile)
}

// AddProfile adds a new profile
func (c *Config) AddProfile(profile Profile) error {
	// Check if profile already exists
	for _, p := range c.Profiles {
		if p.Name == profile.Name {
			return fmt.Errorf("profile '%s' already exists", profile.Name)
		}
	}
	c.Profiles = append(c.Profiles, profile)
	return nil
}

// RemoveProfile removes a profile by name
func (c *Config) RemoveProfile(name string) error {
	for i, p := range c.Profiles {
		if p.Name == name {
			c.Profiles = append(c.Profiles[:i], c.Profiles[i+1:]...)
			if c.DefaultProfile == name {
				if len(c.Profiles) > 0 {
					c.DefaultProfile = c.Profiles[0].Name
				} else {
					c.DefaultProfile = ""
				}
			}
			return nil
		}
	}
	return fmt.Errorf("profile '%s' not found", name)
}

func GetDisplayPath(originalPath string) string {
	if originalPath == "" {
		return ""
	}

	originalPath = strings.TrimSuffix(originalPath, ChatFileExt)
	return filepath.Base(originalPath)
}

func getDefaultConfig() *Config {
	return &Config{
		DefaultProfile: "abacus",
		Profiles: []Profile{
			{
				Name:          "abacus",
				Provider:      "abacus",
				Endpoint:      "https://api.abacus.ai/v1",
				APIKey:        "your-abacus-api-key",
				Model:         "gpt-4",
				Temperature:   0.7,
				MaxTokens:     2000,
				SystemContext: nil,
			},
			{
				Name:          "openai",
				Provider:      "openai",
				Endpoint:      "https://api.openai.com/v1",
				APIKey:        "your-openai-api-key",
				Model:         "gpt-4",
				Temperature:   0.7,
				MaxTokens:     2000,
				SystemContext: nil,
			},
			{
				Name:          "ollama",
				Provider:      "ollama",
				Endpoint:      "http://localhost:11434/v1",
				APIKey:        "ollama",
				Model:         "llama3.1",
				Temperature:   0.7,
				MaxTokens:     2000,
				SystemContext: nil,
			},
		},
		UI: UIConfig{
			Theme:        "auto",
			ColorOutput:  true,
			ShowThinking: true,
		},
		Files: FileConfig{
			MaxFileSize:              10485760, // 10MB
			AutoClearAfterSend:       true,
			IncludeContextInEveryMsg: false,
		},
		ToolConfigs: ToolsConfig{
			MaxIter: 8,
			Config:  make(map[string]map[string]any),
		},

		SystemContext: "You are an AI CLI assistant for a software developer. Provide clear, concise, and actionable responses focused on commands, debugging (including extended sessions), and project development tasks. Use markdown formatting for code and commands. Consider the current project context and previous interactions when necessary. Ask clarifying questions if the request is ambiguous. Avoid unnecessary explanations unless explicitly requested. Handle errors gracefully and suggest best practices or alternatives when appropriate.",
	}
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig() error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	defaultConfig := getDefaultConfig()

	return defaultConfig.Save()
}

// RegisterToolConfig registers or updates configuration for a specific tool
func RegisterToolConfig(toolName string, config any) error {
	if AppConfig == nil {
		panic("config not loaded") // this should never happen since AppConfig is initialized in init()
	}

	if AppConfig.ToolConfigs.Config == nil {
		AppConfig.ToolConfigs.Config = make(map[string]map[string]any)
	}

	if _, ok := AppConfig.ToolConfigs.Config[toolName]; ok {
		// No need to continue, the config has already been registered
		return nil
	}

	b, err := yaml.Marshal(config)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal default config for %s: %v", toolName, err))
	}
	var cf map[string]any
	if err := yaml.Unmarshal(b, &cf); err != nil {
		panic(fmt.Sprintf("failed to unmarshal default config for %s: %v", toolName, err))
	}

	AppConfig.ToolConfigs.Config[toolName] = cf
	return AppConfig.Save()
}

func ParseToolConfig(toolName string, config any) error {
	if AppConfig == nil {
		return fmt.Errorf("config not loaded")
	}

	cf, ok := AppConfig.ToolConfigs.Config[toolName]
	if !ok {
		return fmt.Errorf("no config found for tool '%s'", toolName)
	}

	b, err := yaml.Marshal(cf)
	if err != nil {
		return fmt.Errorf("failed to marshal config for tool '%s': %w", toolName, err)
	}

	if err := yaml.Unmarshal(b, config); err != nil {
		return fmt.Errorf("failed to unmarshal config for tool '%s': %w", toolName, err)
	}

	return nil
}
