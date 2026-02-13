package status

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/tui/layout"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// InfoType determines the visual style of the status message
type InfoType int

const (
	InfoTypeInfo InfoType = iota
	InfoTypeSuccess
	InfoTypeWarning
	InfoTypeError
)

// InfoMsg sets a new status message
type InfoMsg struct {
	Type InfoType
	Msg  string
	TTL  time.Duration // 0 means use default (5s)
}

// ClearStatusMsg clears the current status
type ClearStatusMsg struct{}

const defaultTTL = 5 * time.Second

// StatusCmp is the interface for the status bar component
type StatusCmp interface {
	layout.Model
	SetWidth(int) tea.Cmd
}

type statusCmp struct {
	width             int
	info              *InfoMsg
	provider          string
	providerColor     color.Color
	inputTokens       int
	outputTokens      int
	cacheCreation     int
	cacheRead         int
	cost              float64
	streaming         bool
	streamElapsed     time.Duration
	streamTokens      int
	streamDoneElapsed time.Duration
	streamDoneTokens  int
}

// New creates a new status bar component
func New() StatusCmp {
	return &statusCmp{}
}

func (s *statusCmp) Init() tea.Cmd {
	return nil
}

func (s *statusCmp) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case InfoMsg:
		s.info = &msg
		ttl := msg.TTL
		if ttl == 0 {
			ttl = defaultTTL
		}
		return s, tea.Tick(ttl, func(time.Time) tea.Msg {
			return ClearStatusMsg{}
		})
	case ClearStatusMsg:
		s.info = nil
		return s, nil
	case SetProviderMsg:
		s.provider = msg.Name
		s.providerColor = msg.Color
		return s, nil
	case SetTokensMsg:
		s.inputTokens = msg.Input
		s.outputTokens = msg.Output
		s.cacheCreation = msg.CacheCreation
		s.cacheRead = msg.CacheRead
		s.cost = msg.Cost
		return s, nil
	case SetStreamingMsg:
		if msg.Active {
			s.streaming = true
			s.streamElapsed = msg.Elapsed
			s.streamTokens = msg.Tokens
		} else {
			s.streaming = false
			s.streamDoneElapsed = msg.Elapsed
			s.streamDoneTokens = msg.Tokens
		}
		return s, nil
	}
	return s, nil
}

func (s *statusCmp) View() string {
	dimStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)
	sepStyle := lipgloss.NewStyle().Foreground(styles.Surface2)
	sep := sepStyle.Render(" · ")

	// LEFT: Streaming info
	leftParts := ""
	if s.streaming {
		secs := s.streamElapsed.Seconds()
		tokPerSec := 0.0
		if secs > 0 {
			tokPerSec = float64(s.streamTokens) / secs
		}
		streamStyle := lipgloss.NewStyle().Foreground(styles.Mauve)
		leftParts = streamStyle.Render(fmt.Sprintf("⟳ %.1fs", secs)) +
			sep +
			streamStyle.Render(fmt.Sprintf("%.0f tok/s", tokPerSec))
	} else if s.streamDoneTokens > 0 {
		secs := s.streamDoneElapsed.Seconds()
		tokPerSec := 0.0
		if secs > 0 {
			tokPerSec = float64(s.streamDoneTokens) / secs
		}
		doneStyle := lipgloss.NewStyle().Foreground(styles.Green)
		leftParts = doneStyle.Render(fmt.Sprintf("✓ %.1fs", secs)) +
			sep +
			doneStyle.Render(fmt.Sprintf("%.0f tok/s", tokPerSec))
	}

	// CENTER: Tokens + cost
	centerParts := ""
	if s.inputTokens+s.outputTokens > 0 {
		tokenStr := dimStyle.Render(fmt.Sprintf("%s↑ %s↓",
			formatTokens(s.inputTokens), formatTokens(s.outputTokens)))
		costStr := lipgloss.NewStyle().Foreground(styles.Subtext0).
			Render(fmt.Sprintf("$%.2f", s.cost))
		centerParts = tokenStr + sep + costStr
		if s.cacheRead > 0 {
			cacheStr := lipgloss.NewStyle().Foreground(styles.Surface2).
				Render(fmt.Sprintf("(%s cached)", formatTokens(s.cacheRead)))
			centerParts += " " + cacheStr
		}
	}

	// RIGHT: Status badge + provider
	statusBadge := ""
	if s.info != nil {
		badge := s.renderBadge(s.info.Type)
		msgStyle := lipgloss.NewStyle().Foreground(styles.Subtext0)
		statusBadge = badge + " " + msgStyle.Render(s.info.Msg)
	} else if s.streaming {
		statusBadge = lipgloss.NewStyle().Foreground(styles.Mauve).
			Render("⟳ Streaming")
	} else {
		statusBadge = lipgloss.NewStyle().Foreground(styles.Green).
			Render("✓ Ready")
	}

	// Assemble: left ... center ... status
	right := statusBadge

	// Build the full line
	var parts []string
	if leftParts != "" {
		parts = append(parts, leftParts)
	}
	if centerParts != "" {
		parts = append(parts, centerParts)
	}

	left := strings.Join(parts, "     ")

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gap < 0 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + right

	return lipgloss.NewStyle().
		Background(styles.Mantle).
		Width(s.width).
		Padding(0, 1).
		Render(line)
}

func (s *statusCmp) SetWidth(w int) tea.Cmd {
	s.width = w
	return nil
}

func (s *statusCmp) renderBadge(t InfoType) string {
	switch t {
	case InfoTypeError:
		return lipgloss.NewStyle().
			Background(styles.Red).
			Foreground(styles.Base).
			Bold(true).
			Padding(0, 1).
			Render("ERROR")
	case InfoTypeSuccess:
		return lipgloss.NewStyle().
			Background(styles.Green).
			Foreground(styles.Base).
			Bold(true).
			Padding(0, 1).
			Render("OK")
	case InfoTypeWarning:
		return lipgloss.NewStyle().
			Background(styles.Yellow).
			Foreground(styles.Base).
			Bold(true).
			Padding(0, 1).
			Render("WARN")
	default:
		return ""
	}
}

// SetProviderMsg updates the provider shown in the status bar
type SetProviderMsg struct {
	Name  string
	Color color.Color
}

// SetTokensMsg updates token/cost display
type SetTokensMsg struct {
	Input          int
	Output         int
	CacheCreation  int
	CacheRead      int
	Cost           float64
}

// SetStreamingMsg updates streaming speed indicators
type SetStreamingMsg struct {
	Active   bool
	Elapsed  time.Duration
	Tokens   int
}

// formatTokens formats a token count compactly (e.g. 1.2K, 450)
func formatTokens(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
