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

// AllThemes returns theme names in cycle order
var AllThemes = []ThemeName{ThemeMocha, ThemeMacchiato, ThemeFrappe, ThemeLatte}

// CatppuccinPalette holds all colors for a Catppuccin flavor
type CatppuccinPalette struct {
	Rosewater color.Color
	Flamingo  color.Color
	Pink      color.Color
	Mauve     color.Color
	Red       color.Color
	Maroon    color.Color
	Peach     color.Color
	Yellow    color.Color
	Green     color.Color
	Teal      color.Color
	Sky       color.Color
	Sapphire  color.Color
	Blue      color.Color
	Lavender  color.Color
	Text      color.Color
	Subtext1  color.Color
	Subtext0  color.Color
	Overlay2  color.Color
	Overlay1  color.Color
	Overlay0  color.Color
	Surface2  color.Color
	Surface1  color.Color
	Surface0  color.Color
	Base      color.Color
	Mantle    color.Color
	Crust     color.Color
}

// Palettes maps theme names to their color palettes
var Palettes = map[ThemeName]CatppuccinPalette{
	ThemeMocha: {
		Rosewater: lipgloss.Color("#f5e0dc"),
		Flamingo:  lipgloss.Color("#f2cdcd"),
		Pink:      lipgloss.Color("#f5c2e7"),
		Mauve:     lipgloss.Color("#cba6f7"),
		Red:       lipgloss.Color("#f38ba8"),
		Maroon:    lipgloss.Color("#eba0ac"),
		Peach:     lipgloss.Color("#fab387"),
		Yellow:    lipgloss.Color("#f9e2af"),
		Green:     lipgloss.Color("#a6e3a1"),
		Teal:      lipgloss.Color("#94e2d5"),
		Sky:       lipgloss.Color("#89dceb"),
		Sapphire:  lipgloss.Color("#74c7ec"),
		Blue:      lipgloss.Color("#89b4fa"),
		Lavender:  lipgloss.Color("#b4befe"),
		Text:      lipgloss.Color("#cdd6f4"),
		Subtext1:  lipgloss.Color("#bac2de"),
		Subtext0:  lipgloss.Color("#a6adc8"),
		Overlay2:  lipgloss.Color("#9399b2"),
		Overlay1:  lipgloss.Color("#7f849c"),
		Overlay0:  lipgloss.Color("#6c7086"),
		Surface2:  lipgloss.Color("#585b70"),
		Surface1:  lipgloss.Color("#45475a"),
		Surface0:  lipgloss.Color("#313244"),
		Base:      lipgloss.Color("#1e1e2e"),
		Mantle:    lipgloss.Color("#181825"),
		Crust:     lipgloss.Color("#11111b"),
	},
}

// Active color variables
var (
	Base      = Palettes[ThemeMocha].Base
	Mantle    = Palettes[ThemeMocha].Mantle
	Crust     = Palettes[ThemeMocha].Crust
	Surface0  = Palettes[ThemeMocha].Surface0
	Surface1  = Palettes[ThemeMocha].Surface1
	Surface2  = Palettes[ThemeMocha].Surface2
	Overlay0  = Palettes[ThemeMocha].Overlay0
	Overlay1  = Palettes[ThemeMocha].Overlay1
	Overlay2  = Palettes[ThemeMocha].Overlay2
	Text      = Palettes[ThemeMocha].Text
	Subtext0  = Palettes[ThemeMocha].Subtext0
	Subtext1  = Palettes[ThemeMocha].Subtext1
	Rosewater = Palettes[ThemeMocha].Rosewater
	Flamingo  = Palettes[ThemeMocha].Flamingo
	Pink      = Palettes[ThemeMocha].Pink
	Mauve     = Palettes[ThemeMocha].Mauve
	Red       = Palettes[ThemeMocha].Red
	Maroon    = Palettes[ThemeMocha].Maroon
	Peach     = Palettes[ThemeMocha].Peach
	Yellow    = Palettes[ThemeMocha].Yellow
	Green     = Palettes[ThemeMocha].Green
	Teal      = Palettes[ThemeMocha].Teal
	Sky       = Palettes[ThemeMocha].Sky
	Sapphire  = Palettes[ThemeMocha].Sapphire
	Blue      = Palettes[ThemeMocha].Blue
	Lavender  = Palettes[ThemeMocha].Lavender

	CurrentTheme ThemeName = ThemeMocha
)

