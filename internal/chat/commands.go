package chat

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/KooQix/term-ai/internal/config"
	"github.com/KooQix/term-ai/internal/fileprocessor"
	"github.com/KooQix/term-ai/internal/provider"
	"github.com/KooQix/term-ai/internal/ui"
	"github.com/KooQix/term-ai/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

type commandHandler struct {
	m *chatModel
}

func newCommandHandler(m *chatModel) *commandHandler {
	return &commandHandler{
		m: m,
	}
}

func (c *commandHandler) handle(cmd string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return c.m, nil
	}

	command := parts[0]
	args := parts[1:]

	// var res tea.Model
	// var err tea.Cmd

	switch command {
	case "/exit", "/quit":
		return c.m, tea.Quit
	case "/clear":
		c.m.ctxManager.Clear()
		c.m.AddMessage(ui.FormatSuccess("Conversation context cleared"))
	case "/profile":
		c.profile()
	case "/attach":
		c.attach(args)
	case "/files":
		c.listAttachments()
	case "/clear-files":
		count := len(c.m.attachedFiles)
		c.m.attachedFiles = nil
		c.m.AddMessage(ui.FormatSuccess(fmt.Sprintf("Cleared %d attached file(s)", count)))
	case "/context":
		c.listContext()
	case "/context-add":
		c.addContext(args)
	case "/context-remove":
		c.removeContext(args)
	case "/add-message":
		c.addMessage(args)
	case "/save":
		c.saveChat(args)
	case "/load":
		c.LoadChat(args)
	case "/cp":
		c.copyLastAssistantMessage()
	case "/pager":
		pagerCmd := c.pager()
		c.m.textarea.Reset()
		c.m.updateViewport()
		return c.m, pagerCmd
	case "/help":
		c.m.AddMessage(ui.InfoStyle.Render(c.m.commands.Available))
	default:
		c.m.AddMessage(ui.FormatError(fmt.Errorf("unknown command: %s", command)))
	}

	// if err != nil {
	// 	return res, err
	// }

	c.m.textarea.Reset()
	c.m.updateViewport()

	return c.m, nil
}

func (c *commandHandler) profile() {
	info := fmt.Sprintf("Current Profile: %s\n", c.m.Profile.Name)
	info += fmt.Sprintf("Provider: %s\n", c.m.Profile.Provider)
	info += fmt.Sprintf("Model: %s\n", c.m.Profile.Model)
	info += fmt.Sprintf("Endpoint: %s\n", c.m.Profile.Endpoint)
	info += fmt.Sprintf("Temperature: %.2f\n", c.m.Profile.Temperature)
	info += fmt.Sprintf("Max Tokens: %d\n", c.m.Profile.MaxTokens)

	c.m.AddMessage(ui.InfoStyle.Render(info))
}

func (c *commandHandler) attach(args []string) {
	if len(args) == 0 {
		c.m.AddMessage(ui.FormatError(fmt.Errorf("/attach requires at least one file path")))
	} else {
		// Process files
		attachments, err := fileprocessor.ProcessFiles(args)
		if err != nil {
			c.m.AddMessage(ui.FormatError(err))
		} else {
			c.m.attachedFiles = append(c.m.attachedFiles, attachments...)
			c.m.AddMessage(ui.FormatSuccess(fmt.Sprintf("Attached %d file(s)", len(attachments))))
		}

	}
}

func (c *commandHandler) listAttachments() {
	if len(c.m.attachedFiles) == 0 {
		c.m.AddMessage(ui.InfoStyle.Render("No files currently attached"))
	} else {
		info := fmt.Sprintf("Attached files (%d):\n", len(c.m.attachedFiles))
		for _, file := range c.m.attachedFiles {
			info += fmt.Sprintf("  • %s (%s)\n", file.Name, file.Type)
		}
		c.m.AddMessage(ui.InfoStyle.Render(info))
	}
}

