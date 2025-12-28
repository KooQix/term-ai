# TermAI Themes and Syntax Highlighting

TermAI provides beautiful syntax highlighting and theming for code, JSON, YAML, XML, and Markdown in both inline and chat modes.

## Available Themes

TermAI supports several color themes optimized for different terminal backgrounds:

### Dark Themes (For Dark Terminal Backgrounds)

#### 1. **dark** (Default)
The default dark theme with balanced colors that work well with most dark terminals.
- Best for: General use, dark terminals
- Colors: Balanced blues, greens, and purples
- Optimized for: Ghostty, iTerm2, Terminal.app, VS Code terminal

#### 2. **dracula**
A popular dark theme with vibrant purple and pink accents.
- Best for: Users who prefer bold, saturated colors
- Colors: Purple, pink, cyan, green
- Inspired by: https://draculatheme.com/

#### 3. **monokai**
Classic dark theme with warm tones.
- Best for: Sublime Text users
- Colors: Yellow, orange, green, pink

#### 4. **solarized-dark**
Precision colors for machines and people, dark version.
- Best for: Long coding sessions, reduced eye strain
- Colors: Carefully selected for optimal contrast
- More info: https://ethanschoonover.com/solarized/

### Light Themes (For Light Terminal Backgrounds)

#### 5. **github**
GitHub's light theme for code.
- Best for: Light terminal backgrounds
- Colors: Clean, professional look
- Similar to: GitHub's web interface

#### 6. **solarized-light**
Solarized light version.
- Best for: Light backgrounds, outdoor work
- Colors: Warm, gentle on eyes

### Adaptive Theme

#### 7. **auto**
Automatically detects your terminal's background color and chooses an appropriate theme.
- Best for: Users who switch between light and dark modes
- Behavior: Adapts to terminal settings

## Configuration

### Changing Theme

Edit your config file (`~/.termai/config.yaml`) to change the theme:

```yaml
ui:
  theme: "dark"           # Options: dark, dracula, monokai, github, solarized-dark, solarized-light, auto
  color_output: true      # Enable/disable all color output
  show_thinking: true     # Show AI thinking/reasoning process
```

### Quick Config Edit

```bash
termai config edit
```

## Syntax Highlighting Features

### Supported Formats

TermAI automatically detects and highlights:

1. **JSON** - Pretty-prints and highlights JSON with color-coded keys, values, and brackets
2. **YAML** - Highlights YAML keys, values, and structure
3. **XML** - Highlights tags, attributes, and content
4. **Markdown** - Renders markdown with headers, lists, code blocks, links, and emphasis
5. **Code Blocks** - Highlights 20+ programming languages including:
   - Python, JavaScript, TypeScript, Go, Rust
   - Java, C, C++, C#
   - Ruby, PHP, Swift, Kotlin
   - Shell scripts, SQL, and more

### How It Works

TermAI uses two powerful libraries:

- **Glamour** - For rendering beautiful Markdown with your chosen theme
- **Chroma** - For syntax highlighting code in various languages

The formatter automatically detects content type and applies appropriate highlighting.

### Examples

#### JSON Response
When you ask for JSON data, you'll see:
- Green for keys
- Blue for string values
- Purple for numbers
- Gray for brackets and punctuation

#### Markdown Response
When the AI responds in Markdown, you'll see:
- Bold headers with colors
- Highlighted code blocks
- Styled lists and quotes
- Colored links

#### Code Explanations
When asking about code, the AI's code examples will have:
- Syntax-aware highlighting
- Language-specific colors
- Clear visual structure

## Disabling Syntax Highlighting

If you prefer plain text output or have terminal compatibility issues:

```yaml
ui:
  color_output: false
```

This will disable all ANSI colors and show plain text output.

## Terminal Compatibility

### Fully Supported Terminals

TermAI works best with modern terminals that support 256 colors and Unicode:

- ✅ **Ghostty** (macOS/Linux) - Excellent support
- ✅ **iTerm2** (macOS) - Excellent support
- ✅ **Terminal.app** (macOS) - Good support
- ✅ **Alacritty** (Cross-platform) - Excellent support
- ✅ **kitty** (Cross-platform) - Excellent support
- ✅ **Windows Terminal** (Windows) - Good support
- ✅ **VS Code Terminal** - Good support
- ✅ **tmux** - Good support (ensure 256 color mode)
- ✅ **screen** - Limited support (use `theme: "github"`)

### Basic Support

- ⚠️ **xterm** - Basic colors only
- ⚠️ **cmd.exe** (Windows) - Limited color support, use Windows Terminal instead
- ⚠️ **Basic SSH terminals** - May have limited colors

### Troubleshooting

#### Colors Not Showing

1. Check if your terminal supports 256 colors:
   ```bash
   echo $TERM
   ```
   Should show `xterm-256color` or similar.

2. Set terminal to 256 color mode (for tmux):
   ```bash
   export TERM=screen-256color
   # or
   export TERM=tmux-256color
   ```

3. Try a different theme:
   ```yaml
   ui:
     theme: "github"  # Light theme might work better
   ```

4. Check terminal preferences:
   - Enable "Use bright colors for bold text"
   - Enable "ANSI colors"
   - Set color scheme to support 256 colors

#### Garbled Output

If you see weird characters instead of colors:

```yaml
ui:
  color_output: false  # Disable all colors
```

#### Theme Not Applying

1. Ensure config is saved properly:
   ```bash
   termai config show
   ```

2. Restart TermAI after config changes

3. Check for typos in theme name

## Theme Development

Want to contribute a new theme? Here's how:

1. Themes are defined in `internal/ui/formatter.go`
2. For markdown themes, see Glamour documentation: https://github.com/charmbracelet/glamour
3. For code themes, see Chroma styles: https://github.com/alecthomas/chroma

## Best Practices

1. **Choose theme based on your terminal background**
   - Dark background → Use `dark`, `dracula`, or `monokai`
   - Light background → Use `github` or `solarized-light`
   - Switching often → Use `auto`

2. **For presentations or screenshots**
   - Use `dracula` for vibrant colors
   - Use `github` for clean, professional look

3. **For long coding sessions**
   - Use `solarized-dark` for reduced eye strain

4. **When sharing output**
   - Consider `color_output: false` for plain text
   - Or use `github` theme for universal compatibility

## Examples by Use Case

### Data Science / JSON/YAML Heavy
```yaml
ui:
  theme: "monokai"  # Great for data structures
```

### Writing Documentation
```yaml
ui:
  theme: "github"  # Clean markdown rendering
```

### Code Review / Development
```yaml
ui:
  theme: "dracula"  # Excellent syntax highlighting
```

### SSH / Remote Work
```yaml
ui:
  theme: "dark"     # Good compatibility
  color_output: true
```

## Getting Help

If you have theme issues:

1. Check this documentation
2. Verify terminal capabilities: `echo $TERM`
3. Try `theme: "auto"` to let TermAI choose
4. Try `color_output: false` as a fallback
5. Report issues at: https://github.com/user/termai/issues

## Credits

- Glamour: https://github.com/charmbracelet/glamour
- Chroma: https://github.com/alecthomas/chroma
- Lipgloss: https://github.com/charmbracelet/lipgloss
- Dracula Theme: https://draculatheme.com/
- Solarized: https://ethanschoonover.com/solarized/
