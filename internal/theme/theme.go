package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/tui/styles"
)

// Catppuccin Mocha Mauve palette (defaults, updated by SetTheme)
var (
	// Base colors
	Base     color.Color = lipgloss.Color("#1e1e2e")
	Mantle   color.Color = lipgloss.Color("#181825")
	Crust    color.Color = lipgloss.Color("#11111b")
	Surface0 color.Color = lipgloss.Color("#313244")
	Surface1 color.Color = lipgloss.Color("#45475a")
	Surface2 color.Color = lipgloss.Color("#585b70")
	Overlay0 color.Color = lipgloss.Color("#6c7086")
	Overlay1 color.Color = lipgloss.Color("#7f849c")
	Overlay2 color.Color = lipgloss.Color("#9399b2")

	// Text colors
	Text     color.Color = lipgloss.Color("#cdd6f4")
	Subtext0 color.Color = lipgloss.Color("#a6adc8")
	Subtext1 color.Color = lipgloss.Color("#bac2de")

	// Accent colors
	Rosewater color.Color = lipgloss.Color("#f5e0dc")
	Flamingo  color.Color = lipgloss.Color("#f2cdcd")
	Pink      color.Color = lipgloss.Color("#f5c2e7")
	Mauve     color.Color = lipgloss.Color("#cba6f7")
	Red       color.Color = lipgloss.Color("#f38ba8")
	Maroon    color.Color = lipgloss.Color("#eba0ac")
	Peach     color.Color = lipgloss.Color("#fab387")
	Yellow    color.Color = lipgloss.Color("#f9e2af")
	Green     color.Color = lipgloss.Color("#a6e3a1")
	Teal      color.Color = lipgloss.Color("#94e2d5")
	Sky       color.Color = lipgloss.Color("#89dceb")
	Sapphire  color.Color = lipgloss.Color("#74c7ec")
	Blue      color.Color = lipgloss.Color("#89b4fa")
	Lavender  color.Color = lipgloss.Color("#b4befe")

)

// providerPalette is a cyclic list of Catppuccin accent colors assigned to
// providers in alphabetical order. When a provider has no config override,
// it gets the color at its index (mod palette length).
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
	// App styles
	AppStyle = lipgloss.NewStyle().
			Background(Base)

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Mauve).
			Bold(true).
			Padding(0, 1)

	HeaderTitleStyle = lipgloss.NewStyle().
				Foreground(Text).
				Bold(true)

	HeaderSubtitleStyle = lipgloss.NewStyle().
				Foreground(Overlay1)

	// Chat area
	ChatStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Message styles
	UserLabelStyle = lipgloss.NewStyle().
			Foreground(Mauve).
			Bold(true)

	AssistantLabelStyle = lipgloss.NewStyle().
				Bold(true)

	MessageContentStyle = lipgloss.NewStyle().
				Foreground(Text)

	// Input
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

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Background(Mantle).
			Foreground(Subtext0).
			Padding(0, 1)

	StatusKeyStyle = lipgloss.NewStyle().
			Foreground(Mauve).
			Bold(true)

	StatusValueStyle = lipgloss.NewStyle().
				Foreground(Text)

	// Borders
	BorderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Surface1)

	ActiveBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Mauve)

	// Dialogs/Modals
	DialogStyle = lipgloss.NewStyle().
			Background(Surface0).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Mauve).
			Padding(1, 2)

	DialogTitleStyle = lipgloss.NewStyle().
				Foreground(Mauve).
				Bold(true)

	// Error/Warning/Success
	ErrorStyle = lipgloss.NewStyle().
			Foreground(Red)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Yellow)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Green)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Blue)

	// Help
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(Overlay1)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(Overlay0)
)

// SetTheme switches the active theme and refreshes all colors and styles.
func SetTheme(name styles.ThemeName) {
	// Update styles package colors
	styles.SetTheme(name)

	// Sync our local color variables from styles
	p := styles.Palettes[name]
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

	// Rebuild all styles with new colors
	refreshStyles()
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
// Priority: config override > cyclic palette based on alphabetical position.
func ProviderColor(provider string) color.Color {
	c := config.GetProviderColor(provider)
	if c != "" {
		return lipgloss.Color(c)
	}
	// Find alphabetical index among all configured providers
	names := config.GetProviderNames()
	for i, name := range names {
		if name == provider {
			return ProviderColorByIndex(i)
		}
	}
	return Overlay1
}
