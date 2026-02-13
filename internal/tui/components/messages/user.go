package messages

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// UserMessageItem renders a user message as a right-aligned chat bubble
type UserMessageItem struct {
	id      string
	content string
	images  int
	width   int
}

// NewUserMessage creates a new user message item
func NewUserMessage(id, content string, images int) *UserMessageItem {
	return &UserMessageItem{
		id:      id,
		content: content,
		images:  images,
	}
}

func (u *UserMessageItem) ID() string   { return u.id }
func (u *UserMessageItem) Role() string { return "user" }

func (u *UserMessageItem) Render(width int) string {
	// Max bubble width: 70% of available width
	maxBubbleWidth := width * 70 / 100
	if maxBubbleWidth < 30 {
		maxBubbleWidth = 30
	}

	// Build title: "You" with optional image indicator
	title := " You "
	if u.images > 0 {
		title = fmt.Sprintf(" You [img:%d] ", u.images)
	}

	// Content inside the bubble
	innerWidth := maxBubbleWidth - 4 // border (2) + padding (2)
	if innerWidth < 10 {
		innerWidth = 10
	}

	contentStyle := lipgloss.NewStyle().
		Foreground(styles.Text).
		Width(innerWidth)

	renderedContent := contentStyle.Render(u.content)

	// Build the box with title in top-right border
	bubble := buildBox(renderedContent, title, maxBubbleWidth, styles.Mauve, true)

	// Right-align the bubble within the full width
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Right).
		Render(bubble)
}

// buildBox creates a box with rounded borders and a title in the top border.
func buildBox(content, title string, width int, borderColor color.Color, titleRight bool) string {
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

	// Top border with title
	titleVisualWidth := lipgloss.Width(title)
	topFillTotal := width - 2 - titleVisualWidth
	if topFillTotal < 0 {
		topFillTotal = 0
	}

	var topBorder string
	if titleRight {
		topBorder = borderStyle.Render(topLeft+strings.Repeat(horizontal, topFillTotal)) +
			titleStyle.Render(title) +
			borderStyle.Render(topRight)
	} else {
		topBorder = borderStyle.Render(topLeft) +
			titleStyle.Render(title) +
			borderStyle.Render(strings.Repeat(horizontal, topFillTotal)+topRight)
	}

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
