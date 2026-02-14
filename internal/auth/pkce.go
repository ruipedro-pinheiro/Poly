package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// PKCE holds the code verifier and challenge
type PKCE struct {
	Verifier  string
	Challenge string
}

// PKCECodes is an alias for PKCE
type PKCECodes = PKCE

// GeneratePKCE generates a PKCE code verifier and challenge
func GeneratePKCE() (*PKCE, error) {
	// Generate 32 random bytes for verifier
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, err
	}

	// Base64url encode the verifier
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// SHA256 hash the verifier and base64url encode it for the challenge
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCE{
		Verifier:  verifier,
		Challenge: challenge,
	}, nil
}

// GenerateState generates a random state string for OAuth
func GenerateState() (string, error) {
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(stateBytes), nil
}
