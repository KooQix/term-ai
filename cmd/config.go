package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"

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
	data, err := MarshalYAMLRedacted(*cfg)
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

// Helper functions for redacting sensitive fields in config display
const sensitiveMask = "**"

var sensitiveFields = []string{"api_key", "apikey", "secret"}

func isSensitiveField(fieldName string) bool {
	fieldName = strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldName, sensitive) {
			return true
		}
	}
	return false
}

// MarshalYAMLRedacted marshals v to YAML, replacing fields tagged
// `sensitive:"true"` with "**". Original value is untouched.
func MarshalYAMLRedacted(v any) ([]byte, error) {
	redacted := redactValue("", reflect.ValueOf(v))
	return yaml.Marshal(redacted.Interface())
}

func redactValue(key string, v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return v
		}
		inner := redactValue(key, v.Elem())
		// Wrap back into a pointer if needed
		if v.Kind() == reflect.Ptr {
			ptr := reflect.New(inner.Type())
			ptr.Elem().Set(inner)
			return ptr
		}
		return inner

	case reflect.Struct:
		t := v.Type()
		// Build a new struct value we can write to
		out := reflect.New(t).Elem()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			fv := v.Field(i)

			if field.Tag.Get("sensitive") == "true" || isSensitiveField(field.Name) {
				// Replace with mask if field is a string; otherwise zero + mask via string
				if fv.Kind() == reflect.String {
					out.Field(i).SetString(sensitiveMask)
				} else {
					// For non-strings, set zero value (or handle differently)
					// Alternative: marshal to a map for mixed types.
					out.Field(i).Set(reflect.Zero(fv.Type()))
				}
				continue
			}
			out.Field(i).Set(redactValue(field.Name, fv))
		}
		return out

	case reflect.Slice, reflect.Array:
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(redactValue("", v.Index(i)))
		}
		return out
	case reflect.Map:
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key()
			out.SetMapIndex(key, redactValue(key.String(), iter.Value()))
		}
		return out

	default:

		if key != "" && isSensitiveField(key) {
			if v.Kind() == reflect.String {
				return reflect.ValueOf(sensitiveMask)
			}
			// For non-string types, we could choose to zero them out or handle differently
			return reflect.Zero(v.Type())
		}
		return v
	}
}
