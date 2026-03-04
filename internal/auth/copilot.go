package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pedromelo/poly/internal/security"
)

const (
	// CopilotClientID is the GitHub OAuth app client ID used for the device flow.
	// This is the well-known VS Code Copilot client ID.
	CopilotClientID = "01ab8ac9400c4e429b23"

	CopilotDeviceCodeURL  = "https://github.com/login/device/code"
	CopilotAccessTokenURL = "https://github.com/login/oauth/access_token"
	CopilotTokenURL       = "https://api.github.com/copilot_internal/v2/token"
)

// DeviceFlowResponse holds the response from GitHub's device flow initiation.
type DeviceFlowResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// StartCopilotDeviceFlow initiates the GitHub Device Flow OAuth.
// Returns the device flow info (user code, verification URL) for display in the TUI.
// The browser is opened automatically to the verification URL.
func StartCopilotDeviceFlow() (*DeviceFlowResponse, error) {
	form := url.Values{
		"client_id": {CopilotClientID},
		"scope":     {""},
	}

	req, err := http.NewRequest("POST", CopilotDeviceCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device code request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device code request failed (HTTP %d): %s", resp.StatusCode, security.SanitizeResponseBody(body))
	}

	var result DeviceFlowResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode device code response: %w", err)
	}

	if result.DeviceCode == "" || result.UserCode == "" {
		return nil, fmt.Errorf("empty device code or user code in response")
	}

	// Try to open the verification URL in the browser
	openBrowser(result.VerificationURI)

	return &result, nil
}

// PollCopilotDeviceFlow polls GitHub until the user authorizes the device.
// This blocks until authorized, the device code expires, or the context is cancelled.
// On success, it exchanges the GitHub token for a Copilot session token.
func PollCopilotDeviceFlow(ctx context.Context, deviceCode string, interval int) (*OAuthTokens, error) {
	if interval < 5 {
		interval = 5
	}

	client := &http.Client{Timeout: 30 * time.Second}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}

		form := url.Values{
			"client_id":   {CopilotClientID},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}

		req, err := http.NewRequestWithContext(ctx, "POST", CopilotAccessTokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			// Network error — keep polling
			continue
		}

		var result struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			Scope       string `json:"scope"`
			Error       string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		switch result.Error {
		case "authorization_pending":
			// User hasn't authorized yet — keep polling
			continue
		case "slow_down":
			// GitHub is telling us to slow down — increase interval
			interval += 5
			ticker.Reset(time.Duration(interval) * time.Second)
			continue
		case "expired_token":
			return nil, fmt.Errorf("device code expired — please try again")
		case "access_denied":
			return nil, fmt.Errorf("authorization denied by user")
		case "":
			// Success! Got GitHub token
			if result.AccessToken == "" {
				return nil, fmt.Errorf("empty access token in response")
			}
			// Exchange GitHub token for Copilot session token
			return getCopilotSessionToken(result.AccessToken)
		default:
			return nil, fmt.Errorf("unexpected error: %s", result.Error)
		}
	}
}

// getCopilotSessionToken exchanges a GitHub token for a Copilot session token.
// The GitHub token is stored as RefreshToken for future session token refreshes.
func getCopilotSessionToken(githubToken string) (*OAuthTokens, error) {
	req, err := http.NewRequest("GET", CopilotTokenURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("copilot token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("GitHub token is invalid or expired — please re-authenticate")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("GitHub Copilot is not enabled for this account — check your subscription")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("copilot token request failed (HTTP %d): %s", resp.StatusCode, security.SanitizeResponseBody(body))
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode copilot token response: %w", err)
	}

	if result.Token == "" {
		return nil, fmt.Errorf("empty token in copilot response")
	}

	return &OAuthTokens{
		AccessToken:  result.Token,
		RefreshToken: githubToken, // GitHub token stored for refreshing
		ExpiresAt:    result.ExpiresAt,
	}, nil
}

// RefreshCopilotToken refreshes the Copilot session token using the stored GitHub token.
// The GitHub token (stored as RefreshToken) is long-lived.
// The Copilot session token expires every ~30 minutes.
func RefreshCopilotToken(githubToken string) (*OAuthTokens, error) {
	return getCopilotSessionToken(githubToken)
}
