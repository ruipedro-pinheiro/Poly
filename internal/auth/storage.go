package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ProviderAuth holds authentication info for a provider
type ProviderAuth struct {
	Type     string       `json:"type"` // "oauth" or "apikey"
	Provider string       `json:"provider"`
	Tokens   *OAuthTokens `json:"tokens,omitempty"`
	APIKey   string       `json:"api_key,omitempty"`
}

// Storage holds all provider auth data
type Storage struct {
	Providers map[string]*ProviderAuth `json:"providers"`
	mu        sync.RWMutex
}

var (
	storage     *Storage
	storageOnce sync.Once
	storagePath string
)

// getStoragePath returns the path to the auth storage file
func getStoragePath() string {
	if storagePath != "" {
		return storagePath
	}

	// Use XDG config dir or fallback to ~/.config
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}

	polyDir := filepath.Join(configDir, "poly")
	os.MkdirAll(polyDir, 0755)

	storagePath = filepath.Join(polyDir, "auth.json")
	return storagePath
}

// GetStorage returns the singleton storage instance
func GetStorage() *Storage {
	storageOnce.Do(func() {
		storage = &Storage{
			Providers: make(map[string]*ProviderAuth),
		}
		storage.load()
	})
	return storage
}

// load reads the storage from disk
func (s *Storage) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(getStoragePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, s)
}

// save writes the storage to disk
func (s *Storage) save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(getStoragePath(), data, 0600)
}

// GetAuth returns the auth for a provider
func (s *Storage) GetAuth(provider string) *ProviderAuth {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Providers[provider]
}

// SetAuth sets the auth for a provider
func (s *Storage) SetAuth(provider string, auth *ProviderAuth) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Providers[provider] = auth
	return s.save()
}

// SetOAuthTokens sets OAuth tokens for a provider
func (s *Storage) SetOAuthTokens(provider string, tokens *OAuthTokens) error {
	auth := &ProviderAuth{
		Type:     "oauth",
		Provider: provider,
		Tokens:   tokens,
	}
	return s.SetAuth(provider, auth)
}

// SetAPIKey sets an API key for a provider
func (s *Storage) SetAPIKey(provider string, apiKey string) error {
	auth := &ProviderAuth{
		Type:     "apikey",
		Provider: provider,
		APIKey:   apiKey,
	}
	return s.SetAuth(provider, auth)
}

// RemoveAuth removes auth for a provider
func (s *Storage) RemoveAuth(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Providers, provider)
	return s.save()
}

// IsConnected returns true if a provider has valid auth
func (s *Storage) IsConnected(provider string) bool {
	auth := s.GetAuth(provider)
	if auth == nil {
		return false
	}

	if auth.Type == "apikey" && auth.APIKey != "" {
		return true
	}

	if auth.Type == "oauth" && auth.Tokens != nil && auth.Tokens.AccessToken != "" {
		return true
	}

	return false
}

// GetAccessToken returns the access token or API key for a provider
// Refreshes OAuth token if needed
func (s *Storage) GetAccessToken(provider string) (string, error) {
	auth := s.GetAuth(provider)
	if auth == nil {
		return "", nil
	}

	// API key is simple
	if auth.Type == "apikey" && auth.APIKey != "" {
		return auth.APIKey, nil
	}

	// OAuth needs refresh check
	if auth.Type == "oauth" && auth.Tokens != nil {
		// Check if token needs refresh (10 min buffer)
		if auth.Tokens.ExpiresAt < (time.Now().Unix() + 600) {
			// Refresh token
			var newTokens *OAuthTokens
			var err error

			switch provider {
			case "claude":
				newTokens, err = RefreshAnthropicToken(auth.Tokens.RefreshToken)
			case "gpt":
				newTokens, err = RefreshOpenAIToken(auth.Tokens.RefreshToken)
			case "gemini":
				newTokens, err = RefreshGoogleToken(auth.Tokens.RefreshToken)
			default:
				return auth.Tokens.AccessToken, nil
			}

			if err != nil {
				return "", err
			}

			s.SetOAuthTokens(provider, newTokens)
			return newTokens.AccessToken, nil
		}

		return auth.Tokens.AccessToken, nil
	}

	return "", nil
}
