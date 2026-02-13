package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourstyles "github.com/charmbracelet/glamour/styles"
	"github.com/pedromelo/poly/internal/tui/styles"
)

var mdRenderer *glamour.TermRenderer

// catppuccinGlamourStyle returns a glamour style config based on DarkStyleConfig
// but with Catppuccin colors for headings, code blocks, and links.
func catppuccinGlamourStyle() ansi.StyleConfig {
	style := glamourstyles.DarkStyleConfig

	// Heading colors: use Mauve for visual consistency with the TUI theme
	mauve := styles.ColorToHex(styles.Mauve)
	lavender := styles.ColorToHex(styles.Lavender)
	blue := styles.ColorToHex(styles.Blue)
	green := styles.ColorToHex(styles.Green)
	surface0 := styles.ColorToHex(styles.Surface0)
	overlay0 := styles.ColorToHex(styles.Overlay0)
	text := styles.ColorToHex(styles.Text)

	style.H1.Color = stringPtr(mauve)
	style.H1.Bold = boolPtr(true)
	style.H2.Color = stringPtr(mauve)
	style.H2.Bold = boolPtr(true)
	style.H3.Color = stringPtr(lavender)
	style.H3.Bold = boolPtr(true)
	style.H4.Color = stringPtr(lavender)
	style.H5.Color = stringPtr(lavender)
	style.H6.Color = stringPtr(lavender)

	// Code blocks: Catppuccin Surface0 background
	style.Code.Color = stringPtr(green)
	style.Code.BackgroundColor = stringPtr(surface0)
	style.CodeBlock.Theme = "catppuccin-mocha"
	style.CodeBlock.Chroma = &ansi.Chroma{
		Text: ansi.StylePrimitive{Color: stringPtr(text)},
	}

	// Links: Blue, matching the Info color
	style.Link.Color = stringPtr(blue)
	style.LinkText.Color = stringPtr(blue)
	style.LinkText.Bold = boolPtr(true)

	// Block quote: muted
	style.BlockQuote.Indent = uintPtr(2)
	style.BlockQuote.IndentToken = stringPtr("| ")
	style.BlockQuote.Color = stringPtr(overlay0)

	// Emphasis
	style.Emph.Color = stringPtr(text)
	style.Strong.Color = stringPtr(text)
	style.Strong.Bold = boolPtr(true)

	return style
}

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
func uintPtr(u uint) *uint       { return &u }

func initMarkdown(width int) {
	if width <= 0 {
		width = 80
	}
	mdRenderer, _ = glamour.NewTermRenderer(
		glamour.WithStyles(catppuccinGlamourStyle()),
		glamour.WithWordWrap(width),
	)
}

// renderMarkdown renders markdown content to styled terminal output.
// Falls back to plain text on error or if the renderer is not initialized.
func renderMarkdown(content string, width int) string {
	if mdRenderer == nil || width <= 0 {
		return content
	}
	rendered, err := mdRenderer.Render(content)
	if err != nil {
		return content
	}
	// glamour adds trailing newlines, trim them
	return strings.TrimRight(rendered, "\n")
}
