package core

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Section renders a section header: "Title ────────────"
func Section(title string, width int, titleColor, sepColor color.Color) string {
	if width <= 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true)

	rendered := titleStyle.Render(title)
	titleWidth := lipgloss.Width(rendered)

	sepLen := width - titleWidth - 2
	if sepLen < 3 {
		sepLen = 3
	}

	sep := lipgloss.NewStyle().
		Foreground(sepColor).
		Render(" " + strings.Repeat(IconSep, sepLen))

	return rendered + sep
}

// Title renders a title with diagonal fill: "Title ╱╱╱╱╱╱╱╱╱"
func Title(title string, width int, titleColor, diagColor color.Color) string {
	if width <= 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true)

	rendered := titleStyle.Render(title)
	titleWidth := lipgloss.Width(rendered)

	diagLen := width - titleWidth - 1
	if diagLen < 3 {
		diagLen = 3
	}

	diag := lipgloss.NewStyle().
		Foreground(diagColor).
		Render(" " + strings.Repeat(IconDiag, diagLen))

	return rendered + diag
}

// CenteredSection renders a centered section header: "─── Title ───"
func CenteredSection(title string, width int, titleColor, sepColor color.Color) string {
	if width <= 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true)

	rendered := titleStyle.Render(title)
	titleWidth := lipgloss.Width(rendered)

	// Total separators = width - titleWidth - 2 spaces around title
	totalSep := width - titleWidth - 2
	if totalSep < 6 {
		totalSep = 6
	}
	leftSep := totalSep / 2
	rightSep := totalSep - leftSep

	sepStyle := lipgloss.NewStyle().Foreground(sepColor)

	return sepStyle.Render(strings.Repeat(IconSep, leftSep)) +
		" " + rendered + " " +
		sepStyle.Render(strings.Repeat(IconSep, rightSep))
}

// StatusLine renders "● Title  description"
func StatusLine(icon, title, description string, iconColor, titleColor, descColor color.Color) string {
	iconStyle := lipgloss.NewStyle().Foreground(iconColor)
	titleStyle := lipgloss.NewStyle().Foreground(titleColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(descColor)

	return iconStyle.Render(icon) + " " + titleStyle.Render(title) + "  " + descStyle.Render(description)
}

// DiagFill renders a line of diagonal characters
func DiagFill(width int, clr color.Color) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(clr).
		Render(strings.Repeat(IconDiag, width))
}
