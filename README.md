# TermAI - Your AI Assistant in the Terminal 🤖

TermAI is a powerful, cross-platform CLI tool that brings AI assistance directly to your terminal. With support for multiple AI providers, beautiful terminal UI, and streaming responses, TermAI makes interacting with AI models effortless and enjoyable.

## ✨ Features

- **Multi-Provider Support**: Works with OpenAI, Claude (via OpenAI-compatible endpoint), Abacus.AI, Ollama, and any OpenAI-compatible API
- **Profile Management**: Create and manage multiple profiles for different models and configurations
- **File Attachments**: Attach images, PDFs, text files, and code to your prompts
- **Directory Context**: Load entire directories as context for project-wide AI assistance
- **Vision Support**: Send images to vision-capable models (GPT-4 Vision, Claude 3)
- **Streaming Responses**: Real-time streaming with beautiful terminal formatting
- **Interactive Chat Mode**: Full conversation context with an intuitive chat interface
- **One-Line Prompts**: Quick AI responses directly from the command line
- **Rich Terminal UI**: Markdown rendering, syntax highlighting, and beautiful styling
- **Secure Configuration**: API keys stored with appropriate file permissions
- **Cross-Platform**: Works on Linux, macOS, and Windows

## 📦 Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/KooQix/term-ai.git
cd term-ai

# Build the binary
go build -o termai

# Move to PATH (optional)
sudo mv termai /usr/local/bin/

# Or install directly (will be placed in $GOPATH/bin/term-ai)
go install
```

### Using Go Install

```bash
go install github.com/KooQix/term-ai@latest
```

## 🚀 Quick Start

On first run, TermAI will automatically create a default configuration file at `~/.termai/config.yaml` with example profiles:

```bash
termai --help
```

### Configure Your API Keys

Edit the configuration file to add your API keys:

```bash
termai config edit
```

Or manually edit `~/.termai/config.yaml`:

```yaml
default_profile: "abacus"
profiles:
  - name: "abacus"
    provider: "abacus"
    endpoint: "https://api.abacus.ai/v1"
    api_key: "your-abacus-api-key-here"
    model: "gpt-4"
    temperature: 0.7
    max_tokens: 2000
  - name: "openai"
    provider: "openai"
    endpoint: "https://api.openai.com/v1"
    api_key: "your-openai-api-key-here"
    model: "gpt-4"
    temperature: 0.7
    max_tokens: 2000
  - name: "ollama"
    provider: "ollama"
    endpoint: "http://localhost:11434/v1"
    api_key: "ollama"
    model: "llama3.1"
    temperature: 0.7
    max_tokens: 2000
```

## 💡 Usage

### One-Line Prompts

Ask quick questions directly from the command line:

```bash
# Use default profile
termai "Explain quantum computing in simple terms"

# Use specific profile
termai --profile ollama "What is the capital of France?"
termai -p openai "Write a Python function to sort a list"

# Get creative
termai "Write a haiku about programming"
```

### File Attachments

Attach files to your prompts for AI analysis:

```bash
# Attach a single file
termai -file document.pdf "Summarize this document"

# Attach multiple files
termai -file image.png -file code.go "Analyze these files"

# Use short flag
termai -f report.md -f data.json "Compare these files"
```

**Supported file types:**
- **Images**: `.jpg`, `.png`, `.gif`, `.webp` (for vision models)
- **PDFs**: `.pdf` (text extracted automatically)
- **Text**: `.txt`, `.md`, `.markdown`
- **Code**: `.go`, `.py`, `.js`, `.ts`, `.java`, `.c`, `.cpp`, `.rs`, and more

📖 See [FILE_ATTACHMENTS.md](FILE_ATTACHMENTS.md) for comprehensive documentation.

### Interactive Chat Mode

Start an interactive chat session with conversation context:

```bash
# Start chat with default profile
termai chat

# Start chat with specific profile
termai chat --profile ollama
termai chat -p openai

# Start chat with files attached
termai chat -file document.pdf -file code.go

# Load directory as context
termai chat --dir ./project
termai chat -d ./src
```

#### Chat Commands

Within the chat interface, you can use these commands:

- Type your message and press `Alt+Enter` or `Ctrl+Enter` to send
- `Enter` - New line in message
- `/attach <file> [...]` - Attach one or more files
- `/files` - Show currently attached files
- `/clear-files` - Clear all attached files
- `/context` - Show directory context files
- `/context-add <file> [...]` - Add files to context
- `/context-remove <file>` - Remove file from context
- `/exit` or `/quit` - Exit the chat session
- `/clear` - Clear conversation context
- `/profile` - Show current profile info
- `/help` - Show available commands
- `Ctrl+C` - Exit immediately

### Profile Management

Manage multiple AI provider profiles:

```bash
# List all profiles
termai profiles list