func (c *commandHandler) listContext() {
	if len(c.m.contextFiles) == 0 {
		c.m.AddMessage(ui.InfoStyle.Render("No context files loaded"))
	} else {
		info := fmt.Sprintf("Context: %s (%d files)\n", c.m.contextDirPath, len(c.m.contextFiles))
		info += "Files:\n"
		for _, file := range c.m.contextFiles {
			info += fmt.Sprintf("  • %s (%s)\n", file.Name, file.Type)
		}
		c.m.AddMessage(ui.InfoStyle.Render(info))
	}
}

func (c *commandHandler) addContext(args []string) {
	if len(args) == 0 {
		c.m.AddMessage(ui.FormatError(fmt.Errorf("/context-add requires at least one file path")))
	} else {
		// Process files and add to context
		attachments, err := fileprocessor.ProcessFiles(args)
		if err != nil {
			c.m.AddMessage(ui.FormatError(err))
		} else {
			c.m.AddContextFiles(attachments)
			c.m.AddMessage(ui.FormatSuccess(fmt.Sprintf("Added %d file(s) to context", len(attachments))))
		}

	}
}

func (c *commandHandler) removeContext(args []string) {
	if len(args) == 0 {
		c.m.AddMessage(ui.FormatError(fmt.Errorf("/context-remove requires a filename")))
	} else {
		filename := strings.Join(args, " ")
		removed := false
		for i, file := range c.m.contextFiles {
			if file.Name == filename || file.Path == filename {
				c.m.contextFiles = append(c.m.contextFiles[:i], c.m.contextFiles[i+1:]...)
				removed = true
				break
			}
		}
		if removed {
			c.m.AddMessage(ui.FormatSuccess(fmt.Sprintf("Removed '%s' from context", filename)))
		} else {
			c.m.AddMessage(ui.FormatError(fmt.Errorf("file '%s' not found in context", filename)))
		}
	}
}

func (c *commandHandler) addMessage(args []string) {
	if len(args) == 0 {
		c.m.AddMessage(ui.FormatError(fmt.Errorf("/add-message requires a message text")))
	} else {
		text := strings.Join(args, " ")
		// Add message to the context manager without sending it as a user message
		// Add it to the ui as well so that the user can see it in the conversation history
		c.m.ctxManager.AddUserMessage(text)
		c.m.AddMessage(ui.FormatUserMessage(text))
	}
}

func (c *commandHandler) saveChat(args []string) {
	// If the chatPath is already set (an no name/path is provided), use it as default
	if c.m.chatPath != "" && len(args) == 0 {
		// Save chat to existing path
		if err := c.m.ctxManager.Save(c.m.chatPath); err != nil {
			c.m.AddMessage(ui.FormatError(fmt.Errorf("failed to save chat: %v", err)))
		} else {
			c.m.AddMessage(ui.FormatSuccess(fmt.Sprintf("Chat saved successfully to '%s'", c.m.chatPath)))
		}
		return
	}

	// Otherwise, expect a name and optional directory
	if len(args) == 0 {
		c.m.AddMessage(ui.FormatError(fmt.Errorf("/save requires a chat name")))
	} else {
		name := args[0]
		dir, err := config.GetDefaultChatsPath()
		if err != nil {
			c.m.AddMessage(ui.FormatError(fmt.Errorf("failed to get default chats path: %w", err)))
			return
		}
		if len(args) > 2 && args[1] == "-d" {
			if absDir, err := utils.GetAbsolutePath(args[2]); err != nil {
				c.m.AddMessage(ui.FormatError(fmt.Errorf("invalid directory path: %w", err)))
				return
			} else {
				dir = absDir
			}
		}

		// Save chat
		if err := c.m.ctxManager.Save(filepath.Join(dir, name)); err != nil {
			c.m.AddMessage(ui.FormatError(fmt.Errorf("failed to save chat: %v", err)))
		} else {
			c.m.AddMessage(ui.FormatSuccess(fmt.Sprintf("Conversation '%s' saved successfully", name)))
		}

		// Set the chatPath for future saves (so that subsequent /save commands without a name will overwrite the same file)
		c.m.chatPath = filepath.Join(dir, name)
	}

}

