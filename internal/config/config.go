package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Profile struct {
	Name        string  `yaml:"name"`
	Provider    string  `yaml:"provider"`
	Endpoint    string  `yaml:"endpoint"`
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
	TopP        float64 `yaml:"top_p,omitempty"`
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

type Config struct {
	DefaultProfile string     `yaml:"default_profile"`
	Profiles       []Profile  `yaml:"profiles"`
	UI             UIConfig   `yaml:"ui,omitempty"`
	Files          FileConfig `yaml:"files,omitempty"`
	SystemContext  string     `yaml:"system_context,omitempty"`
}

const (
	ConfigDirName  = ".termai"
	ConfigFileName = "config.yaml"
)

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ConfigDirName, ConfigFileName), nil
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
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		if err := CreateDefaultConfig(); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
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

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig() error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	defaultConfig := Config{
		DefaultProfile: "abacus",
		Profiles: []Profile{
			{
				Name:        "abacus",
				Provider:    "abacus",
				Endpoint:    "https://api.abacus.ai/v1",
				APIKey:      "your-abacus-api-key",
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxTokens:   2000,
			},
			{
				Name:        "openai",
				Provider:    "openai",
				Endpoint:    "https://api.openai.com/v1",
				APIKey:      "your-openai-api-key",
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxTokens:   2000,
			},
			{
				Name:        "ollama",
				Provider:    "ollama",
				Endpoint:    "http://localhost:11434/v1",
				APIKey:      "ollama",
				Model:       "llama3.1",
				Temperature: 0.7,
				MaxTokens:   2000,
			},
		},
		UI: UIConfig{
			Theme:        "dark",
			ColorOutput:  true,
			ShowThinking: true,
		},
		Files: FileConfig{
			MaxFileSize:              10485760, // 10MB
			AutoClearAfterSend:       true,
			IncludeContextInEveryMsg: false,
		},
		SystemContext: "You are an AI CLI assistant for a software developer. Provide clear, concise, and actionable responses focused on commands, debugging (including extended sessions), and project development tasks. Use markdown formatting for code and commands. Consider the current project context and previous interactions when necessary. Ask clarifying questions if the request is ambiguous. Avoid unnecessary explanations unless explicitly requested. Handle errors gracefully and suggest best practices or alternatives when appropriate.",
	}

	return defaultConfig.Save()
}
