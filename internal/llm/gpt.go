package llm

// GPTProvider implements Provider for OpenAI's GPT models.
// All shared logic lives in OAIBaseProvider — this file only defines
// the GPT-specific configuration (endpoint, provider ID, color).
type GPTProvider struct {
	OAIBaseProvider
}

// NewGPTProvider creates a new GPT provider.
func NewGPTProvider(cfg ProviderConfig) *GPTProvider {
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel("gpt")
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = GetProviderMaxTokens("gpt")
	}

	return &GPTProvider{
		OAIBaseProvider: OAIBaseProvider{
			providerID:  "gpt",
			displayName: "GPT",
			color:       "#10A37F",
			endpoint:    "https://api.openai.com/v1/chat/completions",
			config:      cfg,
			httpClient:  newStreamHTTPClient(),
		},
	}
}
