package auth

import (
	"fmt"
	"os"
)

// Google OAuth config (from Gemini CLI, public credentials)
const (
	GoogleClientID            = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
	googleClientSecretDefault = "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"
	GoogleAuthorizeURL        = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL            = "https://oauth2.googleapis.com/token"
	GoogleCallbackPort        = 8086
)

// GoogleClientSecret returns the OAuth client secret, preferring env var over default.
func GoogleClientSecret() string {
	if secret := os.Getenv("GOOGLE_CLIENT_SECRET"); secret != "" {
		return secret
	}
	return googleClientSecretDefault
}

func getGoogleConfig() OAuthConfig {
	return OAuthConfig{
		ClientID:     GoogleClientID,
		ClientSecret: GoogleClientSecret(),
		AuthorizeURL: GoogleAuthorizeURL,
		TokenURL:     GoogleTokenURL,
		RedirectURI:  fmt.Sprintf("http://127.0.0.1:%d/callback", GoogleCallbackPort),
		Scopes:       "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile",
		ExtraParams: map[string]string{
			"access_type": "offline",
			"prompt":      "consent",
		},
	}
}

// StartGoogleOAuth starts the OAuth flow for Google/Gemini (manual mode)
func StartGoogleOAuth() (string, error) {
	_, authURL, err := StartOAuthFlow(getGoogleConfig(), 0)
	return authURL, err
}

// StartGoogleOAuthWithCallback starts OAuth and waits for callback
func StartGoogleOAuthWithCallback() (*OAuthTokens, error) {
	tokens, _, err := StartOAuthFlow(getGoogleConfig(), GoogleCallbackPort)
	return tokens, err
}

// ExchangeGoogleCode exchanges code for tokens (for manual flow)
func ExchangeGoogleCode(code string) (*OAuthTokens, error) {
	return ExchangeCode(getGoogleConfig(), code, "") // Google doesn't strictly require verifier in exchange if not using PKCE params in start, but we use it
}

// RefreshGoogleToken refreshes the access token
func RefreshGoogleToken(refreshToken string) (*OAuthTokens, error) {
	cfg := getGoogleConfig()
	return RefreshToken(cfg.TokenURL, cfg.ClientID, cfg.ClientSecret, refreshToken)
}
