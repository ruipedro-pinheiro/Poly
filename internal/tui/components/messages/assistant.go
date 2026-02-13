package messages

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// AssistantMessageItem renders an assistant message with tool calls
type AssistantMessageItem struct {
	id            string
	provider      string
	providerColor color.Color
	content       string
	thinking      string
	thinkingTime  time.Duration
	expanded      bool
	toolCalls     []string // pre-rendered tool call lines
	responseTime  time.Duration
}

// NewAssistantMessage creates a new assistant message item
func NewAssistantMessage(id, provider string, providerColor color.Color) *AssistantMessageItem {
	return &AssistantMessageItem{
		id:            id,
		provider:      provider,
		providerColor: providerColor,
	}
}

func (a *AssistantMessageItem) ID() string   { return a.id }
func (a *AssistantMessageItem) Role() string { return "assistant" }

func (a *AssistantMessageItem) SetContent(c string)             { a.content = c }
func (a *AssistantMessageItem) SetThinking(t string)            { a.thinking = t }
func (a *AssistantMessageItem) SetThinkingTime(d time.Duration) { a.thinkingTime = d }
func (a *AssistantMessageItem) SetExpanded(e bool)              { a.expanded = e }
func (a *AssistantMessageItem) SetToolCalls(tc []string)        { a.toolCalls = tc }
func (a *AssistantMessageItem) SetResponseTime(d time.Duration) { a.responseTime = d }
func (a *AssistantMessageItem) ToggleExpanded()                 { a.expanded = !a.expanded }

func (a *AssistantMessageItem) Render(width int) string {
	clr := a.providerColor
	if clr == nil {
		clr = styles.Overlay1
	}

	// Build the header title for the top border
	headerTitle := " " + core.IconModel + " " + a.provider
	if a.responseTime > 0 {
		headerTitle += " " + core.IconSep + " " + formatDuration(a.responseTime)
	}
	headerTitle += " "

	// Inner content width (accounting for border + padding)
	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	var parts []string

	// Thinking block
	if a.thinking != "" {
		parts = append(parts, a.renderThinking(innerWidth))
	}

	// Content
	if a.content != "" {
		contentStyle := lipgloss.NewStyle().
			Foreground(styles.Text).
			Width(innerWidth)
		parts = append(parts, contentStyle.Render(a.content))
	}

	// Tool calls
	for _, tc := range a.toolCalls {
		parts = append(parts, tc)
	}

	body := strings.Join(parts, "\n")

	// Build box with manual border construction
	return buildAssistantBox(body, headerTitle, width, clr)
}

// buildAssistantBox creates a box with rounded borders and title in the top-left border
func buildAssistantBox(content, title string, width int, borderColor color.Color) string {
	const (
		topLeft     = "╭"
		topRight    = "╮"
		bottomLeft  = "╰"
		bottomRight = "╯"
		horizontal  = "─"
		vertical    = "│"
	)

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Foreground(borderColor).Bold(true)

	// Pad content
	paddedContent := lipgloss.NewStyle().
		Width(width - 4).
		Padding(0, 1).
		Render(content)

	// Top border with title on the left
	titleVisualWidth := lipgloss.Width(title)
	rightFill := width - 2 - titleVisualWidth
	if rightFill < 0 {
		rightFill = 0
	}
	topBorder := borderStyle.Render(topLeft) +
		titleStyle.Render(title) +
		borderStyle.Render(strings.Repeat(horizontal, rightFill)+topRight)

	// Bottom border
	bottomFill := width - 2
	if bottomFill < 0 {
		bottomFill = 0
	}
	bottomBorder := borderStyle.Render(bottomLeft + strings.Repeat(horizontal, bottomFill) + bottomRight)

	// Content lines with side borders
	contentLines := strings.Split(paddedContent, "\n")
	var lines []string
	lines = append(lines, topBorder)
	for _, line := range contentLines {
		lineWidth := lipgloss.Width(line)
		padding := width - 2 - lineWidth
		if padding < 0 {
			padding = 0
		}
		lines = append(lines, borderStyle.Render(vertical)+line+strings.Repeat(" ", padding)+borderStyle.Render(vertical))
	}
	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

func (a *AssistantMessageItem) renderThinking(width int) string {
	if a.thinkingTime > 0 {
		timeStr := formatDuration(a.thinkingTime)
		header := lipgloss.NewStyle().
			Foreground(styles.Overlay1).
			Italic(true).
			Render(fmt.Sprintf("%s Thought for %s", core.IconLoading, timeStr))

		if !a.expanded {
			preview := a.thinking
			if len(preview) > 200 {
				preview = preview[:197] + "..."
			}
			thinkStyle := lipgloss.NewStyle().
				Background(styles.Surface0).
				Foreground(styles.Overlay1).
				Italic(true).
				Width(width).
				Padding(0, 1)
			return header + "\n" + thinkStyle.Render(preview)
		}

		thinkStyle := lipgloss.NewStyle().
			Background(styles.Surface0).
			Foreground(styles.Overlay1).
			Italic(true).
			Width(width).
			Padding(0, 1)
		return header + "\n" + thinkStyle.Render(a.thinking)
	}

	// Still thinking (no time yet)
	return lipgloss.NewStyle().
		Foreground(styles.Overlay1).
		Italic(true).
		Render(core.IconLoading + " Thinking...")
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
