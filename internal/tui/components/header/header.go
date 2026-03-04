package header

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/layout"
)

const (
	headerCompactWidth = 82
	headerFullWidth    = 104
	headerProviderMax  = 14
	headerMinCwdWidth  = 8
)

// Header is the interface for the header bar component
type Header interface {
	layout.Model
	layout.Sizeable
	SetProvider(name string, color color.Color)
	SetThinkingMode(bool)
	SetTokens(input, output int, cost float64)
	SetCwd(string)
}

type headerCmp struct {
	width         int
	height        int
	provider      string
	providerColor color.Color
	thinkingMode  bool
	inputTokens   int
	outputTokens  int
	cost          float64
	cwd           string
}

// New creates a new header component
func New() Header {
	return &headerCmp{
		height: layout.HeaderHeight,
	}
}

func (h *headerCmp) Init() tea.Cmd {
	return nil
}

func (h *headerCmp) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	return h, nil
}

func (h *headerCmp) View() string {
	if h.width == 0 {
		return ""
	}

	sep := lipgloss.NewStyle().Foreground(theme.Surface2).Render(" │ ")

	logo := core.GradientText("POLY", theme.Mauve, theme.Lavender, true)
	provider := h.provider
	if provider == "" {
		provider = "none"
	}
	provColor := h.providerColor
	if provColor == nil {
		provColor = theme.Overlay1
	}
	providerChip := lipgloss.NewStyle().
		Foreground(provColor).
		Bold(true).
		Render("@" + truncateRight(provider, headerProviderMax))
	modeChip := h.renderModeLabel()

	pctStr, pctVal := h.contextPercent()
	var pctColor color.Color
	switch {
	case pctVal >= 80:
		pctColor = theme.Red
	case pctVal >= 50:
		pctColor = theme.Yellow
	default:
		pctColor = theme.Green
	}
	pctRendered := lipgloss.NewStyle().Foreground(pctColor).Render(pctStr)
	costRendered := lipgloss.NewStyle().Foreground(theme.Overlay0).Render(fmt.Sprintf("$%.2f", h.cost))

	var line string
	if h.width < headerCompactWidth {
		line = logo
	} else if h.width < headerFullWidth {
		left := logo + sep + providerChip + sep + modeChip
		if h.inputTokens+h.outputTokens > 0 || h.cost > 0 {
			left += sep + pctRendered + sep + costRendered
		}
		line = left
	} else {
		left := logo + sep + providerChip + sep + modeChip
		if h.inputTokens+h.outputTokens > 0 || h.cost > 0 {
			left += sep + pctRendered + sep + costRendered
		}
		leftWidth := lipgloss.Width(left)

		cwdRendered := ""
		if h.cwd != "" {
			avail := h.width - leftWidth - 3
			if avail > headerMinCwdWidth {
				cwd := truncateLeft(h.cwd, avail)
				cwdRendered = lipgloss.NewStyle().Foreground(theme.Overlay0).Render(cwd)
			}
		}

		gap := h.width - leftWidth - lipgloss.Width(cwdRendered) - 2
		if gap < 1 {
			gap = 1
		}
		line = left + strings.Repeat(" ", gap) + cwdRendered
	}

	return lipgloss.NewStyle().
		Width(h.width).
		Background(theme.Mantle).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(theme.Surface2).
		Padding(0, 1).
		Render(line)
}

func (h *headerCmp) renderModeLabel() string {
	mode := "FAST"
	clr := theme.Teal
	if h.thinkingMode {
		mode = "THINK"
		clr = theme.Lavender
	}
	return lipgloss.NewStyle().
		Foreground(clr).
		Bold(true).
		Render(mode)
}

// contextPercent returns a formatted string and the numeric value
func (h *headerCmp) contextPercent() (string, int) {
	total := h.inputTokens + h.outputTokens
	if total == 0 {
		return "0%", 0
	}
	const maxContext = layout.DefaultContextWindow
	pct := float64(total) / float64(maxContext) * 100
	if pct < 1 && total > 0 {
		pct = 1
	}
	if pct > 100 {
		pct = 100
	}
	intPct := int(pct)
	return fmt.Sprintf("%d%%", intPct), intPct
}

func truncateLeft(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 3 {
		return strings.Repeat(".", max)
	}
	runes := []rune(s)
	for i := len(runes) - 1; i >= 0; i-- {
		candidate := "..." + string(runes[i:])
		if lipgloss.Width(candidate) <= max {
			return candidate
		}
	}
	return "..."
}

func truncateRight(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 2 {
		return strings.Repeat(".", max)
	}
	runes := []rune(s)
	for i := len(runes); i >= 0; i-- {
		candidate := string(runes[:i]) + ".."
		if lipgloss.Width(candidate) <= max {
			return candidate
		}
	}
	return strings.Repeat(".", max)
}

func (h *headerCmp) SetSize(width, height int) tea.Cmd {
	h.width = width
	h.height = height
	return nil
}

func (h *headerCmp) GetSize() (int, int) {
	return h.width, h.height
}

func (h *headerCmp) SetProvider(name string, color color.Color) {
	h.provider = name
	h.providerColor = color
}

func (h *headerCmp) SetThinkingMode(on bool) {
	h.thinkingMode = on
}

func (h *headerCmp) SetTokens(input, output int, cost float64) {
	h.inputTokens = input
	h.outputTokens = output
	h.cost = cost
}

func (h *headerCmp) SetCwd(cwd string) {
	h.cwd = cwd
}
