package security

import "strings"

// sensitiveEnvSuffixes are suffixes that indicate a sensitive environment variable.
var sensitiveEnvSuffixes = []string{
	"_API_KEY",
	"_SECRET",
	"_TOKEN",
	"_PASSWORD",
	"_CREDENTIAL",
	"_CREDENTIALS",
}

// sensitiveEnvExact are exact environment variable names known to contain secrets.
var sensitiveEnvExact = map[string]bool{
	"ANTHROPIC_API_KEY":         true,
	"OPENAI_API_KEY":            true,
	"GOOGLE_API_KEY":            true,
	"XAI_API_KEY":               true,
	"GITHUB_TOKEN":              true,
	"GH_TOKEN":                  true,
	"AWS_SECRET_ACCESS_KEY":     true,
	"AWS_SESSION_TOKEN":         true,
	"HOMEBREW_GITHUB_API_TOKEN": true,
}

// SafeEnv returns os.Environ() with sensitive variables removed.
// Use this instead of os.Environ() when spawning child processes that
// the user (or an LLM) can inspect (e.g., bash tool, hooks).
func SafeEnv(environ []string) []string {
	safe := make([]string, 0, len(environ))
	for _, e := range environ {
		key, _, found := strings.Cut(e, "=")
		if !found {
			safe = append(safe, e)
			continue
		}
		if isSensitiveEnvVar(key) {
			continue
		}
		safe = append(safe, e)
	}
	return safe
}

// isSensitiveEnvVar checks if an environment variable name looks sensitive.
func isSensitiveEnvVar(key string) bool {
	upper := strings.ToUpper(key)

	if sensitiveEnvExact[upper] {
		return true
	}

	for _, suffix := range sensitiveEnvSuffixes {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}

	return strings.Contains(upper, "PASSWORD") ||
		strings.Contains(upper, "CREDENTIAL")
}
