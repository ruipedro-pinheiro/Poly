package skills

import (
	"os"
	"path/filepath"
	"strings"
)

// Skill represents a reusable prompt template loaded from a .md file
type Skill struct {
	Name    string
	Content string
	Path    string // filesystem path the skill was loaded from
}

// LoadSkills loads skills from ~/.poly/skills/ and .poly/skills/
// Project-level skills override global skills with the same name.
func LoadSkills() map[string]*Skill {
	skills := make(map[string]*Skill)

	// Load global skills first
	home, err := os.UserHomeDir()
	if err == nil {
		loadDir(filepath.Join(home, ".poly", "skills"), skills)
	}

	// Load project-level skills (overrides global)
	loadDir(filepath.Join(".poly", "skills"), skills)

	return skills
}

// GetSkill returns a skill by name, or nil if not found.
func GetSkill(name string) *Skill {
	skills := LoadSkills()
	return skills[strings.ToLower(name)]
}

// ListSkills returns all skill names sorted.
func ListSkills() []string {
	skills := LoadSkills()
	names := make([]string, 0, len(skills))
	for name := range skills {
		names = append(names, name)
	}
	// Simple sort
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}

// loadDir reads all .md files from a directory and adds them to the skills map.
func loadDir(dir string, skills map[string]*Skill) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}

		// Strip the leading "# name" header line if present
		skillName := strings.TrimSuffix(name, ".md")
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) >= 2 && strings.HasPrefix(lines[0], "# ") {
			content = strings.TrimSpace(lines[1])
		}

		skills[strings.ToLower(skillName)] = &Skill{
			Name:    skillName,
			Content: content,
			Path:    path,
		}
	}
}
