package tui

import (
	"os"
	"sort"
	"strings"

	"github.com/pedromelo/poly/internal/skills"
)

// providerMentions is the list of all @provider mentions
var providerMentions = []string{
	"@all",
	"@claude",
	"@gemini",
	"@gpt",
	"@grok",
}

// completionState tracks the current tab completion session
type completionState struct {
	active     bool     // whether we're in a completion cycle
	candidates []string // current list of candidates
	prefix     string   // the prefix being completed
	index      int      // current index in candidates (for cycling)
	startPos   int      // cursor position where the prefix starts in the input
}

// getCompletions returns possible completions for the current input text.
// It looks at the word being typed at the cursor position and returns
// matching candidates along with the prefix and its start position.
func (m *Model) getCompletions(input string, cursorPos int) (candidates []string, prefix string, startPos int) {
	if cursorPos > len(input) {
		cursorPos = len(input)
	}
	if cursorPos == 0 {
		return nil, "", 0
	}

	// Find the start of the current "word" (going backwards from cursor)
	textBefore := input[:cursorPos]

	// Find the last space before cursor to isolate the current word
	wordStart := strings.LastIndex(textBefore, " ") + 1
	word := textBefore[wordStart:]

	if word == "" {
		return nil, "", 0
	}

	// Slash command completion: input starts with /
	if strings.HasPrefix(word, "/") {
		prefix = word
		startPos = wordStart
		lower := strings.ToLower(prefix)

		// Use the command registry for slash command names
		for _, cmd := range m.commands.Names() {
			if strings.HasPrefix(cmd, lower) {
				candidates = append(candidates, cmd)
			}
		}

		// Also complete skill names as /skillname
		for _, name := range skills.ListSkills() {
			skillCmd := "/" + name
			if strings.HasPrefix(skillCmd, lower) {
				// Avoid duplicates with built-in commands
				dup := false
				for _, c := range candidates {
					if c == skillCmd {
						dup = true
						break
					}
				}
				if !dup {
					candidates = append(candidates, skillCmd)
				}
			}
		}
		return candidates, prefix, startPos
	}

	// @mention completion
	if strings.HasPrefix(word, "@") {
		prefix = word
		startPos = wordStart
		lower := strings.ToLower(prefix)

		// Provider mentions
		for _, p := range providerMentions {
			if strings.HasPrefix(p, lower) {
				candidates = append(candidates, p)
			}
		}

		// File mentions: list files in cwd that match
		filePrefix := word[1:] // remove the @
		if filePrefix != "" {
			fileCandidates := matchFiles(filePrefix)
			for _, f := range fileCandidates {
				candidates = append(candidates, "@"+f)
			}
		}

		return candidates, prefix, startPos
	}

	return nil, "", 0
}

// matchFiles returns filenames from the cwd that start with the given prefix
func matchFiles(prefix string) []string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil
	}

	var matches []string
	lower := strings.ToLower(prefix)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(strings.ToLower(name), lower) {
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)
	return matches
}