// Styles (Refonte complète pour un look "Pro")
var (
	AppStyle = lipgloss.NewStyle().Background(Base)

	// User Bubble: un bloc qui ressort
	UserBubbleStyle = lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.ThickBorder()).
			BorderLeft(true).
			BorderForeground(Mauve).
			MarginLeft(1)

	// Assistant Header: élégant et discret
	AssistantHeaderStyle = lipgloss.NewStyle().
				Foreground(Blue).
				Bold(true).
				MarginBottom(0)

	// Thinking Block: un style "code" ou "draft"
	ThinkingStyle = lipgloss.NewStyle().
			Foreground(Overlay1).
			Italic(true).
			Padding(0, 1).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Surface1)

	// Input Box: un vrai champ qui respire
	InputBoxStyle = lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Surface2)

	InputFocusedStyle = lipgloss.NewStyle().
				Padding(0, 1).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Mauve)

	// Status Bar: une barre solide et contrastée
	StatusBarStyle = lipgloss.NewStyle().
			Background(Surface0).
			Foreground(Subtext0).
			Padding(0, 1)

	// Header: clean
	HeaderStyle = lipgloss.NewStyle().
			Background(Mantle).
			Padding(0, 1).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Surface0)
)

// SetTheme switches the active theme and refreshes all colors and styles.
func SetTheme(name ThemeName) {
	p, ok := Palettes[name]
	if !ok {
		return
	}
	CurrentTheme = name

	Base = p.Base
	Mantle = p.Mantle
	Crust = p.Crust
	Surface0 = p.Surface0
	Surface1 = p.Surface1
	Surface2 = p.Surface2
	Overlay0 = p.Overlay0
	Overlay1 = p.Overlay1
	Overlay2 = p.Overlay2
	Text = p.Text
	Subtext0 = p.Subtext0
	Subtext1 = p.Subtext1
	Rosewater = p.Rosewater
	Flamingo = p.Flamingo
	Pink = p.Pink
	Mauve = p.Mauve
	Red = p.Red
	Maroon = p.Maroon
	Peach = p.Peach
	Yellow = p.Yellow
	Green = p.Green
	Teal = p.Teal
	Sky = p.Sky
	Sapphire = p.Sapphire
	Blue = p.Blue
	Lavender = p.Lavender

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
	UserBubbleStyle = lipgloss.NewStyle().Padding(0, 1).BorderStyle(lipgloss.ThickBorder()).BorderLeft(true).BorderForeground(Mauve).MarginLeft(1)
	AssistantHeaderStyle = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	ThinkingStyle = lipgloss.NewStyle().Foreground(Overlay1).Italic(true).Padding(0, 1).BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(Surface1)
	InputBoxStyle = lipgloss.NewStyle().Padding(0, 1).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(Surface2)
	InputFocusedStyle = lipgloss.NewStyle().Padding(0, 1).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(Mauve)
	StatusBarStyle = lipgloss.NewStyle().Background(Surface0).Foreground(Subtext0).Padding(0, 1)
	HeaderStyle = lipgloss.NewStyle().Background(Mantle).Padding(0, 1).BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(Surface0)
}

func ProviderStyle(provider string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ProviderColor(provider)).Bold(true)
}

func ProviderColor(provider string) color.Color {
	c := config.GetProviderColor(provider)
	if c != "" {
		return lipgloss.Color(c)
	}
	// Dynamic fallback
	return Blue
}
