package tui

import "github.com/pedromelo/poly/internal/tui/layout"

// LayoutContext provides computed dimensions for all components.
// Recalculated on every WindowSizeMsg.
type LayoutContext struct {
	ScreenWidth  int
	ScreenHeight int

	// Horizontal zones
	ChatWidth    int // full screen width
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

// ComputeLayout calculates all dimensions from screen size.
func ComputeLayout(screenW, screenH int) LayoutContext {
	return ComputeLayoutWithEditor(screenW, screenH, layout.InputHeight)
}

// ComputeLayoutWithEditor calculates all dimensions with a dynamic editor height.
func ComputeLayoutWithEditor(screenW, screenH int, editorH int) LayoutContext {
	lc := LayoutContext{
		ScreenWidth:  screenW,
		ScreenHeight: screenH,
		HeaderHeight: layout.HeaderHeight,
		StatusHeight: layout.StatusHeight,
		EditorHeight: editorH,
	}

	// Chat width = full screen
	lc.ChatWidth = screenW

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
