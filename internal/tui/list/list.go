package list

import (
	"strings"
)

// List is a high-performance virtualized list that only renders visible items
type List struct {
	width, height int
	items         []Item
	gap           int
	reverse       bool // newest at bottom (chat mode)

	// Scrolling
	focused     bool
	selectedIdx int
	offsetIdx   int // first visible item index
	offsetLine  int // line offset within first visible item

	// Cache
	renderedItems map[int]renderedItem
	rendered      string
	dirty         bool

	// Callbacks
	renderCallbacks []RenderCallback
}

type renderedItem struct {
	view   string
	height int
}

// NewList creates a new virtualized list
func NewList(items ...Item) *List {
	l := &List{
		items:         items,
		renderedItems: make(map[int]renderedItem),
		dirty:         true,
		gap:           1,
	}
	return l
}

// SetSize sets the viewport dimensions
func (l *List) SetSize(width, height int) {
	if l.width != width || l.height != height {
		l.width = width
		l.height = height
		l.invalidateCache()
	}
}

// SetGap sets the spacing between items
func (l *List) SetGap(gap int) {
	if l.gap != gap {
		l.gap = gap
		l.dirty = true
	}
}

// SetReverse sets reverse mode (newest at bottom, for chat)
func (l *List) SetReverse(reverse bool) {
	if l.reverse != reverse {
		l.reverse = reverse
		l.dirty = true
	}
}

// SetItems replaces all items
func (l *List) SetItems(items ...Item) {
	l.items = items
	l.invalidateCache()
	if l.reverse {
		l.scrollToEnd()
	}
}

// UpdateItem updates a single item at index
func (l *List) UpdateItem(idx int, item Item) {
	if idx >= 0 && idx < len(l.items) {
		l.items[idx] = item
		delete(l.renderedItems, idx)
		l.dirty = true
	}
}

// AppendItem adds an item to the end
func (l *List) AppendItem(item Item) {
	wasAtBottom := l.AtBottom()
	l.items = append(l.items, item)
	l.dirty = true
	if l.reverse && wasAtBottom {
		l.scrollToEnd()
	}
}

// Len returns the number of items
func (l *List) Len() int {
	return len(l.items)
}

// SetFocused sets the focus state
func (l *List) SetFocused(focused bool) {
	l.focused = focused
}

// ScrollToBottom scrolls to the bottom
func (l *List) ScrollToBottom() {
	l.scrollToEnd()
}

// ScrollToIndex scrolls to make the given index visible
func (l *List) ScrollToIndex(index int) {
	if index < 0 || index >= len(l.items) {
		return
	}
	l.offsetIdx = index
	l.offsetLine = 0
	l.dirty = true
}

// ScrollBy scrolls by the given number of lines
func (l *List) ScrollBy(lines int) {
	if lines > 0 {
		l.scrollDown(lines)
	} else if lines < 0 {
		l.scrollUp(-lines)
	}
}

// AtBottom returns true if the list is scrolled to the bottom
func (l *List) AtBottom() bool {
	if len(l.items) == 0 {
		return true
	}

	// Calculate total content height from offset to end
	remaining := 0
	for i := l.offsetIdx; i < len(l.items); i++ {
		ri := l.getRenderItem(i)
		remaining += ri.height
		if i < len(l.items)-1 {
			remaining += l.gap
		}
	}
	remaining -= l.offsetLine

	return remaining <= l.height
}

// VisibleItemIndices returns the range of currently visible items
func (l *List) VisibleItemIndices() (startIdx, endIdx int) {
	if len(l.items) == 0 || l.height == 0 {
		return 0, 0
	}

	startIdx = l.offsetIdx
	linesUsed := -l.offsetLine
	endIdx = startIdx

	for i := startIdx; i < len(l.items); i++ {
		ri := l.getRenderItem(i)
		linesUsed += ri.height
		if i > startIdx {
			linesUsed += l.gap
		}
		endIdx = i + 1
		if linesUsed >= l.height {
			break
		}
	}

	return startIdx, endIdx
}

