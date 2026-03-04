package llm

import "encoding/base64"

// Google Gemini API types.
// Used by GeminiProvider and CustomProvider (google format).
// JSON tags match the Gemini REST / Code Assist API specification.

// --- Request Types ---

// GemContent represents a single message (content entry) in the Gemini format.
type GemContent struct {
	Role  string    `json:"role"`
	Parts []GemPart `json:"parts"`
}

// GemPart is a polymorphic part within a content entry.
// Only the fields relevant to the part type are serialized (via omitempty).
type GemPart struct {
	Text             string               `json:"text,omitempty"`
	InlineData       *GemInlineData       `json:"inlineData,omitempty"`
	FunctionCall     *GemFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *GemFunctionResponse `json:"functionResponse,omitempty"`
}

// GemInlineData wraps base64-encoded binary data (images, etc.).
type GemInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// GemFunctionCall represents a function call from the model.
type GemFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"` // dynamic — tool call arguments
}

// GemFunctionResponse wraps a tool execution result to send back to the model.
type GemFunctionResponse struct {
	Name     string            `json:"name"`
	Response GemFunctionResult `json:"response"`
}

// GemFunctionResult is the inner result content of a function response.
type GemFunctionResult struct {
	Content string `json:"content"`
}

// GemFunctionDeclaration defines a tool for the Gemini API.
type GemFunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"` // JSON schema — dynamic
}

// GemToolGroup wraps a set of function declarations.
// Gemini expects tools as an array of groups.
type GemToolGroup struct {
	FunctionDeclarations []GemFunctionDeclaration `json:"functionDeclarations"`
}

// GemGenerationConfig holds generation parameters.
type GemGenerationConfig struct {
	MaxOutputTokens int                `json:"maxOutputTokens,omitempty"`
	ThinkingConfig  *GemThinkingConfig `json:"thinkingConfig,omitempty"`
}

// GemThinkingConfig enables extended thinking.
type GemThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

// GemRequestBody is the top-level request body for the public Gemini API.
type GemRequestBody struct {
	Contents         []GemContent         `json:"contents"`
	GenerationConfig *GemGenerationConfig `json:"generationConfig,omitempty"`
	Tools            []GemToolGroup       `json:"tools,omitempty"`
}

// GemCodeAssistInnerRequest is the inner "request" field for Code Assist.
type GemCodeAssistInnerRequest struct {
	Contents         []GemContent         `json:"contents"`
	SessionID        string               `json:"session_id"`
	GenerationConfig *GemGenerationConfig `json:"generationConfig,omitempty"`
	Tools            []GemToolGroup       `json:"tools,omitempty"`
}

// GemCodeAssistBody is the outer envelope for Code Assist requests.
type GemCodeAssistBody struct {
	Model        string                    `json:"model"`
	Project      string                    `json:"project"`
	UserPromptID string                    `json:"user_prompt_id"`
	Request      GemCodeAssistInnerRequest `json:"request"`
}

// GemLoadCodeAssistBody is the request body for resolving a Code Assist project.
type GemLoadCodeAssistBody struct {
	CloudAICompanionProject string                    `json:"cloudaicompanionProject"`
	Metadata                GemLoadCodeAssistMetadata `json:"metadata"`
}

// GemLoadCodeAssistMetadata holds metadata for the loadCodeAssist call.
type GemLoadCodeAssistMetadata struct {
	IDEType     string `json:"ideType"`
	Platform    string `json:"platform"`
	PluginType  string `json:"pluginType"`
	DuetProject string `json:"duetProject"`
}

// --- Helpers ---

// NewGemTextPart creates a text part.
func NewGemTextPart(text string) GemPart {
	return GemPart{Text: text}
}

// NewGemImagePart creates an inline data part from raw image bytes.
func NewGemImagePart(mimeType string, data []byte) GemPart {
	return GemPart{
		InlineData: &GemInlineData{
			MimeType: mimeType,
			Data:     base64.StdEncoding.EncodeToString(data),
		},
	}
}

// NewGemFunctionCallPart creates a function call part.
func NewGemFunctionCallPart(name string, args map[string]interface{}) GemPart {
	return GemPart{
		FunctionCall: &GemFunctionCall{Name: name, Args: args},
	}
}

// NewGemFunctionResponsePart creates a function response part.
func NewGemFunctionResponsePart(name, content string) GemPart {
	return GemPart{
		FunctionResponse: &GemFunctionResponse{
			Name:     name,
			Response: GemFunctionResult{Content: content},
		},
	}
}

// GemToolDefsFromPoly converts ToolDefinition slice to Gemini tool groups.
func GemToolDefsFromPoly(defs []ToolDefinition) []GemToolGroup {
	if len(defs) == 0 {
		return nil
	}
	decls := make([]GemFunctionDeclaration, len(defs))
	for i, d := range defs {
		decls[i] = GemFunctionDeclaration{
			Name:        d.Name,
			Description: d.Description,
			Parameters:  d.InputSchema,
		}
	}
	return []GemToolGroup{{FunctionDeclarations: decls}}
}

// BuildGemPartsFromMessage creates Gemini parts from a Message,
// including image support.
func BuildGemPartsFromMessage(msg Message) []GemPart {
	if len(msg.Images) == 0 {
		return []GemPart{NewGemTextPart(msg.Content)}
	}
	parts := make([]GemPart, 0, len(msg.Images)+1)
	for _, img := range msg.Images {
		parts = append(parts, NewGemImagePart(img.MediaType, img.Data))
	}
	if msg.Content != "" {
		parts = append(parts, NewGemTextPart(msg.Content))
	}
	return parts
}
