package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestGeneratePKCE_NotNil(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}
	if pkce == nil {
		t.Fatal("GeneratePKCE() returned nil")
	}
}

func TestGeneratePKCE_VerifierLength(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}
	// 32 bytes -> 43 chars in base64url (no padding)
	if len(pkce.Verifier) != 43 {
		t.Errorf("expected verifier length 43, got %d", len(pkce.Verifier))
	}
}

func TestGeneratePKCE_ChallengeLength(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}
	// SHA256 = 32 bytes -> 43 chars in base64url (no padding)
	if len(pkce.Challenge) != 43 {
		t.Errorf("expected challenge length 43, got %d", len(pkce.Challenge))
	}
}

func TestGeneratePKCE_Base64URLFormat(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	// Verifier should be valid base64url (no +, /, or =)
	if strings.ContainsAny(pkce.Verifier, "+/=") {
		t.Errorf("verifier contains non-base64url chars: %s", pkce.Verifier)
	}
	if strings.ContainsAny(pkce.Challenge, "+/=") {
		t.Errorf("challenge contains non-base64url chars: %s", pkce.Challenge)
	}
}

func TestGeneratePKCE_ChallengeMatchesVerifier(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	// Recompute challenge from verifier
	hash := sha256.Sum256([]byte(pkce.Verifier))
	expected := base64.RawURLEncoding.EncodeToString(hash[:])

	if pkce.Challenge != expected {
		t.Errorf("challenge does not match SHA256(verifier)\nverifier:  %s\nchallenge: %s\nexpected:  %s", pkce.Verifier, pkce.Challenge, expected)
	}
}

func TestGeneratePKCE_Unique(t *testing.T) {
	verifiers := make(map[string]bool)
	for i := 0; i < 50; i++ {
		pkce, err := GeneratePKCE()
		if err != nil {
			t.Fatalf("GeneratePKCE() error: %v", err)
		}
		if verifiers[pkce.Verifier] {
			t.Errorf("duplicate verifier: %s", pkce.Verifier)
		}
		verifiers[pkce.Verifier] = true
	}
}

func TestGenerateState_NotEmpty(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error: %v", err)
	}
	if state == "" {
		t.Error("GenerateState() returned empty string")
	}
}

func TestGenerateState_Length(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error: %v", err)
	}
	// 16 bytes -> 22 chars in base64url (no padding)
	if len(state) != 22 {
		t.Errorf("expected state length 22, got %d", len(state))
	}
}

func TestGenerateState_Base64URLFormat(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error: %v", err)
	}
	if strings.ContainsAny(state, "+/=") {
		t.Errorf("state contains non-base64url chars: %s", state)
	}
}

func TestGenerateState_Unique(t *testing.T) {
	states := make(map[string]bool)
	for i := 0; i < 50; i++ {
		state, err := GenerateState()
		if err != nil {
			t.Fatalf("GenerateState() error: %v", err)
		}
		if states[state] {
			t.Errorf("duplicate state: %s", state)
		}
		states[state] = true
	}
}

func TestPKCECodesAlias(t *testing.T) {
	// PKCECodes is an alias for PKCE - verify it works
	var codes PKCECodes
	codes.Verifier = "test"
	codes.Challenge = "test"
	if codes.Verifier != "test" || codes.Challenge != "test" {
		t.Error("PKCECodes alias should work like PKCE")
	}
}
