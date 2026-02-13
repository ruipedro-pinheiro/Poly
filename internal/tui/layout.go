package tui

import "github.com/pedromelo/poly/internal/tui/layout"

// LayoutContext provides computed dimensions for all components.
// Recalculated on every WindowSizeMsg.
type LayoutContext struct {
	ScreenWidth  int
	ScreenHeight int

	// Horizontal zones
	SidebarWidth int // 0 if hidden
	ChatWidth    int // screen - sidebar
	ContentWidth int // chat - borders - padding

	// Vertical zones
	HeaderHeight int
	StatusHeight int
	ChatHeight   int // remaining after header + status + editor
	EditorHeight int

	// Viewport (what the chat scrollable area gets)
	ViewportWidth  int
	ViewportHeight int

	// Dialog dimensions
	DialogWidth  int
	DialogHeight int
}

// ComputeLayout calculates all dimensions from screen size and sidebar state.
func ComputeLayout(screenW, screenH int, sidebarVisible bool) LayoutContext {
	lc := LayoutContext{
		ScreenWidth:  screenW,
		ScreenHeight: screenH,
		HeaderHeight: layout.HeaderHeight,
		StatusHeight: layout.StatusHeight,
		EditorHeight: layout.InputHeight,
	}

	// Sidebar width
	if sidebarVisible && screenW > layout.SidebarMinScreenW {
		lc.SidebarWidth = layout.SidebarWidth
	}

	// Chat width = everything minus sidebar
	lc.ChatWidth = screenW - lc.SidebarWidth

	// Content width = chat minus borders and padding
	lc.ContentWidth = lc.ChatWidth - layout.ChatAreaPadding - layout.ContentPadding
	if lc.ContentWidth < layout.ContentMinWidth {
		lc.ContentWidth = layout.ContentMinWidth
	}

	// Chat height = screen minus fixed zones
	lc.ChatHeight = screenH - lc.HeaderHeight - lc.StatusHeight - lc.EditorHeight
	if lc.ChatHeight < 1 {
		lc.ChatHeight = 1
	}

	// Viewport dimensions
	lc.ViewportWidth = lc.ChatWidth - layout.ChatAreaPadding
	if lc.ViewportWidth < layout.ContentMinWidth {
		lc.ViewportWidth = layout.ContentMinWidth
	}
	lc.ViewportHeight = lc.ChatHeight

	// Dialog dimensions
	lc.DialogWidth = int(float64(screenW) * layout.DialogWidthRatio)
	if lc.DialogWidth > layout.DialogMaxWidth {
		lc.DialogWidth = layout.DialogMaxWidth
	}
	if lc.DialogWidth < layout.DialogMinWidth {
		lc.DialogWidth = layout.DialogMinWidth
	}
	lc.DialogHeight = screenH

	return lc
}
