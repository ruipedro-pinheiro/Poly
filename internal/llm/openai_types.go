package llm

import "encoding/base64"

// OpenAI-compatible API types.
// Used by GPT, Grok, Ollama, and CustomProvider (openai format).
// All JSON tags must exactly match the OpenAI API specification.

// --- Request Types ---

// OAIMessage represents a message in the OpenAI chat completions format.
type OAIMessage struct {
	Role      string            `json:"role"`
	Content   interface{}       `json:"content"`             // string or []OAIContentPart
	ToolCalls []OAIToolCallMsg  `json:"tool_calls,omitempty"` // for assistant messages
	ToolUseID string            `json:"tool_call_id,omitempty"`
}

// OAIContentPart represents a single part of a multi-modal message.
type OAIContentPart struct {
	Type     string       `json:"type"` // "text" or "image_url"
	Text     string       `json:"text,omitempty"`
	ImageURL *OAIImageURL `json:"image_url,omitempty"`
}

// OAIImageURL holds a base64-encoded image data URL.
type OAIImageURL struct {
	URL string `json:"url"` // "data:image/png;base64,..."
}

// OAIToolCallMsg represents an LLM's request to call a tool.
type OAIToolCallMsg struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"` // "function"
	Function OAIToolCallFunc `json:"function"`
}

// OAIToolCallFunc holds the function name and arguments.
type OAIToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// OAIToolDef is a tool definition in OpenAI format.
type OAIToolDef struct {
	Type     string          `json:"type"` // "function"
	Function OAIToolFunction `json:"function"`
}

// OAIToolFunction describes the function's metadata.
type OAIToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// OAIStreamOptions configures additional streaming behavior.
type OAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// OAIRequestBody is the top-level request for the OpenAI chat completions API.
type OAIRequestBody struct {
	Model               string            `json:"model"`
	Stream              bool              `json:"stream"`
	Messages            []OAIMessage      `json:"messages"`
	StreamOptions       *OAIStreamOptions `json:"stream_options,omitempty"`
	MaxTokens           int               `json:"max_tokens,omitempty"`
	Tools               []OAIToolDef      `json:"tools,omitempty"`
	ReasoningEffort     string            `json:"reasoning_effort,omitempty"`
	MaxCompletionTokens int               `json:"max_completion_tokens,omitempty"`
}

// --- Helper constructors ---

// NewOAITextMessage creates a simple text message.
func NewOAITextMessage(role, content string) OAIMessage {
	return OAIMessage{Role: role, Content: content}
}

// NewOAIAssistantMessage creates an assistant message with tool calls.
func NewOAIAssistantMessage(content string, toolCalls []OAIToolCallMsg) OAIMessage {
	return OAIMessage{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	}
}

// NewOAIToolResultMessage creates a message with a tool execution result.
func NewOAIToolResultMessage(toolCallID, content string) OAIMessage {
	return OAIMessage{
		Role:      "tool",
		ToolUseID: toolCallID,
		Content:   content,
	}
}

// OAIToolDefsFromPoly converts Poly ToolDefinitions to OpenAI format.
func OAIToolDefsFromPoly(defs []ToolDefinition) []OAIToolDef {
	if len(defs) == 0 {
		return nil
	}
	result := make([]OAIToolDef, len(defs))
	for i, d := range defs {
		result[i] = OAIToolDef{
			Type: "function",
			Function: OAIToolFunction{
				Name:        d.Name,
				Description: d.Description,
				Parameters:  d.InputSchema,
			},
		}
	}
	return result
}

// BuildOAIImageParts creates content parts from a Poly Message with images.
// Returns nil if the message should be skipped.
func BuildOAIImageParts(msg Message) interface{} {
	// Special cases for messages with tools/results
	if msg.ToolResult != nil || len(msg.ToolCalls) > 0 {
		return msg.Content
	}

	if len(msg.Images) == 0 {
		if msg.Content == "" {
			return nil
		}
		return msg.Content
	}

	parts := make([]OAIContentPart, 0, len(msg.Images)+1)
	for _, img := range msg.Images {
		dataURL := "data:" + img.MediaType + ";base64," + base64.StdEncoding.EncodeToString(img.Data)
		parts = append(parts, OAIContentPart{
			Type:     "image_url",
			ImageURL: &OAIImageURL{URL: dataURL},
		})
	}
	if msg.Content != "" {
		parts = append(parts, OAIContentPart{
			Type: "text",
			Text: msg.Content,
		})
	}

	if len(parts) == 0 {
		return nil
	}
	return parts
}
