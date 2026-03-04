package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pedromelo/poly/internal/security"
)

// OpenAI OAuth config (from Codex CLI, public client)
const (
	OpenAIClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	OpenAIAuthorizeURL = "https://auth.openai.com/oauth/authorize"
	OpenAITokenURL     = "https://auth.openai.com/oauth/token"
	OpenAICallbackPort = 1455
)

// StartOpenAIOAuth starts the OAuth flow for OpenAI
func StartOpenAIOAuth() (string, error) {
	pkce, err := GeneratePKCE()
	if err != nil {
		return "", fmt.Errorf("failed to generate PKCE: %w", err)
	}

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", OpenAICallbackPort)
	state, err := GenerateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	params := url.Values{
		"response_type":              {"code"},
		"client_id":                  {OpenAIClientID},
		"redirect_uri":               {redirectURI},
		"scope":                      {"openid profile email offline_access model.request"},
		"code_challenge":             {pkce.Challenge},
		"code_challenge_method":      {"S256"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
		"state":                      {state},
		"originator":                 {"poly"},
	}

	authURL := OpenAIAuthorizeURL + "?" + params.Encode()
	_ = openBrowser(authURL)

	return authURL, nil
}

// StartOpenAIOAuthWithCallback starts OAuth and waits for callback
func StartOpenAIOAuthWithCallback() (*OAuthTokens, error) {
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKCE: %w", err)
	}

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", OpenAICallbackPort)
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	params := url.Values{
		"response_type":              {"code"},
		"client_id":                  {OpenAIClientID},
		"redirect_uri":               {redirectURI},
		"scope":                      {"openid profile email offline_access model.request"},
		"code_challenge":             {pkce.Challenge},
		"code_challenge_method":      {"S256"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
		"state":                      {state},
		"originator":                 {"poly"},
	}

	authURL := OpenAIAuthorizeURL + "?" + params.Encode()

	// Start local server to receive callback
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", OpenAICallbackPort))
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			code := r.URL.Query().Get("code")
			returnedState := r.URL.Query().Get("state")

			if returnedState != state {
				errChan <- fmt.Errorf("state mismatch")
				_, _ = w.Write([]byte("Error: state mismatch"))
				return
			}

			if code == "" {
				errChan <- fmt.Errorf("no code in callback")
				_, _ = w.Write([]byte("Error: no code received"))
				return
			}

			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`
				<html><body style="font-family: system-ui; padding: 40px; text-align: center;">
					<h1>Success!</h1>
					<p>You can close this window and return to Poly.</p>
				</body></html>
			`))
			codeChan <- code
		}),
	}

	go func() { _ = server.Serve(listener) }()
	defer server.Close()

	// Open browser
	_ = openBrowser(authURL)

	// Wait for callback (with timeout)
	select {
	case code := <-codeChan:
		return exchangeOpenAICode(code, redirectURI, pkce.Verifier)
	case err := <-errChan:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("OAuth timeout")
	}
}

// exchangeOpenAICode exchanges the code for tokens
func exchangeOpenAICode(code, redirectURI, verifier string) (*OAuthTokens, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {OpenAIClientID},
		"code_verifier": {verifier},
	}

	req, err := http.NewRequest("POST", OpenAITokenURL, strings.NewReader(data.Encode()))
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", security.SanitizeResponseBody(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		IDToken      string `json:"id_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

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

// RefreshOpenAIToken refreshes the access token
func RefreshOpenAIToken(refreshToken string) (*OAuthTokens, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {OpenAIClientID},
	}

	req, err := http.NewRequest("POST", OpenAITokenURL, strings.NewReader(data.Encode()))
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s", security.SanitizeResponseBody(body))
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
