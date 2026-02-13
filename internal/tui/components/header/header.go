package header

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/layout"
	"github.com/pedromelo/poly/internal/tui/styles"
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

	sep := lipgloss.NewStyle().Foreground(styles.Surface2).Render(" │ ")

	// Logo
	logo := core.GradientText(core.IconModel+" POLY", styles.Mauve, styles.Lavender, true)

	// Default provider: colored dot + name (no @ since multiple providers can be in the same chat)
	provDot := lipgloss.NewStyle().Foreground(h.providerColor).Render("●")
	provName := lipgloss.NewStyle().Foreground(h.providerColor).Bold(true).Render(h.provider)
	provRendered := provDot + " " + provName

	// Context %
	pctStr, pctVal := h.contextPercent()
	var pctColor color.Color
	switch {
	case pctVal >= 80:
		pctColor = styles.Red
	case pctVal >= 50:
		pctColor = styles.Yellow
	default:
		pctColor = styles.Green
	}
	pctRendered := lipgloss.NewStyle().Foreground(pctColor).Render(pctStr)

	// cwd on the right
	cwdRendered := ""
	cwdWidth := 0
	if h.cwd != "" {
		cwdRendered = lipgloss.NewStyle().Foreground(styles.Overlay0).Render(h.cwd)
		cwdWidth = lipgloss.Width(h.cwd)
	}

	left := logo + sep + provRendered + sep + pctRendered
	leftWidth := lipgloss.Width(core.IconModel+" POLY") + 3 + lipgloss.Width("● "+h.provider) + 3 + lipgloss.Width(pctStr)

	gap := h.width - leftWidth - cwdWidth - 4
	if gap < 1 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + cwdRendered

	return lipgloss.NewStyle().
		Width(h.width).
		Padding(0, 1).
		Render(line)
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
