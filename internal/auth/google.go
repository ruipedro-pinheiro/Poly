package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Google OAuth config (from Gemini CLI, public credentials)
const (
	GoogleClientID           = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
	googleClientSecretDefault = "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"
	GoogleAuthorizeURL       = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL           = "https://oauth2.googleapis.com/token"
	GoogleCallbackPort       = 8086
)

// GoogleClientSecret returns the OAuth client secret, preferring env var over default.
func GoogleClientSecret() string {
	if secret := os.Getenv("GOOGLE_CLIENT_SECRET"); secret != "" {
		return secret
	}
	return googleClientSecretDefault
}

// StartGoogleOAuth starts the OAuth flow for Google/Gemini
func StartGoogleOAuth() (string, error) {
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", GoogleCallbackPort)
	state, err := GenerateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	params := url.Values{
		"client_id":     {GoogleClientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {"https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}

	authURL := GoogleAuthorizeURL + "?" + params.Encode()
	openBrowser(authURL)

	return authURL, nil
}

// StartGoogleOAuthWithCallback starts OAuth and waits for callback
func StartGoogleOAuthWithCallback() (*OAuthTokens, error) {
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", GoogleCallbackPort)
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	params := url.Values{
		"client_id":     {GoogleClientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {"https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}

	authURL := GoogleAuthorizeURL + "?" + params.Encode()

	// Start local server to receive callback
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", GoogleCallbackPort))
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
				w.Write([]byte("Error: state mismatch"))
				return
			}

			if code == "" {
				errorMsg := r.URL.Query().Get("error")
				errChan <- fmt.Errorf("no code in callback: %s", errorMsg)
				w.Write([]byte("Error: " + errorMsg))
				return
			}

			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
				<html><body style="font-family: system-ui; padding: 40px; text-align: center;">
					<h1>Success!</h1>
					<p>You can close this window and return to Poly.</p>
				</body></html>
			`))
			codeChan <- code
		}),
	}

	go server.Serve(listener)
	defer server.Close()

	// Open browser
	openBrowser(authURL)

	// Wait for callback (with timeout)
	select {
	case code := <-codeChan:
		return exchangeGoogleCode(code, redirectURI)
	case err := <-errChan:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("OAuth timeout")
	}
}

// ExchangeGoogleCode exchanges code for tokens (for manual flow)
func ExchangeGoogleCode(code string) (*OAuthTokens, error) {
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", GoogleCallbackPort)
	return exchangeGoogleCode(code, redirectURI)
}

// exchangeGoogleCode exchanges the code for tokens
func exchangeGoogleCode(code, redirectURI string) (*OAuthTokens, error) {
	data := url.Values{
		"code":          {code},
		"client_id":     {GoogleClientID},
		"client_secret": {GoogleClientSecret()},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequest("POST", GoogleTokenURL, strings.NewReader(data.Encode()))
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
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
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

// RefreshGoogleToken refreshes the access token
func RefreshGoogleToken(refreshToken string) (*OAuthTokens, error) {
	data := url.Values{
		"client_id":     {GoogleClientID},
		"client_secret": {GoogleClientSecret()},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequest("POST", GoogleTokenURL, strings.NewReader(data.Encode()))
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
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
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
		RefreshToken: refreshToken, // Google doesn't return new refresh token
		ExpiresAt:    time.Now().Unix() + expiresIn,
	}, nil
}
