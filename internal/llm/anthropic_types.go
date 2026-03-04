package llm

import "encoding/base64"

// Anthropic API types.
// Used by AnthropicProvider and CustomProvider (anthropic format).
// JSON tags match the Anthropic Messages API specification.

// --- Request Types ---

// AntMessage represents a message in the Anthropic Messages API format.
// Content can be a string (simple text) or a []AntContentBlock (structured).
type AntMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []AntContentBlock
}

// AntContentBlock is a polymorphic content block within a message.
// Only the fields relevant to the block's Type are serialized (via omitempty).
type AntContentBlock struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text,omitempty"`
	Thinking     string                 `json:"thinking,omitempty"`
	Signature    string                 `json:"signature,omitempty"`
	ID           string                 `json:"id,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Input        map[string]interface{} `json:"input,omitempty"`
	ToolUseID    string                 `json:"tool_use_id,omitempty"`
	Content      string                 `json:"content,omitempty"`
	IsError      bool                   `json:"is_error,omitempty"`
	Source       *AntImageSource        `json:"source,omitempty"`
	CacheControl map[string]string      `json:"cache_control,omitempty"`
}

// AntImageSource describes a base64-encoded image.
type AntImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // e.g. "image/png"
	Data      string `json:"data"`       // base64-encoded
}

// AntToolDef is a tool definition in Anthropic format.
type AntToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `description:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// AntThinkingConfig enables extended thinking.
type AntThinkingConfig struct {
	Type         string `json:"type"` // "enabled"
	BudgetTokens int    `json:"budget_tokens"`
}

// AntRequestBody is the top-level request for the Anthropic Messages API.
type AntRequestBody struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream"`
	Messages  []AntMessage       `json:"messages"`
	System    interface{}        `json:"system,omitempty"` // string or []AntContentBlock
	Tools     []AntToolDef       `json:"tools,omitempty"`
	Thinking  *AntThinkingConfig `json:"thinking,omitempty"`
}

// --- Helper constructors ---

// NewAntTextMessage creates a simple text message.
func NewAntTextMessage(role, content string) AntMessage {
	return AntMessage{Role: role, Content: content}
}

// AntToolDefsFromPoly converts Poly ToolDefinitions to Anthropic format.
func AntToolDefsFromPoly(defs []ToolDefinition, isOAuth bool) []AntToolDef {
	if len(defs) == 0 {
		return nil
	}
	result := make([]AntToolDef, len(defs))
	for i, d := range defs {
		name := d.Name
		if isOAuth {
			name = mcpToolPrefix + name
		}
		result[i] = AntToolDef{
			Name:        name,
			Description: d.Description,
			InputSchema: d.InputSchema,
		}
	}
	return result
}

// BuildAntImageContent creates content blocks from a Poly Message with images.
// Returns nil if the message should be skipped.
func BuildAntImageContent(msg Message) interface{} {
	// For messages with tool results
	if msg.ToolResult != nil {
		return []AntContentBlock{NewAntToolResultBlock(msg.ToolResult.ToolUseID, msg.ToolResult.Content, msg.ToolResult.IsError)}
	}

	// For messages with tool calls
	if len(msg.ToolCalls) > 0 {
		blocks := make([]AntContentBlock, 0, len(msg.ToolCalls)+1)
		if msg.Content != "" {
			blocks = append(blocks, AntContentBlock{Type: "text", Text: msg.Content})
		}
		for _, tc := range msg.ToolCalls {
			blocks = append(blocks, AntContentBlock{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Arguments,
			})
		}
		return blocks
	}

	// For regular messages (with or without images)
	if len(msg.Images) == 0 {
		if msg.Content == "" {
			return nil
		}
		return msg.Content
	}

	blocks := make([]AntContentBlock, 0, len(msg.Images)+1)
	for _, img := range msg.Images {
		blocks = append(blocks, AntContentBlock{
			Type: "image",
			Source: &AntImageSource{
				Type:      "base64",
				MediaType: img.MediaType,
				Data:      base64.StdEncoding.EncodeToString(img.Data),
			},
		})
	}
	if msg.Content != "" {
		blocks = append(blocks, AntContentBlock{
			Type: "text",
			Text: msg.Content,
		})
	}

	if len(blocks) == 0 {
		return nil
	}
	return blocks
}

// NewAntToolResultBlock creates a tool result content block.
func NewAntToolResultBlock(toolUseID, content string, isError bool) AntContentBlock {
	return AntContentBlock{
		Type:      "tool_result",
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}
}
