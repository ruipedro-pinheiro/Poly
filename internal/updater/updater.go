package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CurrentVersion is the current version of Poly.
// TODO: embed from build using ldflags
const CurrentVersion = "0.1.0"

const (
	githubReleasesURL = "https://api.github.com/repos/pedromelo/poly/releases/latest"
	checkInterval     = 24 * time.Hour
	httpTimeout       = 3 * time.Second
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdate checks GitHub for a newer release.
// Returns the new version string if available, empty string if up to date or on error.
func CheckForUpdate() string {
	if !ShouldCheck() {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", githubReleasesURL, nil)
	if err != nil {
		return ""
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ""
	}

	// Save check timestamp regardless of result
	saveCheckTimestamp()

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(CurrentVersion, "v")

	if latest != "" && latest != current {
		return latest
	}
	return ""
}

// ShouldCheck returns true if we haven't checked in the last 24 hours.
func ShouldCheck() bool {
	path := checkFilePath()
	if path == "" {
		return true
	}

	info, err := os.Stat(path)
	if err != nil {
		return true // file doesn't exist, should check
	}

	return time.Since(info.ModTime()) > checkInterval
}

func checkFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".poly", "last_update_check")
}

func saveCheckTimestamp() {
	path := checkFilePath()
	if path == "" {
		return
	}
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0700)
	_ = os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0600)
}
