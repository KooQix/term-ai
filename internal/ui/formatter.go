package ui

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/KooQix/term-ai/internal/config"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Styles for terminal output
	UserStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true)

	AssistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	ThinkingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00")).
			Italic(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
)

// ContentFormat represents the detected format of content
type ContentFormat int

const (
	ContentFormatUnknown ContentFormat = iota
	ContentFormatJSON
	ContentFormatYAML
	ContentFormatXML
	ContentFormatMarkdown
	ContentFormatPlainText
)

// detectFormat attempts to detect the format of the content
func detectFormat(content string) ContentFormat {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ContentFormatPlainText
	}

	// Check for JSON
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		// Try to parse as JSON
		var js json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &js); err == nil {
			return ContentFormatJSON
		}
	}

	// Check for YAML indicators
	if strings.HasPrefix(trimmed, "---") ||
		strings.Contains(trimmed, ":\n") ||
		strings.Contains(trimmed, ": ") {
		// Simple YAML detection
		lines := strings.Split(trimmed, "\n")
		yamlLines := 0
		for _, line := range lines {
			if strings.Contains(line, ": ") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
				yamlLines++
			}
		}
		if yamlLines > 2 {
			return ContentFormatYAML
		}
	}

	// Check for XML
	if strings.HasPrefix(trimmed, "<?xml") ||
		(strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") && strings.Count(trimmed, "<") > 2) {
		return ContentFormatXML
	}

	// Check for markdown indicators
	if strings.Contains(trimmed, "```") ||
		strings.Contains(trimmed, "# ") ||
		strings.Contains(trimmed, "## ") ||
		strings.Contains(trimmed, "- ") ||
		strings.Contains(trimmed, "* ") ||
		strings.Contains(trimmed, "[") && strings.Contains(trimmed, "](") {
		return ContentFormatMarkdown
	}

	// Default to plain text if no format is detected
	return ContentFormatPlainText
}

// FormatResponse is the main formatting function with auto-detection
func FormatResponse(content string) (string, error) {
	format := detectFormat(content)

	switch format {
	case ContentFormatJSON:
		return FormatJSON(content)
	case ContentFormatYAML:
		return FormatYAML(content)
	case ContentFormatXML:
		return FormatXML(content)
	case ContentFormatMarkdown:
		return FormatMarkdown(content)
	default:
		// For plain text or unknown, try markdown anyway as it handles plain text well
		formatted, err := FormatMarkdown(content)
		if err != nil {
			return content, nil
		}
		return formatted, nil
	}
}

// FormatJSON formats JSON content with syntax highlighting
func FormatJSON(content string) (string, error) {
	// First, try to parse and pretty-print the JSON
	var parsed interface{}
	trimmed := strings.TrimSpace(content)

	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		// If it's not valid JSON, return as-is
		return content, err
	}

	// Pretty print JSON
	prettyJSON, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return content, err
	}

	// Apply syntax highlighting
	highlighted, err := highlightCode(string(prettyJSON), "json")
	if err != nil {
		return string(prettyJSON), nil
	}

	return highlighted, nil
}

// FormatYAML formats YAML content with syntax highlighting
func FormatYAML(content string) (string, error) {
	highlighted, err := highlightCode(content, "yaml")
	if err != nil {
		return content, nil
	}
	return highlighted, nil
}

// FormatXML formats XML content with syntax highlighting
func FormatXML(content string) (string, error) {
	highlighted, err := highlightCode(content, "xml")
	if err != nil {
		return content, nil
	}
	return highlighted, nil
}

// FormatMarkdown renders markdown content with glamour
func FormatMarkdown(content string) (string, error) {
	// Try dark style first (works best for most terminals)
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		// Fall back to dracula style
		r, err = glamour.NewTermRenderer(
			glamour.WithStylePath("dracula"),
			glamour.WithWordWrap(100),
		)
		if err != nil {
			// Final fallback to auto style
			r, err = glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(100),
			)
			if err != nil {
				return content, err
			}
		}
	}

	out, err := r.Render(content)
	if err != nil {
		return content, err
	}
	return strings.TrimSpace(out), nil
}

// FormatCodeBlock formats a code block with syntax highlighting
func FormatCodeBlock(code, language string) (string, error) {
	return highlightCode(code, language)
}

// highlightCode applies syntax highlighting to code using chroma
func highlightCode(code, language string) (string, error) {
	// Get the lexer for the language
	var lexer chroma.Lexer
	if language != "" {
		lexer = lexers.Get(language)
	}
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Get the style from config
	style := styles.Get(config.AppConfig.UI.Theme)

	if style == nil {
		style = styles.Fallback
	}

	// Get the formatter for terminal with 256 colors
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Tokenize
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, err
	}

	// Format to buffer
	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code, err
	}

	return buf.String(), nil
}

// FormatUserMessage formats a user message
func FormatUserMessage(content string) string {
	return UserStyle.Render("You: ") + content
}

// FormatAssistantMessage formats an assistant message
func FormatAssistantMessage(content string) string {
	return AssistantStyle.Render("Assistant: ") + "\n" + content
}

// FormatThinking formats thinking/reasoning text
func FormatThinking(content string) string {
	return ThinkingStyle.Render("💭 Thinking: ") + ThinkingStyle.Render(content)
}

// FormatError formats an error message
func FormatError(err error) string {
	return ErrorStyle.Render("❌ Error: ") + err.Error()
}

// FormatSuccess formats a success message
func FormatSuccess(msg string) string {
	return SuccessStyle.Render("✓ ") + msg
}

// FormatInfo formats an info message
func FormatInfo(msg string) string {
	return InfoStyle.Render(msg)
}

// FormatSeparator returns a visual separator
func FormatSeparator() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444"))
	return style.Render(strings.Repeat("─", 80))
}
