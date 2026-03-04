package styles

import (
	"fmt"
	"image/color"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/pedromelo/poly/internal/theme"
)

// ColorToHex converts an image/color.Color to a hex string like "#cba6f7".
// Used by markdown rendering and gradient functions to bridge the color types.
func ColorToHex(c color.Color) string {
	cf, ok := colorful.MakeColor(c)
	if ok {
		return cf.Hex()
	}
	// Fallback: manual RGBA conversion
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

// Re-export types and constants from theme for convenience
type ThemeName = theme.ThemeName

const (
	ThemeMocha     = theme.ThemeMocha
	ThemeMacchiato = theme.ThemeMacchiato
	ThemeFrappe    = theme.ThemeFrappe
	ThemeLatte     = theme.ThemeLatte
)

var AllThemes = theme.AllThemes

// SetTheme is now a wrapper around theme.SetTheme
func SetTheme(name ThemeName) {
	theme.SetTheme(name)
}

// NextTheme is now a wrapper around theme.NextTheme
func NextTheme() ThemeName {
	return theme.NextTheme()
}
