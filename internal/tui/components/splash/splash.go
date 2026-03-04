package splash

import (
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tui/core"
	"github.com/pedromelo/poly/internal/tui/layout"
)

// ProviderStatus represents a provider's connection state
type ProviderStatus struct {
	Name      string
	Connected bool
	Color     color.Color
}

// Splash is the interface for the splash screen component
type Splash interface {
	layout.Model
	layout.Sizeable
	SetProviders([]ProviderStatus)
	SetVersion(string)
}

type splashCmp struct {
	width, height int
	providers     []ProviderStatus
	version       string
}

// New creates a new splash screen component
func New() Splash {
	return &splashCmp{
		version: "dev",
	}
}

func (s *splashCmp) Init() tea.Cmd { return nil }

func (s *splashCmp) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	return s, nil
}

func (s *splashCmp) View() string {
	if s.width == 0 || s.height == 0 {
		return ""
	}

	// Logo with gradient - 3 lines of diagonal fill for more impact
	logoWidth := layout.SplashLogoWidth
	diag := core.IconDiag

	diagFull := core.GradientText(strings.Repeat(diag, logoWidth), theme.Mauve, theme.Lavender, false)
	diagShort := core.GradientText(strings.Repeat(diag, 3), theme.Mauve, theme.Lavender, false)
	titleText := core.GradientText("\u25C7  P  O  L  Y", theme.Mauve, theme.Lavender, true)

	// Calculate inner padding to center title in the box
	titleVisualWidth := layout.SplashTitleLen
	innerPad := logoWidth - 6 - titleVisualWidth // 6 for 3 diag on each side
	leftPad := innerPad / 2
	rightPad := innerPad - leftPad

	// 3-line bordered logo block
	emptyInner := strings.Repeat(" ", logoWidth-6)
	logoBlock := []string{
		diagFull,
		diagShort + emptyInner + diagShort,
		diagShort + strings.Repeat(" ", leftPad+1) + titleText + strings.Repeat(" ", rightPad+1) + diagShort,
		diagShort + emptyInner + diagShort,
		diagFull,
	}

	// Subtitle
	subtitle := lipgloss.NewStyle().
		Foreground(theme.Overlay1).
		Italic(true).
		Render("multi-model terminal interface")

	// Provider status row
	var provParts []string
	for _, p := range s.providers {
		var icon string
		var clr color.Color
		if p.Connected {
			icon = core.IconCheck
			clr = p.Color
		} else {
			icon = core.IconError
			clr = theme.Overlay0
		}
		iconStr := lipgloss.NewStyle().Foreground(clr).Render(icon)
		nameStr := lipgloss.NewStyle().Foreground(clr).Render(p.Name)
		provParts = append(provParts, iconStr+" "+nameStr)
	}
	providerRow := strings.Join(provParts, "     ")

	// Hint + version
	helpStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
	versionStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	sepStyle := lipgloss.NewStyle().Foreground(theme.Surface2)

	hintLine := helpStyle.Render("type /help for commands") +
		sepStyle.Render(" \u00B7 ") +
		versionStyle.Render(s.version)

	// Assemble vertically with breathing room
	parts := make([]string, 0, 16)
	parts = append(parts, "")
	parts = append(parts, "")
	parts = append(parts, logoBlock...)
	parts = append(parts, "")
	parts = append(parts, subtitle)
	parts = append(parts, "")
	parts = append(parts, providerRow)
	parts = append(parts, "")
	parts = append(parts, "")
	parts = append(parts, hintLine)

	content := lipgloss.JoinVertical(lipgloss.Center, parts...)

	return lipgloss.Place(
		s.width, s.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (s *splashCmp) SetSize(width, height int) tea.Cmd {
	s.width = width
	s.height = height
	return nil
}

func (s *splashCmp) GetSize() (int, int) {
	return s.width, s.height
}

func (s *splashCmp) SetProviders(providers []ProviderStatus) {
	s.providers = providers
}

func (s *splashCmp) SetVersion(v string) {
	s.version = v
}
