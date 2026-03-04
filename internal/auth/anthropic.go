package auth

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pedromelo/poly/internal/config"
)

const (
	AnthropicRedirectURI    = "https://console.anthropic.com/oauth/code/callback"
	AnthropicTokenURL       = "https://console.anthropic.com/v1/oauth/token"
	AnthropicAuthURLMax     = "https://claude.ai/oauth/authorize"
	AnthropicAuthURLConsole = "https://console.anthropic.com/oauth/authorize"
)

func getAnthropicConfig(mode string) OAuthConfig {
	baseURL := AnthropicAuthURLMax
	if mode == "console" {
		baseURL = AnthropicAuthURLConsole
	}

	return OAuthConfig{
		ClientID:     config.Get().Providers["claude"].OAuthClientID,
		AuthorizeURL: baseURL,
		TokenURL:     AnthropicTokenURL,
		RedirectURI:  AnthropicRedirectURI,
		Scopes:       "org:create_api_key user:profile user:inference",
		ExtraParams: map[string]string{
			"code": "true",
		},
	}
}

// StartAnthropicOAuth starts the OAuth flow for Anthropic (manual mode)
func StartAnthropicOAuth(mode string) (string, error) {
	_, authURL, err := StartOAuthFlow(getAnthropicConfig(mode), 0)
	return authURL, err
}

// ExchangeAnthropicCode exchanges the authorization code for tokens.
// Accepts: full callback URL (?code=...&state=...), raw "CODE#STATE", or raw code.
func ExchangeAnthropicCode(input string) (*OAuthTokens, error) {
	verifier, _ := GetPendingAuth()
	if verifier == "" {
		return nil, fmt.Errorf("no pending OAuth flow. Start OAuth first")
	}

	input = strings.TrimSpace(input)
	var code string

	if strings.HasPrefix(input, "http") {
		parsedURL, err := url.Parse(input)
		if err != nil {
			return nil, fmt.Errorf("invalid callback URL: %w", err)
		}
		q := parsedURL.Query()
		code = q.Get("code")
		if code == "" {
			return nil, fmt.Errorf("authorization code not found in callback URL")
		}
	} else if strings.Contains(input, "#") {
		parts := strings.SplitN(input, "#", 2)
		code = parts[0]
	} else {
		code = input
	}

	if code == "" {
		return nil, fmt.Errorf("no authorization code provided")
	}

	tokens, err := ExchangeCode(getAnthropicConfig(""), code, verifier)
	if err == nil {
		ClearPendingAuth()
	}
	return tokens, err
}

// RefreshAnthropicToken refreshes the access token
func RefreshAnthropicToken(refreshToken string) (*OAuthTokens, error) {
	cfg := getAnthropicConfig("")
	return RefreshToken(cfg.TokenURL, cfg.ClientID, "", refreshToken)
}
