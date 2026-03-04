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
	ThemeMacchiato: {
		Rosewater: lipgloss.Color("#f4dbd6"),
		Flamingo:  lipgloss.Color("#f0c6c6"),
		Pink:      lipgloss.Color("#f5bde6"),
		Mauve:     lipgloss.Color("#c6a0f6"),
		Red:       lipgloss.Color("#ed8796"),
		Maroon:    lipgloss.Color("#ee99a0"),
		Peach:     lipgloss.Color("#f5a97f"),
		Yellow:    lipgloss.Color("#eed49f"),
		Green:     lipgloss.Color("#a6da95"),
		Teal:      lipgloss.Color("#8bd5ca"),
		Sky:       lipgloss.Color("#91d7e3"),
		Sapphire:  lipgloss.Color("#7dc4e4"),
		Blue:      lipgloss.Color("#8aadf4"),
		Lavender:  lipgloss.Color("#b7bdf8"),
		Text:      lipgloss.Color("#cad3f5"),
		Subtext1:  lipgloss.Color("#b8c0e0"),
		Subtext0:  lipgloss.Color("#a5adcb"),
		Overlay2:  lipgloss.Color("#939ab7"),
		Overlay1:  lipgloss.Color("#8087a2"),
		Overlay0:  lipgloss.Color("#6e738d"),
		Surface2:  lipgloss.Color("#5b6078"),
		Surface1:  lipgloss.Color("#494d64"),
		Surface0:  lipgloss.Color("#363a4f"),
		Base:      lipgloss.Color("#24273a"),
		Mantle:    lipgloss.Color("#1e2030"),
		Crust:     lipgloss.Color("#181926"),
	},
	ThemeFrappe: {
		Rosewater: lipgloss.Color("#f2d5cf"),
		Flamingo:  lipgloss.Color("#eebebe"),
		Pink:      lipgloss.Color("#f4b8e4"),
		Mauve:     lipgloss.Color("#ca9ee6"),
		Red:       lipgloss.Color("#e78284"),
		Maroon:    lipgloss.Color("#ea999c"),
		Peach:     lipgloss.Color("#ef9f76"),
		Yellow:    lipgloss.Color("#e5c890"),
		Green:     lipgloss.Color("#a6d189"),
		Teal:      lipgloss.Color("#81c8be"),
		Sky:       lipgloss.Color("#99d1db"),
		Sapphire:  lipgloss.Color("#85c1dc"),
		Blue:      lipgloss.Color("#8caaee"),
		Lavender:  lipgloss.Color("#babbf1"),
		Text:      lipgloss.Color("#c6d0f5"),
		Subtext1:  lipgloss.Color("#b5bfe2"),
		Subtext0:  lipgloss.Color("#a5adce"),
		Overlay2:  lipgloss.Color("#949cbb"),
		Overlay1:  lipgloss.Color("#838ba7"),
		Overlay0:  lipgloss.Color("#737994"),
		Surface2:  lipgloss.Color("#626880"),
		Surface1:  lipgloss.Color("#51576d"),
		Surface0:  lipgloss.Color("#414559"),
		Base:      lipgloss.Color("#303446"),
		Mantle:    lipgloss.Color("#292c3c"),
		Crust:     lipgloss.Color("#232634"),
	},
	ThemeLatte: {
		Rosewater: lipgloss.Color("#dc8a78"),
		Flamingo:  lipgloss.Color("#dd7878"),
		Pink:      lipgloss.Color("#ea76cb"),
		Mauve:     lipgloss.Color("#8839ef"),
		Red:       lipgloss.Color("#d20f39"),
		Maroon:    lipgloss.Color("#e64553"),
		Peach:     lipgloss.Color("#fe640b"),
		Yellow:    lipgloss.Color("#df8e1d"),
		Green:     lipgloss.Color("#40a02b"),
		Teal:      lipgloss.Color("#179299"),
		Sky:       lipgloss.Color("#04a5e5"),
		Sapphire:  lipgloss.Color("#209fb5"),
		Blue:      lipgloss.Color("#1e66f5"),
		Lavender:  lipgloss.Color("#7287fd"),
		Text:      lipgloss.Color("#4c4f69"),
		Subtext1:  lipgloss.Color("#5c5f77"),
		Subtext0:  lipgloss.Color("#6c6f85"),
		Overlay2:  lipgloss.Color("#7c7f93"),
		Overlay1:  lipgloss.Color("#8c8fa1"),
		Overlay0:  lipgloss.Color("#9ca0b0"),
		Surface2:  lipgloss.Color("#acb0be"),
		Surface1:  lipgloss.Color("#bcc0cc"),
		Surface0:  lipgloss.Color("#ccd0da"),
		Base:      lipgloss.Color("#eff1f5"),
		Mantle:    lipgloss.Color("#e6e9ef"),
		Crust:     lipgloss.Color("#dce0e8"),
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

// providerPalette is a cyclic list of Catppuccin accent colors
var providerPalette = []color.Color{
	Mauve, Blue, Green, Peach, Pink, Teal, Yellow, Red,
	Flamingo, Rosewater, Sky, Sapphire, Lavender,
}

// ProviderColorByIndex returns the palette color for a given index (cyclic).
func ProviderColorByIndex(index int) color.Color {
	return providerPalette[index%len(providerPalette)]
}

// Styles
var (
	AppStyle = lipgloss.NewStyle().Background(Base)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(Mauve).
			Bold(true).
			Padding(0, 1)

	HeaderTitleStyle = lipgloss.NewStyle().
				Foreground(Text).
				Bold(true)

	HeaderSubtitleStyle = lipgloss.NewStyle().
				Foreground(Overlay1)

	ChatStyle = lipgloss.NewStyle().Padding(1, 2)

	UserLabelStyle = lipgloss.NewStyle().
			Foreground(Mauve).
			Bold(true)

	AssistantLabelStyle = lipgloss.NewStyle().Bold(true)

	MessageContentStyle = lipgloss.NewStyle().Foreground(Text)

	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Surface2).
			Padding(0, 1)

	InputFocusedStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Mauve).
				Padding(0, 1)

	PromptStyle = lipgloss.NewStyle().
			Foreground(Mauve).
			Bold(true)

	StatusBarStyle = lipgloss.NewStyle().
			Background(Mantle).
			Foreground(Subtext0).
			Padding(0, 1)

	StatusKeyStyle = lipgloss.NewStyle().
			Foreground(Mauve).
			Bold(true)

	StatusValueStyle = lipgloss.NewStyle().Foreground(Text)

	BorderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Surface1)

	ActiveBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Mauve)

	DialogStyle = lipgloss.NewStyle().
			Background(Surface0).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Mauve).
			Padding(1, 2)

	DialogTitleStyle = lipgloss.NewStyle().
				Foreground(Mauve).
				Bold(true)

	ErrorStyle   = lipgloss.NewStyle().Foreground(Red)
	WarningStyle = lipgloss.NewStyle().Foreground(Yellow)
	SuccessStyle = lipgloss.NewStyle().Foreground(Green)
	InfoStyle    = lipgloss.NewStyle().Foreground(Blue)

	HelpKeyStyle  = lipgloss.NewStyle().Foreground(Overlay1)
	HelpDescStyle = lipgloss.NewStyle().Foreground(Overlay0)
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

