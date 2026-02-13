package tools

// ToolStatus represents the execution state of a tool call
type ToolStatus int

const (
	ToolStatusPending ToolStatus = iota
	ToolStatusRunning
	ToolStatusSuccess
	ToolStatusError
	ToolStatusDenied
)

// RenderOpts holds the data needed to render a tool call
type RenderOpts struct {
	Name    string
	Args    map[string]interface{}
	Result  string
	IsError bool
	Status  ToolStatus
	Compact bool
}

// ToolRenderer renders a tool call into a styled string
type ToolRenderer interface {
	Render(width int, opts *RenderOpts) string
}
