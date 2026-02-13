package llm

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/pedromelo/poly/internal/config"
)

// ToolFormat defines how a provider expects tool definitions
type ToolFormat string

const (
	ToolFormatAnthropic ToolFormat = "anthropic" // tools with input_schema
	ToolFormatOpenAI    ToolFormat = "openai"    // tools with parameters (function calling)
	ToolFormatGoogle    ToolFormat = "google"    // functionDeclarations
)

// Message represents a chat message
type Message struct {
	Role       string      `json:"role"` // "user", "assistant", "system"
	Content    string      `json:"content"`
	Images     []Image     `json:"images,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// Image represents an image attachment
type Image struct {
	Data      []byte `json:"data"`       // Base64-decoded image data
	MediaType string `json:"media_type"` // e.g., "image/png", "image/jpeg"
	Path      string `json:"path"`       // Original file path (for display)
}

// ToolCall represents a tool invocation from the LLM
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of executing a tool
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// Response represents a complete response from a provider
type Response struct {
	Content             string
	Provider            string
	Model               string
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int // Anthropic: tokens written to cache
	CacheReadTokens     int // Anthropic: tokens read from cache
	ToolCalls           []ToolCall // Tools the LLM wants to call
	StopReason          string     // "end_turn", "tool_use", etc.
}

// StreamEvent represents a streaming event
type StreamEvent struct {
	Type       string      // "content", "thinking", "tool_use", "tool_result", "done", "error"
	Content    string
	Thinking   string
	ToolCall   *ToolCall
	ToolResult *ToolResult // Result of a tool execution (paired with ToolCall)
	Response   *Response
	Error      error
}

// ToolDefinition for sending to LLMs
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"` // JSON Schema (Anthropic format)
}

// Provider interface for all LLM providers
type Provider interface {
	// Name returns the provider name (claude, gpt, gemini, grok)
	Name() string

	// DisplayName returns a human-readable name
	DisplayName() string

	// Color returns the theme color for this provider (hex)
	Color() string

	// ToolFormat returns how this provider expects tool definitions
	ToolFormat() ToolFormat

	// Stream sends a message and streams the response
	Stream(ctx context.Context, messages []Message, tools []ToolDefinition, opts ...StreamOptions) <-chan StreamEvent

	// IsConfigured returns true if the provider has valid credentials
	IsConfigured() bool

	// SetModel changes the model for this provider
	SetModel(model string)

	// GetModel returns the current model
	GetModel() string

	// SupportsTools returns true if this provider supports tool use
	SupportsTools() bool
}

// ProviderConfig holds common provider configuration
type ProviderConfig struct {
	APIKey       string
	Model        string
	MaxTokens    int
	SystemPrompt string
}

// StreamOptions configures streaming behavior
type StreamOptions struct {
	Role         string // "default", "responder", "reviewer"
	ThinkingMode bool   // request extended thinking from provider
}

// GetRole extracts the role from StreamOptions, defaulting to "default"
func GetRole(opts []StreamOptions) string {
	if len(opts) > 0 && opts[0].Role != "" {
		return opts[0].Role
	}
	return "default"
}

// GetThinkingMode extracts thinking mode from StreamOptions
func GetThinkingMode(opts []StreamOptions) bool {
	if len(opts) > 0 {
		return opts[0].ThinkingMode
	}
	return false
}

// Registry holds all registered providers
var (
	providerRegistry = make(map[string]Provider)
	registryMu       sync.RWMutex

	// imageSupport tracks which providers support images (auto-detected)
	// true = supported, false = not supported, missing = unknown (try with images)
	imageSupport   = make(map[string]bool)
	imageSupportMu sync.RWMutex
)

// SupportsImages returns whether a provider supports images (auto-detected)
func SupportsImages(providerName string) bool {
	imageSupportMu.RLock()
	supported, known := imageSupport[providerName]
	imageSupportMu.RUnlock()

	if !known {
		return true // Unknown = try with images
	}
	return supported
}

// SetImageSupport marks whether a provider supports images
func SetImageSupport(providerName string, supported bool) {
	imageSupportMu.Lock()
	imageSupport[providerName] = supported
	imageSupportMu.Unlock()
}

// IsImageError checks if an error indicates image inputs aren't supported
func IsImageError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "image") && (strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "unsupported") ||
		strings.Contains(msg, "invalid"))
}

// RegisterProvider adds a provider to the registry
func RegisterProvider(p Provider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	providerRegistry[p.Name()] = p
}

// GetProvider returns a provider by name
func GetProvider(name string) (Provider, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := providerRegistry[name]
	return p, ok
}

// GetAllProviders returns all registered providers
func GetAllProviders() map[string]Provider {
	registryMu.RLock()
	defer registryMu.RUnlock()
	result := make(map[string]Provider)
	for k, v := range providerRegistry {
		result[k] = v
	}
	return result
}

// GetConfiguredProviders returns only providers that are configured
func GetConfiguredProviders() []Provider {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var result []Provider
	for _, p := range providerRegistry {
		if p.IsConfigured() {
			result = append(result, p)
		}
	}
	return result
}

// GetProviderNames returns all registered provider names sorted alphabetically.
// No hardcoded order - custom providers appear alongside native ones.
func GetProviderNames() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make([]string, 0, len(providerRegistry))
	for name := range providerRegistry {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// DefaultModels returns defaults from config (for backwards compatibility)
var DefaultModels = defaultModelsFromConfig()

func defaultModelsFromConfig() map[string]string {
	result := make(map[string]string)
	cfg := config.Get()
	for id, p := range cfg.Providers {
		if model, ok := p.Models["default"]; ok {
			result[id] = model
		}
	}
	return result
}
