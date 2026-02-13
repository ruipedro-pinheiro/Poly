package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/pedromelo/poly/internal/config"
)

const (
	AnthropicRedirectURI    = "https://console.anthropic.com/oauth/code/callback"
	AnthropicTokenURL       = "https://console.anthropic.com/v1/oauth/token"
	AnthropicAuthURLMax     = "https://claude.ai/oauth/authorize"
	AnthropicAuthURLConsole = "https://console.anthropic.com/oauth/authorize"
)

// OAuthTokens holds the OAuth tokens
type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// pendingAuth stores the pending PKCE verifier and state
type pendingAuth struct {
	Verifier string
	State    string
}

var pendingOAuth *pendingAuth

// StartAnthropicOAuth starts the OAuth flow and opens the browser
// Returns the auth URL for display if browser can't be opened
func StartAnthropicOAuth(mode string) (string, error) {
	pkce, err := GeneratePKCE()
	if err != nil {
		return "", fmt.Errorf("failed to generate PKCE: %w", err)
	}

	state, err := generateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	pendingOAuth = &pendingAuth{
		Verifier: pkce.Verifier,
		State:    state,
	}

	baseURL := AnthropicAuthURLMax
	if mode == "console" {
		baseURL = AnthropicAuthURLConsole
	}

	params := url.Values{
		"client_id":             {config.Get().Providers["claude"].OAuthClientID},
		"response_type":         {"code"},
		"redirect_uri":          {AnthropicRedirectURI},
		"scope":                 {"org:create_api_key user:profile user:inference"},
		"code_challenge":        {pkce.Challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
		"code":                  {"true"},
	}

	authURL := baseURL + "?" + params.Encode()

	// Try to open browser
	openBrowser(authURL)

	return authURL, nil
}

// ExchangeAnthropicCode exchanges the authorization code for tokens.
// Accepts: full callback URL (?code=...&state=...), raw "CODE#STATE", or raw code.
func ExchangeAnthropicCode(input string) (*OAuthTokens, error) {
	if pendingOAuth == nil {
		return nil, fmt.Errorf("no pending OAuth flow. Start OAuth first")
	}

	input = strings.TrimSpace(input)
	var code, state string

	if strings.HasPrefix(input, "http") {
		// Full callback URL: extract code and state from query params
		parsedURL, err := url.Parse(input)
		if err != nil {
			return nil, fmt.Errorf("invalid callback URL: %w", err)
		}
		q := parsedURL.Query()
		code = q.Get("code")
		state = q.Get("state")
		if code == "" {
			return nil, fmt.Errorf("authorization code not found in callback URL")
		}
	} else if strings.Contains(input, "#") {
		// Anthropic callback page displays "CODE#STATE" — split it
		parts := strings.SplitN(input, "#", 2)
		code = parts[0]
		state = parts[1]
	} else {
		// Raw code only
		code = input
		state = pendingOAuth.State
	}

	if code == "" {
		return nil, fmt.Errorf("no authorization code provided")
	}

	// Build form-encoded request body (OAuth 2.0 standard)
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {config.Get().Providers["claude"].OAuthClientID},
		"code":          {code},
		"state":         {state},
		"redirect_uri":  {AnthropicRedirectURI},
		"code_verifier": {pendingOAuth.Verifier},
	}

	req, err := http.NewRequest("POST", AnthropicTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(respBody))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Clear pending verifier and state
	pendingOAuth = nil

	expiresIn := result.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}

	return &OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Unix() + expiresIn,
	}, nil
}

// RefreshAnthropicToken refreshes the access token
func RefreshAnthropicToken(refreshToken string) (*OAuthTokens, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {config.Get().Providers["claude"].OAuthClientID},
	}

	req, err := http.NewRequest("POST", AnthropicTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s", string(respBody))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	expiresIn := result.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}

	// Use new refresh token if provided, otherwise keep the old one
	newRefreshToken := result.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = refreshToken
	}

	return &OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Unix() + expiresIn,
	}, nil
}

// HasPendingAuth returns true if there's a pending OAuth flow
func HasPendingAuth() bool {
	return pendingOAuth != nil
}

// openBrowser opens the URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// generateRandomString generates a secure random string of the specified length.
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