// RegisterRenderCallback adds a callback that transforms items before rendering
func (l *List) RegisterRenderCallback(cb RenderCallback) {
	l.renderCallbacks = append(l.renderCallbacks, cb)
}

// Render returns the visible portion of the list as a string
func (l *List) Render() string {
	if l.width == 0 || l.height == 0 || len(l.items) == 0 {
		return ""
	}

	var lines []string
	linesUsed := 0

	startIdx := l.offsetIdx
	skipLines := l.offsetLine

	for i := startIdx; i < len(l.items); i++ {
		ri := l.getRenderItem(i)
		itemLines := strings.Split(ri.view, "\n")

		// Skip lines for partial first item
		start := 0
		if i == startIdx && skipLines > 0 {
			start = skipLines
			if start >= len(itemLines) {
				continue
			}
		}

		// Add gap between items
		if i > startIdx && linesUsed > 0 {
			for g := 0; g < l.gap && linesUsed < l.height; g++ {
				lines = append(lines, "")
				linesUsed++
			}
		}

		for j := start; j < len(itemLines) && linesUsed < l.height; j++ {
			lines = append(lines, itemLines[j])
			linesUsed++
		}

		if linesUsed >= l.height {
			break
		}
	}

	l.dirty = false
	l.rendered = strings.Join(lines, "\n")
	return l.rendered
}

// internal methods

func (l *List) getRenderItem(idx int) renderedItem {
	if ri, ok := l.renderedItems[idx]; ok {
		return ri
	}

	item := l.items[idx]

	// Apply callbacks
	for _, cb := range l.renderCallbacks {
		item = cb(idx, l.selectedIdx, item)
	}

	view := item.Render(l.width)
	height := strings.Count(view, "\n") + 1

	ri := renderedItem{view: view, height: height}
	l.renderedItems[idx] = ri
	return ri
}

func (l *List) invalidateCache() {
	l.renderedItems = make(map[int]renderedItem)
	l.dirty = true
}

func (l *List) scrollToEnd() {
	if len(l.items) == 0 {
		l.offsetIdx = 0
		l.offsetLine = 0
		return
	}

	// Walk backwards from the end, accumulating height
	totalHeight := 0
	startIdx := len(l.items) - 1

	for i := len(l.items) - 1; i >= 0; i-- {
		ri := l.getRenderItem(i)
		needed := ri.height
		if i < len(l.items)-1 {
			needed += l.gap
		}

		if totalHeight+needed > l.height {
			startIdx = i
			// Calculate line offset within this item
			overflow := totalHeight + needed - l.height
			l.offsetIdx = startIdx
			l.offsetLine = overflow
			l.dirty = true
			return
		}
		totalHeight += needed
		startIdx = i
	}

	l.offsetIdx = startIdx
	l.offsetLine = 0
	l.dirty = true
}

func (l *List) scrollDown(lines int) {
	for i := 0; i < lines; i++ {
		if l.offsetIdx >= len(l.items) {
			return
		}
		ri := l.getRenderItem(l.offsetIdx)
		l.offsetLine++
		if l.offsetLine >= ri.height {
			l.offsetIdx++
			l.offsetLine = 0
			// Skip gap
		}
	}
	l.dirty = true
}

func (l *List) scrollUp(lines int) {
	for i := 0; i < lines; i++ {
		if l.offsetIdx == 0 && l.offsetLine == 0 {
			return
		}
		l.offsetLine--
		if l.offsetLine < 0 {
			l.offsetIdx--
			if l.offsetIdx < 0 {
				l.offsetIdx = 0
				l.offsetLine = 0
				return
			}
			ri := l.getRenderItem(l.offsetIdx)
			l.offsetLine = ri.height - 1
		}
	}
	l.dirty = true
}