# Add a new profile (interactive)
termai profiles add

# Show profile details
termai profiles show <name>

# Set default profile
termai profiles set-default <name>

# Remove a profile
termai profiles remove <name>
```

#### Example: Adding a New Profile

```bash
$ termai profiles add
Add New Profile
────────────────────────────────────────────────────────────────────────────────
Profile name: claude
Provider (openai/claude/abacus/ollama/custom): claude
API Endpoint: https://api.anthropic.com/v1
API Key: your-claude-api-key
Model name: claude-3-5-sonnet-20240620
Temperature (0.0-1.0, default 0.7): 0.7
Max tokens (default 2000): 4000

✓ Profile 'claude' added successfully
```

### Configuration Management

Manage your TermAI configuration:

```bash
# Show current configuration
termai config show

# Show config file path
termai config path

# Edit config in default editor
termai config edit
```

## 🔑 Getting API Keys

### OpenAI
1. Visit [OpenAI Platform](https://platform.openai.com/)
2. Sign up or log in
3. Navigate to API Keys section
4. Create a new API key

### Abacus.AI
1. Visit [Abacus.AI](https://abacus.ai/)
2. Sign up or log in
3. Go to Settings → API Keys
4. Generate a new API key

### Claude (Anthropic)
1. Visit [Anthropic Console](https://console.anthropic.com/)
2. Sign up or log in
3. Navigate to API Keys
4. Create a new API key

Note: Claude requires using OpenAI-compatible libraries/endpoints if you want to use it with OpenAI-compatible clients.

### Ollama (Local)
1. Install Ollama from [ollama.ai](https://ollama.ai/)
2. Pull a model: `ollama pull llama3.1`
3. Ollama runs locally and doesn't require an API key (use "ollama" as placeholder)

## 🎨 Features in Detail

### Streaming Responses

All responses stream in real-time, providing immediate feedback and a smooth user experience. Watch as the AI types its response character by character.

### Markdown Rendering

TermAI automatically renders markdown content with:
- Headers and formatting
- Code blocks with syntax highlighting
- Lists and tables
- Links and emphasis

### Conversation Context

In interactive chat mode, TermAI maintains full conversation context, allowing for natural, flowing conversations with the AI.

### Thinking/Reasoning Display

If the AI model provides thinking or reasoning tokens, TermAI will display them in a distinct style, giving you insight into the AI's thought process.

### Syntax Highlighting & Themes

TermAI provides beautiful syntax highlighting for code, JSON, YAML, XML, and Markdown in both inline and chat modes:

- **Automatic Detection**: Detects content type and applies appropriate highlighting
- **20+ Languages**: Python, JavaScript, Go, Rust, Java, C++, and many more
- **Multiple Themes**: Choose from dark, dracula, monokai, github, solarized-dark, solarized-light, or auto
- **Terminal Optimized**: Works great with modern terminals like Ghostty, iTerm2, Alacritty, kitty, and Windows Terminal

#### Configuring Themes

Add UI configuration to your `~/.termai/config.yaml`:

```yaml
ui:
  theme: "dark"           # Options: dark, dracula, monokai, github, solarized-dark, solarized-light, auto
  color_output: true      # Enable/disable colored output
  show_thinking: true     # Show thinking/reasoning in output
```

Quick edit:

```bash
termai config edit
```

#### Available Themes

| Theme | Best For | Description |
|-------|----------|-------------|
| `dark` | General use | Default dark theme with balanced colors |
| `dracula` | Bold colors | Vibrant purple and pink accents |
| `monokai` | Warm tones | Classic dark theme from Sublime Text |
| `github` | Light mode | Clean, professional light theme |
| `solarized-dark` | Eye comfort | Precision colors for reduced strain |
| `solarized-light` | Light mode | Warm, gentle light theme |
| `auto` | Adaptive | Automatically detects terminal background |

For more details on themes, syntax highlighting, and terminal compatibility, see [THEMES.md](THEMES.md).

#### File Configuration

Configure file attachment behavior in your `~/.termai/config.yaml`:

```yaml
files:
  max_file_size: 10485760          # Maximum file size in bytes (10MB)
  auto_clear_after_send: true      # Clear attached files after sending
  include_context_in_every_msg: false  # Include context files in every message
