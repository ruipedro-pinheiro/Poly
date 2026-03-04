package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pedromelo/poly/internal/security"
)

// OAuthTokens holds the OAuth tokens
type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// OAuthConfig holds the configuration for an OAuth provider
type OAuthConfig struct {
	ClientID     string
	ClientSecret string // Optional, used by some providers like Google
	AuthorizeURL string
	TokenURL     string
	RedirectURI  string
	Scopes       string
	ExtraParams  map[string]string
}

// StartOAuthFlow starts the PKCE OAuth flow. 
func StartOAuthFlow(cfg OAuthConfig, callbackPort int) (*OAuthTokens, string, error) {
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate PKCE: %w", err)
	}

	state, err := GenerateState()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate state: %w", err)
	}

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {cfg.RedirectURI},
		"scope":                 {cfg.Scopes},
		"code_challenge":        {pkce.Challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}

	for k, v := range cfg.ExtraParams {
		params.Set(k, v)
	}

	authURL := cfg.AuthorizeURL + "?" + params.Encode()

	// Mode 1: Manual copy-paste
	if callbackPort == 0 {
		pendingOAuthMu.Lock()
		pendingOAuth = &pendingAuth{
			Verifier: pkce.Verifier,
			State:    state,
		}
		pendingOAuthMu.Unlock()

		_ = openBrowser(authURL)
		return nil, authURL, nil
	}

	// Mode 2: Local callback server
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", callbackPort))
	if err != nil {
		return nil, "", fmt.Errorf("failed to start callback server: %w", err)
	}

	server := &http.Server{
		ReadHeaderTimeout: 3 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/callback") && r.URL.Path != "/" {
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
					<h1 style="color: #cba6f7;">Success!</h1>
					<p>Poly is now authenticated. You can close this window and return to your terminal.</p>
				</body></html>
			`))
			codeChan <- code
		}),
	}

	go func() { _ = server.Serve(listener) }()
	defer server.Close()

	_ = openBrowser(authURL)

	select {
	case code := <-codeChan:
		tokens, err := ExchangeCode(cfg, code, pkce.Verifier)
		return tokens, authURL, err
	case err := <-errChan:
		return nil, authURL, err
	case <-time.After(5 * time.Minute):
		return nil, authURL, fmt.Errorf("OAuth timeout")
	}
}

// ExchangeCode exchanges the authorization code for tokens
func ExchangeCode(cfg OAuthConfig, code, verifier string) (*OAuthTokens, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {cfg.ClientID},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURI},
	}
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}
	if verifier != "" {
		form.Set("code_verifier", verifier)
	}

	return postTokenRequest(cfg.TokenURL, form)
}

// RefreshToken refreshes the access token using a refresh token
func RefreshToken(tokenURL, clientID, clientSecret, refreshToken string) (*OAuthTokens, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}

	tokens, err := postTokenRequest(tokenURL, form)
	if err != nil {
		return nil, err
	}

	if tokens.RefreshToken == "" {
		tokens.RefreshToken = refreshToken
	}

	return tokens, nil
}

func postTokenRequest(tokenURL string, form url.Values) (*OAuthTokens, error) {
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
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

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed (HTTP %d): %s", resp.StatusCode, security.SanitizeResponseBody(respBody))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
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

var (
	pendingOAuth   *pendingAuth
	pendingOAuthMu sync.Mutex
)

type pendingAuth struct {
	Verifier string
	State    string
}

func HasPendingAuth() bool {
	pendingOAuthMu.Lock()
	defer pendingOAuthMu.Unlock()
	return pendingOAuth != nil
}

func GetPendingAuth() (string, string) {
	pendingOAuthMu.Lock()
	defer pendingOAuthMu.Unlock()
	if pendingOAuth == nil {
		return "", ""
	}
	return pendingOAuth.Verifier, pendingOAuth.State
}

func ClearPendingAuth() {
	pendingOAuthMu.Lock()
	pendingOAuth = nil
	pendingOAuthMu.Unlock()
}
