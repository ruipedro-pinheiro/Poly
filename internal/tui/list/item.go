package list

// Item is the interface for list items that can be rendered
type Item interface {
	Render(width int) string
}

// RawRenderable allows items to provide pre-rendered content
type RawRenderable interface {
	RawRender(width int) string
}

// Focusable allows items to respond to focus changes
type Focusable interface {
	SetFocused(focused bool)
}

// Highlightable allows items to have highlighted regions
type Highlightable interface {
	SetHighlight(startLine, startCol, endLine, endCol int)
	Highlight() (startLine, startCol, endLine, endCol int)
}

// RenderCallback transforms items before rendering
type RenderCallback func(idx, selectedIdx int, item Item) Item
