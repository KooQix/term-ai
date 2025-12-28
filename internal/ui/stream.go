package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StreamWriter handles streaming output to terminal
type StreamWriter struct {
	content  strings.Builder
	thinking strings.Builder
}

// NewStreamWriter creates a new stream writer
func NewStreamWriter() *StreamWriter {
	return &StreamWriter{}
}

// WriteContent writes content chunk and displays it
func (sw *StreamWriter) WriteContent(chunk string) {
	sw.content.WriteString(chunk)
	fmt.Print(chunk)
}

// WriteThinking writes thinking/reasoning chunk
func (sw *StreamWriter) WriteThinking(chunk string) {
	if sw.thinking.Len() == 0 {
		fmt.Println()
		fmt.Println(FormatThinking(""))
	}
	sw.thinking.WriteString(chunk)
	fmt.Print(chunk)
}

// AccumulateContent accumulates content without displaying it
func (sw *StreamWriter) AccumulateContent(chunk string) {
	sw.content.WriteString(chunk)
}

// AccumulateThinking accumulates thinking without displaying it
func (sw *StreamWriter) AccumulateThinking(chunk string) {
	sw.thinking.WriteString(chunk)
}

// Finish completes the streaming output
func (sw *StreamWriter) Finish() {
	fmt.Println()
}

// GetContent returns the accumulated content
func (sw *StreamWriter) GetContent() string {
	return sw.content.String()
}

// GetThinking returns the accumulated thinking
func (sw *StreamWriter) GetThinking() string {
	return sw.thinking.String()
}

// GetFormattedContent returns the accumulated content with enhanced formatting
func (sw *StreamWriter) GetFormattedContent() (string, error) {
	content := sw.content.String()
	return FormatResponse(content)
}

// ShowSpinner displays a loading spinner
func ShowSpinner(msg string) {
	spinner := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00"))
	fmt.Fprint(os.Stderr, spinner.Render("⏳ "+msg+"..."))
}

// ClearSpinner clears the spinner line
func ClearSpinner() {
	fmt.Fprint(os.Stderr, "\r\033[K")
}
