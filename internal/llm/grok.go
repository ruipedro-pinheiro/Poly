package llm

// GrokProvider implements Provider for xAI's Grok models.
// All shared logic lives in OAIBaseProvider — this file only defines
// the Grok-specific configuration:
//   - hasReasoningContent: Grok streams reasoning_content in SSE deltas
//   - alwaysUseReasoningTokens: Grok models like grok-4 reason automatically,
//     so MaxCompletionTokens is always used for reasoning models (not just when
//     thinkingMode is active)
type GrokProvider struct {
	OAIBaseProvider
}

// NewGrokProvider creates a new Grok provider.
func NewGrokProvider(cfg ProviderConfig) *GrokProvider {
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel("grok")
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = GetProviderMaxTokens("grok")
	}

	return &GrokProvider{
		OAIBaseProvider: OAIBaseProvider{
			providerID:               "grok",
			displayName:              "Grok",
			color:                    "#1DA1F2",
			endpoint:                 "https://api.x.ai/v1/chat/completions",
			config:                   cfg,
			httpClient:               newStreamHTTPClient(),
			hasReasoningContent:      true,
			alwaysUseReasoningTokens: true,
		},
	}
}