func (c *commandHandler) LoadChat(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		c.m.AddMessage(ui.FormatError(fmt.Errorf("/load requires a chat file path")))
	} else {
		path := args[0] + config.ChatFileExt

		// Looking for relative path in default chats directory
		if !filepath.IsAbs(path) {
			defaultDir, err := config.GetDefaultChatsPath()
			if err != nil {
				c.m.AddMessage(ui.FormatError(fmt.Errorf("failed to get default chats path: %w", err)))
				return c.m, nil
			}

			if _, err := os.Stat(filepath.Join(defaultDir, path)); err == nil {
				// File exists as given (relative to cwd), use it
				path = filepath.Join(defaultDir, path)
			}
		}

		// Load chat
		if absPath, err := utils.GetAbsolutePath(path); err != nil {
			c.m.AddMessage(ui.FormatError(fmt.Errorf("invalid file path: %w", err)))
		} else {
			err := c.m.loadChat(absPath)
			if err != nil {
				c.m.AddMessage(ui.FormatError(fmt.Errorf("failed to load chat: %v", err)))
			} else {
				// Successfully loaded, set chatPath
				c.m.chatPath = absPath
			}
			// Success message added in loadChat
		}
	}

	return c.m, nil
}

// pagerExec is a tea.ExecCommand that dumps text to stdout and blocks until
// the user presses Enter (\n or \r). Implemented in Go rather than as a shell
// `cat + read` so we can control stdin/stdout precisely — the shell version
// silently short-circuited on some setups (the `read` builtin returned
// immediately, snapping Bubble Tea back into alt-screen before anything was
// visible).
type pagerExec struct {
	content string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func (p *pagerExec) SetStdin(r io.Reader)  { p.stdin = r }
func (p *pagerExec) SetStdout(w io.Writer) { p.stdout = w }
func (p *pagerExec) SetStderr(w io.Writer) { p.stderr = w }

func (p *pagerExec) Run() error {
	if _, err := io.WriteString(p.stdout, p.content); err != nil {
		return err
	}
	if _, err := io.WriteString(p.stdout, "\n\033[7m press Enter to return to chat \033[0m "); err != nil {
		return err
	}

	// Read byte-by-byte and stop on the first line terminator. Accepting both
	// \n and \r covers cooked and raw modes — we don't want to depend on
	// Bubble Tea's release leaving the tty in any specific state.
	r := bufio.NewReader(p.stdin)
	for {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if b == '\n' || b == '\r' {
			return nil
		}
	}
}

// pager dumps the current chat view to the main terminal buffer and blocks
// on a "press Enter" prompt. While the prompt is up the user is fully outside
// Bubble Tea's alt-screen — scroll the terminal natively (mouse wheel /
// Cmd+Up), select across screens with the mouse, Cmd+C to copy. Press Enter
// to resume the TUI.
//
// We deliberately don't use less / bat / $PAGER: those tools own the screen
// (and often enter their own alt-screen), so the moment they quit Bubble Tea
// re-enters alt-screen and hides everything before selection is possible. A
// plain dump + read keeps the content visible in the main buffer for the
// entire viewing window.
func (c *commandHandler) pager() tea.Cmd {
	content := strings.Join(c.m.messages, "\n")
	if strings.TrimSpace(content) == "" {
		c.m.AddMessage(ui.InfoStyle.Render("Nothing to view yet."))
		return nil
	}

	return tea.Exec(&pagerExec{content: content}, func(err error) tea.Msg {
		if err != nil {
			return errMsg{err}
		}
		return nil
	})
}

func (c *commandHandler) copyLastAssistantMessage() {
	// Copy last assistant response to clipboard
	var lastResponse string
	// Get the last assistant message from context manager
	messages := c.m.ctxManager.GetMessages()
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == provider.RoleAssistant {
			lastResponse = messages[i].Content
			break
		}
	}

	if lastResponse != "" {
		if err := utils.CopyToClipboard(lastResponse); err != nil {
			c.m.AddMessage(ui.FormatError(fmt.Errorf("failed to copy to clipboard: %v", err)))
		}
	}
}
