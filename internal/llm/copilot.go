package llm

import (
	"net/http"
	"strings"

	"github.com/pedromelo/poly/internal/auth"
)

// CopilotProvider implements Provider for GitHub Copilot.
// Copilot exposes an OpenAI-compatible chat completions API with custom headers
// and device-flow-based auth. Its specifics vs plain GPT:
//   - Custom HTTP headers (Copilot-Integration-Id, Editor-Version, etc.)
//   - 401 retry with automatic token refresh (session tokens expire ~30 min)
//   - Custom auth error message directing users to the Control Room
type CopilotProvider struct {
	OAIBaseProvider
}

// NewCopilotProvider creates a new Copilot provider.
func NewCopilotProvider(cfg ProviderConfig) *CopilotProvider {
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel("copilot")
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = GetProviderMaxTokens("copilot")
	}

	return &CopilotProvider{
		OAIBaseProvider: OAIBaseProvider{
			providerID:  "copilot",
			displayName: "Copilot",
			color:       "#6e40c9",
			endpoint:    "https://api.githubcopilot.com/chat/completions",
			config:      cfg,
			httpClient:  newStreamHTTPClient(),
			authError:   "not authenticated \u2014 connect via Control Room",

			setHeaders: func(req *http.Request, token string) {
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Copilot-Integration-Id", "vscode-chat")
				req.Header.Set("Editor-Version", "vscode/1.96.0")
				req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.24.0")
				req.Header.Set("Openai-Intent", "conversation-panel")
			},

			handleStreamError: func(err error, currentToken string) (string, error) {
				// Session tokens expire ~30 min. On 401, try refreshing.
				if strings.Contains(err.Error(), "401") {
					newToken, refreshErr := auth.GetStorage().GetAccessToken("copilot")
					if refreshErr == nil && newToken != "" && newToken != currentToken {
						return newToken, nil
					}
				}
				return "", err
			},
		},
	}
}
