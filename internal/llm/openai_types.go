package llm

import "encoding/base64"

// OpenAI-compatible API types.
// Used by GPT, Grok, Ollama, and CustomProvider (openai format).
// All JSON tags must exactly match the OpenAI API specification.

// --- Request Types ---

// OAIMessage represents a message in OpenAI chat completion format.
// Content can be a string (text-only) or a slice of OAIContentPart (multimodal).
type OAIMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"` // string or []OAIContentPart
	ToolCalls  []OAIToolCallMsg `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"` // present when role="tool"
}

// OAIContentPart is a content block within a multimodal message.
type OAIContentPart struct {
	Type     string       `json:"type"` // "text" or "image_url"
	Text     string       `json:"text,omitempty"`
	ImageURL *OAIImageURL `json:"image_url,omitempty"`
}

// OAIImageURL wraps an image URL for vision-capable models.
type OAIImageURL struct {
	URL string `json:"url"`
}

// OAIToolDef wraps a function tool definition.
type OAIToolDef struct {
	Type     string         `json:"type"` // always "function"
	Function OAIFunctionDef `json:"function"`
}

// OAIFunctionDef describes a callable function tool.
type OAIFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// OAIToolCallMsg represents a tool call within an assistant message.
type OAIToolCallMsg struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"` // always "function"
	Function OAIToolCallFunc `json:"function"`
}

// OAIToolCallFunc holds the function name and its serialized arguments.
type OAIToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// OAIStreamOptions configures streaming behavior.
type OAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// OAIRequestBody is the top-level request for OpenAI-compatible chat completions.
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

// NewOAIToolResultMessage creates a tool result message.
func NewOAIToolResultMessage(toolCallID, content string) OAIMessage {
	return OAIMessage{Role: "tool", ToolCallID: toolCallID, Content: content}
}

// NewOAIAssistantMessage creates an assistant message with optional tool calls.
func NewOAIAssistantMessage(content string, toolCalls []OAIToolCallMsg) OAIMessage {
	msg := OAIMessage{Role: "assistant"}
	if content != "" {
		msg.Content = content
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}
	return msg
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
			Function: OAIFunctionDef{
				Name:        d.Name,
				Description: d.Description,
				Parameters:  d.InputSchema,
			},
		}
	}
	return result
}

// BuildOAIImageParts creates multimodal content parts from a Poly Message.
func BuildOAIImageParts(msg Message) interface{} {
	if len(msg.Images) == 0 {
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
	parts = append(parts, OAIContentPart{
		Type: "text",
		Text: msg.Content,
	})
	return parts
}
