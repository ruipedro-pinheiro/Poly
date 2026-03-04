package core

import (
	"os"
	"strings"
)

// Icons used throughout the TUI.
//
// Default set avoids ambiguous circular glyphs.
// Nerd-font-like symbols are intentionally not used for provider/status markers.
var (
	IconCheck    = "󰄬"
	IconError    = "󰅖"
	IconWarning  = "󰀪"
	IconInfo     = "󰋽"
	IconProvider = "󰐕"
	IconPending  = "…"
	IconModel    = "󰭻"
	IconArrow    = "󰁔"
	IconBorder   = "▌"
	IconDiag     = "┈"
	IconSep      = "─"
	IconLoading  = "󰔟"
	IconTodo     = "󰄱"
	IconDone     = "󰄬"
	IconActive   = "󰘥"
)

func init() {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("POLY_ICON_SET")))
	if v == "unicode" {
		setUnicodeIcons()
	}
}

func setUnicodeIcons() {
	IconCheck = "✓"
	IconError = "✕"
	IconWarning = "⚠"
	IconInfo = "ⓘ"
	IconProvider = ">"
	IconPending = "…"
	IconModel = "◇"
	IconArrow = "›"
	IconBorder = "▌"
	IconDiag = "┈"
	IconSep = "─"
	IconLoading = "⟳"
	IconTodo = "•"
	IconDone = "✓"
	IconActive = "▸"
}
