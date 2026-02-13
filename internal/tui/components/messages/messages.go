package messages

// MessageItem is the interface for all chat message items
type MessageItem interface {
	ID() string
	Role() string
	Render(width int) string
}
