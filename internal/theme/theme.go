package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/config"
)

// ThemeName identifies a Catppuccin flavor
type ThemeName string

const (
	ThemeMocha     ThemeName = "mocha"
	ThemeMacchiato ThemeName = "macchiato"
	ThemeFrappe    ThemeName = "frappe"
	ThemeLatte     ThemeName = "latte"
)

var AllThemes = []ThemeName{ThemeMocha, ThemeMacchiato, ThemeFrappe, ThemeLatte}

type CatppuccinPalette struct {
	Rosewater, Flamingo, Pink, Mauve, Red, Maroon, Peach, Yellow, Green, Teal, Sky, Sapphire, Blue, Lavender color.Color
	Text, Subtext1, Subtext0, Overlay2, Overlay1, Overlay0, Surface2, Surface1, Surface0, Base, Mantle, Crust color.Color
}

var Palettes = map[ThemeName]CatppuccinPalette{
	ThemeMocha: {
		Rosewater: lipgloss.Color("#f5e0dc"), Flamingo: lipgloss.Color("#f2cdcd"), Pink: lipgloss.Color("#f5c2e7"), Mauve: lipgloss.Color("#cba6f7"),
		Red: lipgloss.Color("#f38ba8"), Maroon: lipgloss.Color("#eba0ac"), Peach: lipgloss.Color("#fab387"), Yellow: lipgloss.Color("#f9e2af"),
		Green: lipgloss.Color("#a6e3a1"), Teal: lipgloss.Color("#94e2d5"), Sky: lipgloss.Color("#89dceb"), Sapphire: lipgloss.Color("#74c7ec"),
		Blue: lipgloss.Color("#89b4fa"), Lavender: lipgloss.Color("#b4befe"), Text: lipgloss.Color("#cdd6f4"), Subtext1: lipgloss.Color("#bac2de"),
		Subtext0: lipgloss.Color("#a6adc8"), Overlay2: lipgloss.Color("#9399b2"), Overlay1: lipgloss.Color("#7f849c"), Overlay0: lipgloss.Color("#6c7086"),
		Surface2: lipgloss.Color("#585b70"), Surface1: lipgloss.Color("#45475a"), Surface0: lipgloss.Color("#313244"), Base: lipgloss.Color("#1e1e2e"),
		Mantle: lipgloss.Color("#181825"), Crust: lipgloss.Color("#11111b"),
	},
}

var (
	Base, Mantle, Crust, Surface0, Surface1, Surface2, Overlay0, Overlay1, Overlay2 color.Color
	Text, Subtext0, Subtext1, Rosewater, Flamingo, Pink, Mauve, Red, Maroon, Peach, Yellow, Green, Teal, Sky, Sapphire, Blue, Lavender color.Color
	CurrentTheme ThemeName = ThemeMocha
)

// Elite Styles
var (
	AppStyle = lipgloss.NewStyle().Background(Base)

	// Sidebar-like vertical line for messages
	UserPrefixStyle = lipgloss.NewStyle().Foreground(Mauve).Bold(true)
	UserContentStyle = lipgloss.NewStyle().Foreground(Text)

	AssistantPrefixStyle = lipgloss.NewStyle().Bold(true)
	AssistantContentStyle = lipgloss.NewStyle().Foreground(Text)

	// Subtle thinking block
	ThinkingStyle = lipgloss.NewStyle().Foreground(Overlay0).Italic(true).PaddingLeft(2)

	// Clean borders
	SeparatorStyle = lipgloss.NewStyle().Foreground(Surface1)
	
	// Input area
	InputStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderTop(true).
		BorderForeground(Surface1).
		Padding(0, 1)

	// Components
	HeaderStyle = lipgloss.NewStyle().Background(Mantle).Foreground(Text).Padding(0, 1)
	StatusStyle = lipgloss.NewStyle().Background(Mantle).Foreground(Overlay1).Padding(0, 1)
)

func init() {
	SetTheme(ThemeMocha)
}

func SetTheme(name ThemeName) {
	p, ok := Palettes[name]
	if !ok { return }
	CurrentTheme = name
	Base, Mantle, Crust, Surface0, Surface1, Surface2, Overlay0, Overlay1, Overlay2 = p.Base, p.Mantle, p.Crust, p.Surface0, p.Surface1, p.Surface2, p.Overlay0, p.Overlay1, p.Overlay2
	Text, Subtext0, Subtext1, Rosewater, Flamingo, Pink, Mauve, Red, Maroon, Peach, Yellow, Green, Teal, Sky, Sapphire, Blue, Lavender = p.Text, p.Subtext0, p.Subtext1, p.Rosewater, p.Flamingo, p.Pink, p.Mauve, p.Red, p.Maroon, p.Peach, p.Yellow, p.Green, p.Teal, p.Sky, p.Sapphire, p.Blue, p.Lavender
	refreshStyles()
}

func NextTheme() ThemeName {
	for i, t := range AllThemes {
		if t == CurrentTheme {
			next := AllThemes[(i+1)%len(AllThemes)]
			SetTheme(next)
			return next
		}
	}
	SetTheme(ThemeMocha)
	return ThemeMocha
}

func refreshStyles() {
	AppStyle = lipgloss.NewStyle().Background(Base)
	UserPrefixStyle = lipgloss.NewStyle().Foreground(Mauve).Bold(true)
	UserContentStyle = lipgloss.NewStyle().Foreground(Text)
	AssistantPrefixStyle = lipgloss.NewStyle().Bold(true)
	AssistantContentStyle = lipgloss.NewStyle().Foreground(Text)
	ThinkingStyle = lipgloss.NewStyle().Foreground(Overlay0).Italic(true).PaddingLeft(2)
	SeparatorStyle = lipgloss.NewStyle().Foreground(Surface1)
	InputStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(Surface1).Padding(0, 1)
	HeaderStyle = lipgloss.NewStyle().Background(Mantle).Foreground(Text).Padding(0, 1)
	StatusStyle = lipgloss.NewStyle().Background(Mantle).Foreground(Overlay1).Padding(0, 1)
}

func ProviderColor(provider string) color.Color {
	c := config.GetProviderColor(provider)
	if c != "" { return lipgloss.Color(c) }
	return Blue
}
