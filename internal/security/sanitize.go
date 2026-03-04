package security

import (
	"encoding/json"
	"strings"
)

// SanitizeResponseBody extracts a safe error message from an HTTP response body.
// API error responses can contain tokens, keys, or sensitive internal details.
// This function:
//  1. Tries to extract a structured error message from JSON (all major APIs return JSON errors)
//  2. Falls back to a truncated version of the raw body (max 200 chars)
//  3. Never returns raw response bodies verbatim
//
// Callers replace string(body) with SanitizeResponseBody(body) in error formatting.
func SanitizeResponseBody(body []byte) string {
	if len(body) == 0 {
		return "(empty response)"
	}

	// Try structured JSON extraction — covers OpenAI, Anthropic, Google, GitHub
	if msg := extractJSONErrorMessage(body); msg != "" {
		return truncate(msg, 500)
	}

	// Fallback: truncate raw body aggressively
	s := strings.TrimSpace(string(body))
	return truncate(s, 200)
}

// extractJSONErrorMessage tries multiple common API error formats.
func extractJSONErrorMessage(body []byte) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return ""
	}

	// OpenAI / xAI / GitHub format: {"error": {"message": "...", "type": "...", "code": "..."}}
	// Also Anthropic: {"type": "error", "error": {"type": "...", "message": "..."}}
	if errField, ok := obj["error"]; ok {
		switch v := errField.(type) {
		case map[string]interface{}:
			if msg, ok := v["message"].(string); ok && msg != "" {
				return msg
			}
			// Some APIs use "error": {"error": "description"}
			if desc, ok := v["error"].(string); ok && desc != "" {
				return desc
			}
		case string:
			// Simple: {"error": "invalid_grant"}
			if v != "" {
				return v
			}
		}
	}

	// Google format: {"error": {"message": "...", "status": "..."}} — already handled above

	// Simple "message" field: {"message": "Not Found"}
	if msg, ok := obj["message"].(string); ok && msg != "" {
		return msg
	}

	// OAuth error format: {"error": "invalid_grant", "error_description": "..."}
	if errDesc, ok := obj["error_description"].(string); ok && errDesc != "" {
		return errDesc
	}

	return ""
}

// truncate limits a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
