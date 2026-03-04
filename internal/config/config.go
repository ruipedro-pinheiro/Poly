package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/pedromelo/poly/internal/hooks"
)

// ProviderConfig defines a provider's settings
type ProviderConfig struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Endpoint      string            `json:"endpoint"`
	Models        map[string]string `json:"models"` // variant -> model name
	Color         string            `json:"color"`
	MaxTokens     int               `json:"max_tokens"`
	Timeout       int               `json:"timeout_seconds"`
	Format        string            `json:"format"`          // "anthropic", "openai", "google"
	AuthType      string            `json:"auth_type"`       // "api_key", "oauth", "none"
	AuthHeader    string            `json:"auth_header"`     // "Bearer", "x-api-key", etc.
	OAuthClientID string            `json:"oauth_client_id"` // OAuth client ID for providers using OAuth
	CostTier      int               `json:"cost_tier"`       // 1=cheap, 2=mid, 3=expensive (for @all cascade ordering)

	// ReasoningModels lists model prefixes that use reasoning tokens.
	// For these models, max_completion_tokens is sent instead of max_tokens.
	// Config-driven to avoid hardcoding model names in provider code.
	ReasoningModels []string `json:"reasoning_models,omitempty"`

	// ReasoningEffortModels lists model prefixes that accept the reasoning_effort parameter.
	// This is a subset of ReasoningModels — some models reason by default but reject reasoning_effort.
	ReasoningEffortModels []string `json:"reasoning_effort_models,omitempty"`
}

// MCPServerConfig defines an MCP server in the config file
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Type    string            `json:"type,omitempty"` // default: "stdio"
	URL     string            `json:"url,omitempty"`  // for sse/http
}

// Config holds all Poly configuration
type Config struct {
	Providers       map[string]ProviderConfig  `json:"providers"`
	DefaultProvider string                     `json:"default_provider"`
	Theme           ThemeConfig                `json:"theme"`
	Settings        SettingsConfig             `json:"settings"`
	Hooks           hooks.HooksConfig          `json:"hooks"`
	MCPServers      map[string]MCPServerConfig `json:"mcp_servers,omitempty"`
}

// ThemeConfig defines UI colors
type ThemeConfig struct {
	ProviderColors map[string]string `json:"provider_colors"`
}

// SettingsConfig defines general settings
type SettingsConfig struct {
	MaxToolTurns    int    `json:"max_tool_turns"`
	MaxTableRounds  int    `json:"max_table_rounds"`
	StreamingBuffer int    `json:"streaming_buffer"`
	SaveSessions    bool   `json:"save_sessions"`
	ColorTheme      string `json:"color_theme,omitempty"`
	Notifications   *bool  `json:"notifications,omitempty"`
	Sandbox         bool   `json:"sandbox,omitempty"`
	SandboxImage    string `json:"sandbox_image,omitempty"`
}

var (
	current             *Config
	configMu            sync.RWMutex
	configDir           string
	projectConfigLoaded bool
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		// Fallback to current directory if home is not available (common on some Windows setups)
		cwd, _ := os.Getwd()
		configDir = filepath.Join(cwd, ".poly")
	} else {
		configDir = filepath.Join(home, ".poly")
	}
}

// GetConfigDir returns the config directory
func GetConfigDir() string {
	return configDir
}

// Load loads config from disk, merging with defaults
func Load() (*Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	// Start with defaults
	cfg := DefaultConfig()

	// Try to load user config
	configFile := filepath.Join(configDir, "config.json")
	data, err := os.ReadFile(configFile)
	if err == nil {
		var userCfg Config
		if err := json.Unmarshal(data, &userCfg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid JSON in %s: %v (using defaults)\n", configFile, err)
		} else {
			// Merge user config into defaults
			cfg = mergeConfig(cfg, &userCfg)
		}
	}

	// Try to load project-level config (.poly/config.json in cwd)
	projectConfigLoaded = false
	loadProjectConfig(cfg)

	current = cfg
	hooks.SetConfig(&cfg.Hooks)
	return cfg, nil
}