```

**Configuration options:**
- `max_file_size`: Maximum allowed file size (default: 10MB)
- `auto_clear_after_send`: Automatically clear attached files after sending a message
- `include_context_in_every_msg`: Include directory context files in every message (vs. first message only)

### Error Handling

Clear, helpful error messages guide you when things go wrong:
- Network errors
- API authentication issues
- Invalid configurations
- Missing API keys

## ⚙️ Configuration Options

### Profile Settings

Each profile supports the following settings:

| Setting | Description | Default |
|---------|-------------|---------|
| `name` | Unique profile identifier | Required |
| `provider` | Provider name (openai, claude, abacus, ollama, custom) | Required |
| `endpoint` | API endpoint URL | Required |
| `api_key` | API authentication key | Required |
| `model` | Model identifier | Required |
| `temperature` | Randomness (0.0-1.0) | 0.7 |
| `max_tokens` | Maximum response length | 2000 |
| `top_p` | Nucleus sampling parameter | (optional) |

### UI Settings

Configure the appearance and behavior of TermAI:

| Setting | Description | Default |
|---------|-------------|---------|
| `theme` | Color theme (dark, dracula, monokai, github, solarized-dark, solarized-light, auto) | dark |
| `color_output` | Enable/disable all colored output | true |
| `show_thinking` | Display AI thinking/reasoning process | true |

### Environment Variables

You can also use environment variables:

- `EDITOR` - Default text editor for `termai config edit` (defaults to `vi`)

## 🔒 Security

- API keys are stored in `~/.termai/config.yaml` with file permissions set to `0600` (owner read/write only)
- Never commit your config file to version control
- Add `~/.termai/` to your `.gitignore`
- Use separate API keys for different projects
- Regularly rotate your API keys

## 🐛 Troubleshooting

### "API key not found" or "Invalid API key"

Make sure you've edited the config file and replaced the placeholder API keys with your actual keys:

```bash
termai config edit
```

### Connection Errors with Ollama

Ensure Ollama is running:

```bash
ollama serve
```

Check that the endpoint in your config is correct (default: `http://localhost:11434/v1`)

### Chat Mode Not Displaying Properly

Try resizing your terminal window. The chat UI adapts to terminal size on initialization.

### Config File Not Found

TermAI automatically creates a default config on first run. If you deleted it, run any command to recreate it:

```bash
termai --help
```

## 📚 Advanced Usage

### Custom OpenAI-Compatible Endpoints

TermAI works with any OpenAI-compatible API endpoint:

```yaml
- name: "custom"
  provider: "custom"
  endpoint: "https://your-custom-api.com/v1"
  api_key: "your-api-key"
  model: "your-model-name"
  temperature: 0.7
  max_tokens: 2000
```

### Shell Integration

Create convenient aliases in your shell:

```bash
# Add to ~/.bashrc or ~/.zshrc
alias ai='termai'
alias chat='termai chat'
alias ai-ollama='termai -p ollama'
alias ai-gpt4='termai -p openai'
```

Then use them:

```bash
ai "What is Rust?"
chat
ai-ollama "Tell me a joke"
```

### Piping Input

You can pipe content to termai:

```bash
echo "Explain this code" | termai
cat myfile.py | termai "Review this Python code"
```

## 🛠️ Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/user/termai.git
cd termai

# Install dependencies
go mod download

# Build
go build -o termai

# Run tests
go test ./...

# Install locally
go install
```

### Project Structure

```
termai/
├── main.go                      # Entry point
├── cmd/                         # CLI commands
│   ├── root.go                  # Root command
│   ├── chat.go                  # Interactive chat
│   ├── config.go                # Config management
│   └── profiles.go              # Profile management
├── internal/
│   ├── config/                  # Configuration handling
│   │   └── config.go
│   ├── provider/                # AI provider implementations
│   │   ├── provider.go
│   │   └── openai_compatible.go
│   ├── ui/                      # Terminal UI components
│   │   ├── chat.go
│   │   ├── stream.go
│   │   └── formatter.go
│   └── context/                 # Conversation management
│       └── manager.go
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- OpenAI, Anthropic, Abacus.AI, and Ollama teams for their amazing AI models

## 📧 Support

If you encounter any issues or have questions:
- Open an issue on GitHub
- Check existing issues for solutions
- Read the troubleshooting section above


---
### ☕ Support
If this saved you some time, you can buy me a coffee in crypto — non-custodial, I keep over 99% of every tip:
👉 **[qbf.cash/@kooqix](https://qbf.cash/@kooqix)**
