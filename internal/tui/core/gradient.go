package core

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/lucasb-eyer/go-colorful"
)

// GradientText renders each character of text with a smooth color gradient
// between two colors. Uses HCL blending for perceptually uniform transitions.
// Spaces are passed through without styling to avoid background color artifacts.
func GradientText(text string, from, to color.Color, bold bool) string {
	if text == "" {
		return ""
	}
	c1, _ := colorful.MakeColor(from)
	c2, _ := colorful.MakeColor(to)
	runes := []rune(text)
	var b strings.Builder
	for i, r := range runes {
		if r == ' ' {
			b.WriteRune(' ')
			continue
		}
		t := float64(i) / math.Max(1, float64(len(runes)-1))
		c := c1.BlendHcl(c2, t).Clamped()
		style := lipgloss.NewStyle().Foreground(c)
		if bold {
			style = style.Bold(true)
		}
		b.WriteString(style.Render(string(r)))
	}
	return b.String()
}
