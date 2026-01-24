package chat

import (
	"fmt"
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
	case "/save":
		c.saveChat(args)
	case "/load":
		c.LoadChat(args)
	case "/cp":
		c.copyLastAssistantMessage()
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
