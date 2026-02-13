package tui

import "github.com/pedromelo/poly/internal/llm"

// viewState represents the current UI view
type viewState int

const (
	viewSplash viewState = iota
	viewChat
	viewModelPicker
	viewControlRoom
	viewHelp
	viewAddProvider
	viewCommandPalette
	viewSessionList
	viewApproval
)

// ToolCallData holds structured data for a tool call and its result
type ToolCallData struct {
	Name    string
	Args    map[string]interface{}
	Result  string
	IsError bool
	Status  int // 0=pending, 1=running, 2=success, 3=error
}

// ContentBlock represents an ordered piece of message content (text or tool)
type ContentBlock struct {
	Type    string // "text" or "tool"
	Text    string // content for "text" blocks
	ToolIdx int    // index into ToolCalls for "tool" blocks
}

// Message represents a chat message
type Message struct {
	Role       string   // "user" or provider name
	Content    string   // full text (for persistence)
	Provider   string   // which provider responded
	Thinking   string   // thinking content (for extended thinking models)
	Images     []string // file paths for persisted images
	ImageData  [][]byte // raw image data (for pasted images, not persisted)
	ImageTypes []string // media types for ImageData
	ToolCalls  []ToolCallData  // structured tool call data
	Blocks     []ContentBlock  // ordered interleaved content (text + tools)
}

// modelOption represents a selectable model in the picker
type modelOption struct {
	provider string
	variant  string
	display  string
}

// StreamMsg is sent when we receive streaming content
type StreamMsg struct {
	Content             string
	Thinking            string
	Done                bool
	Error               error
	Provider            string
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	ToolCall            *llm.ToolCall   // tool_use event
	ToolResult          *llm.ToolResult // tool_result event
}

// CascadePhase tracks which phase of the @all cascade we're in
type CascadePhase int

const (
	CascadeResponder CascadePhase = iota
	CascadeReviewer
)

// CascadeStreamMsg is sent during @all cascade streaming
type CascadeStreamMsg struct {
	Provider   string
	Content    string
	Thinking   string
	Done       bool
	Error      error
	Phase      CascadePhase
	ToolCall   *llm.ToolCall   // tool_use event
	ToolResult *llm.ToolResult // tool_result event
}

// cascadeState tracks @all cascade orchestration state
type cascadeState struct {
	responder        string            // provider name of the first responder
	responderContent string            // accumulated response from the responder
	reviewers        []string          // reviewer provider names
	activeReviewers  map[string]bool   // tracks which reviewers are still streaming
	messageIndices   map[string]int    // provider -> message index
	phase            CascadePhase      // current phase
	userQuestion     string            // original user question
	userImages       []llm.Image       // original user images (for reviewers)
}

// OAuthResultMsg is sent when OAuth completes
type OAuthResultMsg struct {
	Provider string
	Success  bool
	Error    string
}

// StreamTickMsg is sent periodically during streaming to update elapsed time display
type StreamTickMsg struct{}

// CompactMsg triggers context compaction
type CompactMsg struct{}

// CompactDoneMsg is sent when compaction finishes
type CompactDoneMsg struct {
	Messages []llm.Message
	Error    error
}

// UpdateAvailableMsg is sent when a newer version of Poly is found on GitHub
type UpdateAvailableMsg struct {
	Version string
}

// CompareResultMsg is sent when a single provider finishes its /compare response
type CompareResultMsg struct {
	Provider    string
	Model       string
	Content     string
	Error       error
	ElapsedMs   int64
	Index       int // 0-based index in the compare sequence
	Total       int // total providers being compared
}

// CommandEntry represents a command in the command palette
type CommandEntry struct {
	Name     string
	Shortcut string
	Action   func(m *Model)
}