// NextTheme cycles to the next theme and returns its name.
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

// refreshStyles rebuilds all lipgloss styles from current color variables.
func refreshStyles() {
	AppStyle = lipgloss.NewStyle().Background(Base)
	HeaderStyle = lipgloss.NewStyle().Foreground(Mauve).Bold(true).Padding(0, 1)
	HeaderTitleStyle = lipgloss.NewStyle().Foreground(Text).Bold(true)
	HeaderSubtitleStyle = lipgloss.NewStyle().Foreground(Overlay1)
	UserLabelStyle = lipgloss.NewStyle().Foreground(Mauve).Bold(true)
	MessageContentStyle = lipgloss.NewStyle().Foreground(Text)
	InputStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(Surface2).Padding(0, 1)
	InputFocusedStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(Mauve).Padding(0, 1)
	PromptStyle = lipgloss.NewStyle().Foreground(Mauve).Bold(true)
	StatusBarStyle = lipgloss.NewStyle().Background(Mantle).Foreground(Subtext0).Padding(0, 1)
	StatusKeyStyle = lipgloss.NewStyle().Foreground(Mauve).Bold(true)
	StatusValueStyle = lipgloss.NewStyle().Foreground(Text)
	BorderStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(Surface1)
	ActiveBorderStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(Mauve)
	DialogStyle = lipgloss.NewStyle().Background(Surface0).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(Mauve).Padding(1, 2)
	DialogTitleStyle = lipgloss.NewStyle().Foreground(Mauve).Bold(true)
	ErrorStyle = lipgloss.NewStyle().Foreground(Red)
	WarningStyle = lipgloss.NewStyle().Foreground(Yellow)
	SuccessStyle = lipgloss.NewStyle().Foreground(Green)
	InfoStyle = lipgloss.NewStyle().Foreground(Blue)
	HelpKeyStyle = lipgloss.NewStyle().Foreground(Overlay1)
	HelpDescStyle = lipgloss.NewStyle().Foreground(Overlay0)
}

// ProviderStyle returns the style for a given provider
func ProviderStyle(provider string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ProviderColor(provider)).Bold(true)
}

// ProviderColor returns the color for a given provider.
func ProviderColor(provider string) color.Color {
	c := config.GetProviderColor(provider)
	if c != "" {
		return lipgloss.Color(c)
	}
	names := config.GetProviderNames()
	for i, name := range names {
		if name == provider {
			return ProviderColorByIndex(i)
		}
	}
	return Overlay1
}
