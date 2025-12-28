package ui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// ChatModel represents the chat UI model
type ChatModel struct {
	Textarea textarea.Model
	Viewport viewport.Model
	Messages []string
	Ready    bool
}

// NewChatModel creates a new chat model
func NewChatModel() ChatModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()

	vp := viewport.New(80, 20)

	return ChatModel{
		Textarea: ta,
		Viewport: vp,
		Messages: []string{},
		Ready:    false,
	}
}

func (m ChatModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.Textarea, tiCmd = m.Textarea.Update(msg)
	m.Viewport, vpCmd = m.Viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m ChatModel) View() string {
	return m.Viewport.View() + "\n" + m.Textarea.View()
}