// Get returns the current config (loads if needed)
func Get() *Config {
	configMu.RLock()
	if current != nil {
		defer configMu.RUnlock()
		return current
	}
	configMu.RUnlock()

	cfg, _ := Load()
	return cfg
}

// Save saves the current config to disk
func Save() error {
	configMu.RLock()
	cfg := current
	configMu.RUnlock()

	if cfg == nil {
		return nil
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(configDir, "config.json"), data, 0600)
}

// GetProviderColor returns a provider's color
func GetProviderColor(providerID string) string {
	cfg := Get()
	// Check theme override first
	if color, ok := cfg.Theme.ProviderColors[providerID]; ok {
		return color
	}
	// Fall back to provider config
	if p, ok := cfg.Providers[providerID]; ok {
		return p.Color
	}
	return "#888888" // fallback
}

// GetProviderNames returns all provider IDs sorted alphabetically.
// No hardcoded order - custom providers appear alongside native ones.
func GetProviderNames() []string {
	cfg := Get()
	result := make([]string, 0, len(cfg.Providers))
	for id := range cfg.Providers {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

// SetProvider adds or updates a provider config
// GetColorTheme returns the saved color theme name
func GetColorTheme() string {
	cfg := Get()
	return cfg.Settings.ColorTheme
}

// SetColorTheme updates the color theme and saves config
func SetColorTheme(name string) {
	configMu.Lock()
	if current == nil {
		current = DefaultConfig()
	}
	current.Settings.ColorTheme = name
	configMu.Unlock()
	_ = Save()
}

// NotificationsEnabled returns whether desktop notifications are enabled (default: true)
func NotificationsEnabled() bool {
	cfg := Get()
	if cfg.Settings.Notifications == nil {
		return true
	}
	return *cfg.Settings.Notifications
}

// SetNotifications updates the notifications setting and saves config
func SetNotifications(enabled bool) {
	configMu.Lock()
	if current == nil {
		current = DefaultConfig()
	}
	current.Settings.Notifications = &enabled
	configMu.Unlock()
	_ = Save()
}

// SandboxEnabled returns whether sandbox mode is enabled
func SandboxEnabled() bool {
	cfg := Get()
	return cfg.Settings.Sandbox
}

// SetSandbox updates the sandbox setting and saves config
func SetSandbox(enabled bool) {
	configMu.Lock()
	if current == nil {
		current = DefaultConfig()
	}
	current.Settings.Sandbox = enabled
	configMu.Unlock()
	_ = Save()
}

// GetSandboxImage returns the configured sandbox image (default: alpine:latest)
func GetSandboxImage() string {
	cfg := Get()
	if cfg.Settings.SandboxImage != "" {
		return cfg.Settings.SandboxImage
	}
	return "alpine:latest"
}

// SetMaxTableRounds updates the max table rounds setting and saves config
func SetMaxTableRounds(n int) {
	configMu.Lock()
	if current == nil {
		current = DefaultConfig()
	}
	current.Settings.MaxTableRounds = n
	configMu.Unlock()
	_ = Save()
}

// GetMCPServers returns the configured MCP servers
func GetMCPServers() map[string]MCPServerConfig {
	cfg := Get()
	if cfg.MCPServers == nil {
		return nil
	}
	return cfg.MCPServers
}

// mergeConfig merges user config into defaults
func mergeConfig(base, user *Config) *Config {
	if user.DefaultProvider != "" {
		base.DefaultProvider = user.DefaultProvider
	}

	// Merge providers (user overrides defaults)
	for id, p := range user.Providers {
		if existing, ok := base.Providers[id]; ok {
			// Merge individual fields
			if p.Name != "" {
				existing.Name = p.Name
			}
			if p.Endpoint != "" {
				existing.Endpoint = p.Endpoint
			}
			if p.Color != "" {
				existing.Color = p.Color
			}
			if p.MaxTokens > 0 {
				existing.MaxTokens = p.MaxTokens
			}
			if p.Timeout > 0 {
				existing.Timeout = p.Timeout
			}
			if p.Format != "" {
				existing.Format = p.Format
			}
			if p.AuthType != "" {
				existing.AuthType = p.AuthType
			}
			if p.AuthHeader != "" {
				existing.AuthHeader = p.AuthHeader
			}
			if p.CostTier > 0 {
				existing.CostTier = p.CostTier
			}
			if p.OAuthClientID != "" {
				existing.OAuthClientID = p.OAuthClientID
			}
			// Merge models
			if p.Models != nil {
				for k, v := range p.Models {
					existing.Models[k] = v
				}
			}
			// Merge reasoning config (user overrides entirely if provided)
			if len(p.ReasoningModels) > 0 {
				existing.ReasoningModels = p.ReasoningModels
			}
			if len(p.ReasoningEffortModels) > 0 {
				existing.ReasoningEffortModels = p.ReasoningEffortModels
			}
			base.Providers[id] = existing
		} else {
			base.Providers[id] = p
		}
	}

	// Merge theme
	for id, color := range user.Theme.ProviderColors {
		base.Theme.ProviderColors[id] = color
	}

	// Merge settings
	if user.Settings.MaxToolTurns > 0 {
		base.Settings.MaxToolTurns = user.Settings.MaxToolTurns
	}
	if user.Settings.MaxTableRounds > 0 {
		base.Settings.MaxTableRounds = user.Settings.MaxTableRounds
	}
	if user.Settings.StreamingBuffer > 0 {
		base.Settings.StreamingBuffer = user.Settings.StreamingBuffer
	}
	if user.Settings.ColorTheme != "" {
		base.Settings.ColorTheme = user.Settings.ColorTheme
	}
	if user.Settings.Notifications != nil {
		base.Settings.Notifications = user.Settings.Notifications
	}
	if user.Settings.Sandbox {
		base.Settings.Sandbox = user.Settings.Sandbox
	}
	if user.Settings.SandboxImage != "" {
		base.Settings.SandboxImage = user.Settings.SandboxImage
	}

	// Merge hooks (user overrides entirely if provided)
	if len(user.Hooks.PreTool) > 0 {
		base.Hooks.PreTool = user.Hooks.PreTool
	}
	if len(user.Hooks.PostTool) > 0 {
		base.Hooks.PostTool = user.Hooks.PostTool
	}
	if len(user.Hooks.OnMessage) > 0 {
		base.Hooks.OnMessage = user.Hooks.OnMessage
	}

	// Merge MCP servers
	if len(user.MCPServers) > 0 {
		if base.MCPServers == nil {
			base.MCPServers = make(map[string]MCPServerConfig)
		}
		for name, srv := range user.MCPServers {
			base.MCPServers[name] = srv
		}
	}

	return base
}

// ProjectConfigLoaded returns whether a project-level config was found and loaded
func ProjectConfigLoaded() bool {
	return projectConfigLoaded
}

// ProjectConfigPath returns the path to the project config if it exists, empty string otherwise
func ProjectConfigPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	path := filepath.Join(cwd, ".poly", "config.json")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// loadProjectConfig loads .poly/config.json from cwd if it exists
func loadProjectConfig(cfg *Config) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	path := filepath.Join(cwd, ".poly", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return // no project config, that's fine
	}
	var projectCfg Config
	if err := json.Unmarshal(data, &projectCfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: invalid JSON in %s: %v (skipping project config)\n", path, err)
		return // invalid JSON, warn and skip
	}
	mergeConfig(cfg, &projectCfg)
	projectConfigLoaded = true
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider: "claude",
		Providers: map[string]ProviderConfig{
			"claude": {
				ID:            "claude",
				Name:          "Claude",
				Endpoint:      "https://api.anthropic.com/v1",
				Format:        "anthropic",
				AuthType:      "oauth",
				OAuthClientID: "9d1c250a-e61b-44d9-88ed-5944d1962f5e",
				MaxTokens:     8192,
				Timeout:       300,
				Color:         "#D97706",
				CostTier:      3,
				Models: map[string]string{
					"default": "claude-sonnet-4-6",
					"fast":    "claude-haiku-4-5",
					"think":   "claude-sonnet-4-6",
					"opus":    "claude-opus-4-6",
					"sonnet":  "claude-sonnet-4-6",
				},
				// Claude uses extended thinking (budgetTokens), not reasoning_effort.
				// Reasoning detection handled natively by the Anthropic provider.
			},
			"gpt": {
				ID:        "gpt",
				Name:      "GPT",
				Endpoint:  "https://api.openai.com/v1",
				Format:    "openai",
				AuthType:  "api_key",
				MaxTokens: 4096,
				Timeout:   300,
				Color:     "#10A37F",
				CostTier:  1,
				Models: map[string]string{
					"default": "gpt-4.1",
					"fast":    "gpt-4.1-mini",
					"nano":    "gpt-4.1-nano",
					"think":   "gpt-5-mini",
					"gpt5":    "gpt-5.2",
					"o3":      "o3",
					"o4":      "o4-mini",
				},
				ReasoningModels:       []string{"gpt-5", "o3", "o4"},
				ReasoningEffortModels: []string{"gpt-5", "o3", "o4"},
			},
			"gemini": {
				ID:        "gemini",
				Name:      "Gemini",
				Endpoint:  "https://generativelanguage.googleapis.com/v1beta",
				Format:    "google",
				AuthType:  "oauth",
				MaxTokens: 4096,
				Timeout:   300,
				Color:     "#4285F4",
				CostTier:  2,
				Models: map[string]string{
					"default": "gemini-2.5-flash",
					"pro":     "gemini-2.5-pro",
					"think":   "gemini-2.5-pro",
					"lite":    "gemini-2.5-flash-lite",
				},
				// Gemini uses thinkingConfig, not reasoning_effort.
				// Reasoning detection handled natively by the Gemini provider.
			},
			"grok": {
				ID:        "grok",
				Name:      "Grok",
				Endpoint:  "https://api.x.ai/v1",
				Format:    "openai",
				AuthType:  "api_key",
				MaxTokens: 4096,
				Timeout:   300,
				Color:     "#1DA1F2",
				CostTier:  2,
				Models: map[string]string{
					"default":    "grok-4-0709",
					"fast":       "grok-4-1-fast-non-reasoning",
					"think":      "grok-3-mini",
					"think-fast": "grok-4-1-fast-reasoning",
					"code":       "grok-code-fast-1",
				},
				ReasoningModels:       []string{"grok-3-mini", "grok-4-0", "grok-4-fast-reasoning", "grok-4-1-fast-reasoning", "grok-code"},
				ReasoningEffortModels: []string{"grok-3-mini"},
			},
			"copilot": {
				ID:        "copilot",
				Name:      "Copilot",
				Endpoint:  "https://api.githubcopilot.com",
				Format:    "openai",
				AuthType:  "device_flow",
				MaxTokens: 4096,
				Timeout:   300,
				Color:     "#6e40c9",
				CostTier:  0, // Included with GitHub Copilot subscription
				Models: map[string]string{
					"default": "gpt-4o",
					"fast":    "gpt-4o-mini",
					"claude":  "claude-sonnet-4",
					"think":   "o4-mini",
				},
				ReasoningModels:       []string{"o3", "o4"},
				ReasoningEffortModels: []string{"o3", "o4"},
			},
			"ollama": {
				ID:        "ollama",
				Name:      "Ollama",
				Endpoint:  "http://localhost:11434/api",
				Format:    "openai",
				AuthType:  "none",
				MaxTokens: 4096,
				Timeout:   300,
				Color:     "#FFFFFF",
				CostTier:  0, // Local, so it's free
				Models: map[string]string{
					"default":   "llama3",
					"llama3":    "llama3",
					"codellama": "codellama",
				},
			},
		},
		Theme: ThemeConfig{
			ProviderColors: map[string]string{},
		},
		Settings: SettingsConfig{
			MaxToolTurns:    50,
			StreamingBuffer: 4096,
			SaveSessions:    true,
		},
	}
}
