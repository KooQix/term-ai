package chat

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/KooQix/term-ai/internal/config"
	"github.com/KooQix/term-ai/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

func (m chatModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var sb strings.Builder

	// Render header (compact, single-line)
	sb.WriteString(m.renderHeader())
	sb.WriteString("\n")

	// Render viewport (chat messages) - maximize this area
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	// Render input separator (no extra spacing)
	sb.WriteString(ui.FormatSeparator())
	sb.WriteString("\n")

	// Render input area (compact)
	sb.WriteString(m.renderInputArea())

	// Render suggestions if showing (inline, no extra newline)
	if m.showSuggestions && len(m.suggestions) > 0 {
		sb.WriteString("\n")
		sb.WriteString(m.renderSuggestions())
	}

	// Render footer (compact, single-line)
	sb.WriteString("\n")
	sb.WriteString(m.renderFooter())

	return sb.String()
}

func (m chatModel) renderHeader() string {
	// Status indicator
	status := "Ready"
	statusColor := "#00FF00" // Green
	if m.streaming {
		status = "Streaming..."
		statusColor = "#FFAA00" // Orange
	}
	if m.err != nil {
		status = "Error"
		statusColor = "#FF0000" // Red
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(statusColor))

	profileStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#5C4D7B"))

	contextStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#4A90E2"))

	header := headerStyle.Render(" ✨ TermAI ") + " " +
		profileStyle.Render(" "+m.profile.Name+" ") + " " +
		profileStyle.Render(" "+m.profile.Model+" ")

	if m.chatPath != "" {
		chatPathStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5C4D7B"))

		header += " " + chatPathStyle.Render(fmt.Sprintf(" 💾 %s ", config.GetDisplayPath(m.chatPath)))
	}

	// Add context information if present
	if m.contextDirPath != "" && len(m.contextFiles) > 0 {
		contextInfo := fmt.Sprintf(" 📁 %s (%d files) ", filepath.Base(m.contextDirPath), len(m.contextFiles))
		header += " " + contextStyle.Render(contextInfo)
	}

	header += " " + statusStyle.Render(" ● "+status+" ")

	return header
}

func (m chatModel) renderInputArea() string {
	var sb strings.Builder

	// Show attached files if any
	if len(m.attachedFiles) > 0 {
		attachStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Background(lipgloss.Color("#1A1A1A"))

		fileNames := make([]string, 0, len(m.attachedFiles))
		for _, file := range m.attachedFiles {
			fileType := ""
			switch file.Type {
			case "image":
				fileType = "📷"
			case "pdf":
				fileType = "📄"
			case "code":
				fileType = "💻"
			case "text":
				fileType = "📝"
			}
			fileNames = append(fileNames, fmt.Sprintf("%s %s", fileType, file.Name))
		}

		attachInfo := fmt.Sprintf("📎 Attached: %s", strings.Join(fileNames, ", "))
		sb.WriteString(attachStyle.Render(attachInfo))
		sb.WriteString("\n")
	}

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00D9FF")).
		Bold(true)

	sb.WriteString(promptStyle.Render("You ▸ "))
	sb.WriteString(m.textarea.View())

	return sb.String()
}

func (m chatModel) renderSuggestions() string {
	suggestionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Background(lipgloss.Color("#1A1A1A"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true)

	var parts []string
	for i, suggestion := range m.suggestions {
		if i == m.selectedSuggestion {
			parts = append(parts, selectedStyle.Render(" "+suggestion+" "))
		} else {
			parts = append(parts, suggestionStyle.Render(" "+suggestion+" "))
		}
	}

	return suggestionStyle.Render(" ▸ ") + strings.Join(parts, " ")
}

func (m chatModel) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Background(lipgloss.Color("#1A1A1A"))

	hints := footerStyle.Render(" /help /exit /clear /profile ")
	shortcuts := footerStyle.Render(" Alt+Enter or Ctrl+Enter to send | Enter for new line | Ctrl+C=quit ")

	// Calculate spacing
	totalWidth := m.viewport.Width
	hintsWidth := lipgloss.Width(hints)
	shortcutsWidth := lipgloss.Width(shortcuts)
	spacing := max(totalWidth-hintsWidth-shortcutsWidth, 0)

	return hints + strings.Repeat(" ", spacing) + shortcuts
}

func (m *chatModel) updateViewport() {
	content := strings.Join(m.messages, "\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *chatModel) updateSuggestions() {
	input := m.textarea.Value()

	// Only show suggestions if input starts with "/"
	if !strings.HasPrefix(input, "/") {
		m.showSuggestions = false
		m.suggestions = nil
		m.selectedSuggestion = 0
		return
	}

	// Filter commands that match the input
	var matches []string
	for _, cmd := range m.commands.Commands {
		if strings.HasPrefix(cmd, input) {
			matches = append(matches, cmd)
		}
	}

	// Update suggestions
	if len(matches) > 0 {
		m.showSuggestions = true
		m.suggestions = matches
		// Keep selected suggestion in bounds
		if m.selectedSuggestion >= len(matches) {
			m.selectedSuggestion = 0
		}
	} else {
		m.showSuggestions = false
		m.suggestions = nil
		m.selectedSuggestion = 0
	}
}
