package security

import "regexp"

// BlockedPatterns contains dangerous command patterns that should be blocked.
// Shared between tools/bash.go and shell/executor.go.
var BlockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\brm\s+(-[rf]+\s+)+/($|\s)`),                                                      // rm -rf /
	regexp.MustCompile(`(?i)\brm\s+(-[rf]+\s+)+/(bin|boot|dev|etc|home|lib|opt|root|sbin|srv|sys|usr|var)\b`), // rm on system dirs
	regexp.MustCompile(`(?i)\bmkfs\b`),                                                                         // format disk
	regexp.MustCompile(`(?i)\bdd\s+.*of=/dev/`),                                                                // write to devices
	regexp.MustCompile(`:\(\)\s*\{\s*:\|\s*:&\s*\}\s*;?\s*:`),                                                  // fork bomb
	regexp.MustCompile(`(?i)\bcurl\b.*\|\s*(ba)?sh`),                                                           // curl | bash
	regexp.MustCompile(`(?i)\bwget\b.*\|\s*(ba)?sh`),                                                           // wget | bash
	regexp.MustCompile(`(?i)\bchmod\s+[0-7]*777\s+/`),                                                          // chmod 777 /
	regexp.MustCompile(`(?i)\bshutdown\b`),                                                                      // shutdown
	regexp.MustCompile(`(?i)\breboot\b`),                                                                        // reboot
}

// IsBlocked checks if a command matches any blocked pattern.
func IsBlocked(command string) bool {
	for _, pattern := range BlockedPatterns {
		if pattern.MatchString(command) {
			return true
		}
	}
	return false
}
