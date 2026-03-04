package auth

import (
	"fmt"

	"github.com/pedromelo/poly/internal/config"
)

// OpenAI OAuth config (from Codex CLI, public client)
const (
	OpenAIClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	OpenAIAuthorizeURL = "https://auth.openai.com/oauth/authorize"
	OpenAITokenURL     = "https://auth.openai.com/oauth/token"
	OpenAICallbackPort = 1455
)

func getOpenAIConfig() OAuthConfig {
	clientID := config.Get().Providers["gpt"].OAuthClientID
	if clientID == "" {
		clientID = OpenAIClientID
	}

	return OAuthConfig{
		ClientID:     clientID,
		AuthorizeURL: OpenAIAuthorizeURL,
		TokenURL:     OpenAITokenURL,
		RedirectURI:  fmt.Sprintf("http://127.0.0.1:%d/callback", OpenAICallbackPort),
		Scopes:       "openid profile email offline_access model.request",
		ExtraParams: map[string]string{
			"id_token_add_organizations": "true",
			"codex_cli_simplified_flow":  "true",
			"originator":                 "poly",
		},
	}
}

// StartOpenAIOAuth starts the OAuth flow for OpenAI (manual mode)
func StartOpenAIOAuth() (string, error) {
	_, authURL, err := StartOAuthFlow(getOpenAIConfig(), 0)
	return authURL, err
}

// StartOpenAIOAuthWithCallback starts OAuth and waits for callback
func StartOpenAIOAuthWithCallback() (*OAuthTokens, error) {
	tokens, _, err := StartOAuthFlow(getOpenAIConfig(), OpenAICallbackPort)
	return tokens, err
}

// RefreshOpenAIToken refreshes the access token
func RefreshOpenAIToken(refreshToken string) (*OAuthTokens, error) {
	cfg := getOpenAIConfig()
	return RefreshToken(cfg.TokenURL, cfg.ClientID, "", refreshToken)
}
