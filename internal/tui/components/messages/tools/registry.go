package tools

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tui/core"
)

// Factory creates a renderer for specific tool opts
type Factory func(*RenderOpts) ToolRenderer

var renderers = map[string]Factory{}

// Register adds a tool renderer factory
func Register(name string, factory Factory) {
	renderers[name] = factory
}

// Get returns the factory for a given tool name, or nil
func Get(name string) Factory {
	return renderers[name]
}

// RenderToolCall renders a tool call using the registry, with fallback to generic
func RenderToolCall(width int, opts *RenderOpts) string {
	factory := Get(opts.Name)
	if factory == nil {
		factory = Get("_generic")
	}
	if factory == nil {
		return renderFallback(width, opts)
	}
	renderer := factory(opts)
	return renderer.Render(width, opts)
}

func renderFallback(width int, opts *RenderOpts) string {
	return FormatHeader(opts.Status, opts.Name, "", "", width)
}

// displayNames maps tool names to prettier display names
var displayNames = map[string]string{
	"bash":       "bash",
	"read_file":  "read",
	"write_file": "write",
	"edit_file":  "edit",
	"glob":       "glob",
	"grep":       "grep",
	"web_fetch":  "fetch",
	"web_search": "search",
	"todos":      "todos",
	"multiedit":  "multiedit",
}

// DisplayName returns the pretty display name for a tool
func DisplayName(name string) string {
	if d, ok := displayNames[name]; ok {
		return d
	}
	return name
}

// ShortenPath shortens a file path to just the filename or last 2 segments
func ShortenPath(path string) string {
	if path == "" {
		return ""
	}
	// If short enough, keep it
	if len(path) <= 40 {
		return path
	}
	// Use last 2 path segments
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	parent := filepath.Base(dir)
	if parent == "." || parent == "/" {
		return base
	}
	return parent + "/" + base
}

// statusIcon returns the icon and color for a tool status
func statusIcon(s ToolStatus) (string, color.Color) {
	switch s {
	case ToolStatusPending:
		return core.IconPending, theme.Yellow
	case ToolStatusRunning:
		return core.IconLoading, theme.Mauve
	case ToolStatusSuccess:
		return core.IconCheck, theme.Green
	case ToolStatusError:
		return core.IconError, theme.Red
	case ToolStatusDenied:
		return core.IconError, theme.Red
	default:
		return core.IconPending, theme.Overlay0
	}
}

// StatusIcon is exported for use by individual renderers
func StatusIcon(s ToolStatus) (string, color.Color) {
	return statusIcon(s)
}

// FormatHeader creates compact inline: "  ✓ bash: ls -la  (3 files)"
// name = tool name, args = main arg, suffix = optional summary like "(3 files)"
func FormatHeader(status ToolStatus, name string, args string, suffix string, width int) string {
	icon, iconColor := statusIcon(status)
	iconStyle := lipgloss.NewStyle().Foreground(iconColor)

	displayName := DisplayName(name)

	// For success: dim the whole thing
	isDone := status == ToolStatusSuccess

	var nameStyle, argsStyle, suffixStyle lipgloss.Style
	if isDone {
		nameStyle = lipgloss.NewStyle().Foreground(theme.Overlay0).Bold(true)
		argsStyle = lipgloss.NewStyle().Foreground(theme.Surface2)
		suffixStyle = lipgloss.NewStyle().Foreground(theme.Surface2)
	} else {
		nameStyle = lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
		argsStyle = lipgloss.NewStyle().Foreground(theme.Overlay1)
		suffixStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)
	}

	header := "  " + iconStyle.Render(icon) + " " + nameStyle.Render(displayName)

	if args != "" {
		header += argsStyle.Render(": ")

		// Calculate available space for args
		// "  X name: " + args + "  (suffix)"
		usedLen := 2 + 2 + 1 + len(displayName) + 2 // indent + icon + space + name + ": "
		suffixLen := 0
		if suffix != "" {
			suffixLen = len(suffix) + 3 // "  " + suffix
		}
		maxArgsLen := width - usedLen - suffixLen
		if maxArgsLen < 10 {
			maxArgsLen = 10
		}
		displayArgs := args
		if len(displayArgs) > maxArgsLen {
			displayArgs = displayArgs[:maxArgsLen-1] + "~"
		}
		header += argsStyle.Render(displayArgs)
	}

	if suffix != "" {
		header += "  " + suffixStyle.Render(suffix)
	}

	return header
}

// FormatResultPreview creates indented result lines with ┃ prefix
func FormatResultPreview(result string, maxLines int, width int) string {
	lines := strings.Split(result, "\n")
	// Remove trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}

	pipeStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	textStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)

	var output []string
	showLines := maxLines
	if showLines > len(lines) {
		showLines = len(lines)
	}

	for i := 0; i < showLines; i++ {
		line := lines[i]
		maxLen := width - 6
		if maxLen > 0 && len(line) > maxLen {
			line = line[:maxLen-1] + "~"
		}
		output = append(output, "  "+pipeStyle.Render("┃ ")+textStyle.Render(line))
	}

	if len(lines) > showLines {
		remaining := len(lines) - showLines
		output = append(output, "  "+textStyle.Render(fmt.Sprintf("  +%d lines", remaining)))
	}

	return strings.Join(output, "\n")
}

// FormatDetail creates a single detail line: "  ┃ detail"
func FormatDetail(detail string) string {
	pipeStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	textStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
	return "  " + pipeStyle.Render("┃ ") + textStyle.Render(detail)
}

// FormatSummary creates a compact summary line
func FormatSummary(status ToolStatus, name string, description string) string {
	icon, iconColor := statusIcon(status)
	iconStyle := lipgloss.NewStyle().Foreground(iconColor)

	isDone := status == ToolStatusSuccess
	var nameStyle, descStyle lipgloss.Style
	if isDone {
		nameStyle = lipgloss.NewStyle().Foreground(theme.Overlay0).Bold(true)
		descStyle = lipgloss.NewStyle().Foreground(theme.Surface2)
	} else {
		nameStyle = lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
		descStyle = lipgloss.NewStyle().Foreground(theme.Overlay1)
	}

	displayName := DisplayName(name)
	line := "  " + iconStyle.Render(icon) + " " + nameStyle.Render(displayName)
	if description != "" {
		line += "  " + descStyle.Render(description)
	}
	return line
}

// FormatError creates an error detail line
func FormatError(msg string) string {
	errStyle := lipgloss.NewStyle().Foreground(theme.Red)
	return "  " + errStyle.Render("  "+msg)
}

// RenderBatchSummary renders a single-line summary when all tools succeeded
// e.g. "  ✓ 4 tools: read x2, bash, edit"
func RenderBatchSummary(toolNames []string) string {
	// Count occurrences
	counts := map[string]int{}
	order := []string{}
	for _, n := range toolNames {
		if counts[n] == 0 {
			order = append(order, n)
		}
		counts[n]++
	}

	var parts []string
	for _, n := range order {
		dn := DisplayName(n)
		if counts[n] > 1 {
			parts = append(parts, fmt.Sprintf("%s x%d", dn, counts[n]))
		} else {
			parts = append(parts, dn)
		}
	}

	icon, iconColor := statusIcon(ToolStatusSuccess)
	iconStyle := lipgloss.NewStyle().Foreground(iconColor)
	textStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)

	total := len(toolNames)
	return "  " + iconStyle.Render(icon) + " " +
		textStyle.Render(fmt.Sprintf("%d tools: %s", total, strings.Join(parts, ", ")))
}
