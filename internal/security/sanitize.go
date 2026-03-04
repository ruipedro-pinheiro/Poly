package security

import (
	"encoding/json"
)

// SanitizeResponseBody extracts a safe error message from an HTTP response body.
// This function is critical to prevent API tokens or sensitive data from leaking into the UI.
func SanitizeResponseBody(body []byte) string {
	if len(body) == 0 {
		return "(empty response)"
	}

	// Try structured JSON extraction — covers OpenAI, Anthropic, Google, GitHub
	if msg := extractJSONErrorMessage(body); msg != "" {
		return truncate(msg, 500)
	}

	// Fallback: strictly avoid dumping raw body if it's not JSON.
	// This prevents leaking keys that might be in an unexpected HTML/text error page.
	return "unrecognized error format (raw body hidden for security)"
}

// extractJSONErrorMessage tries multiple common API error formats.
func extractJSONErrorMessage(body []byte) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return ""
	}

	// OpenAI / xAI / GitHub format: {"error": {"message": "...", "type": "...", "code": "..."}}
	if errField, ok := obj["error"]; ok {
		switch v := errField.(type) {
		case map[string]interface{}:
			if msg, ok := v["message"].(string); ok && msg != "" {
				return msg
			}
			if desc, ok := v["error"].(string); ok && desc != "" {
				return desc
			}
		case string:
			if v != "" {
				return v
			}
		}
	}

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
