package tools

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const maxBackups = 20

// FileBackup stores the original content of a file before modification
type FileBackup struct {
	Path    string
	Content []byte
	Time    time.Time
}

var (
	backups   []FileBackup
	backupsMu sync.Mutex
)

// BackupFile saves the current content of a file before modification.
// If the file doesn't exist yet, no backup is created.
func BackupFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // file doesn't exist yet, nothing to backup
	}

	backupsMu.Lock()
	defer backupsMu.Unlock()

	backups = append(backups, FileBackup{
		Path:    path,
		Content: data,
		Time:    time.Now(),
	})

	// Trim to max
	if len(backups) > maxBackups {
		backups = backups[len(backups)-maxBackups:]
	}
}

// GetLastBackup returns the most recent backup for a given path, or nil.
func GetLastBackup(path string) *FileBackup {
	backupsMu.Lock()
	defer backupsMu.Unlock()

	for i := len(backups) - 1; i >= 0; i-- {
		if backups[i].Path == path {
			return &backups[i]
		}
	}
	return nil
}

// GetAllBackups returns a copy of all backups (most recent last).
func GetAllBackups() []FileBackup {
	backupsMu.Lock()
	defer backupsMu.Unlock()

	result := make([]FileBackup, len(backups))
	copy(result, backups)
	return result
}

// RevertLast restores the most recently backed up file and removes that backup.
func RevertLast() (string, error) {
	backupsMu.Lock()
	defer backupsMu.Unlock()

	if len(backups) == 0 {
		return "", fmt.Errorf("no backups available")
	}

	last := backups[len(backups)-1]
	backups = backups[:len(backups)-1]

	if err := os.WriteFile(last.Path, last.Content, 0644); err != nil {
		return last.Path, fmt.Errorf("failed to restore %s: %w", last.Path, err)
	}

	return last.Path, nil
}

// RevertFile restores the most recent backup for a specific file path.
func RevertFile(path string) error {
	backupsMu.Lock()
	defer backupsMu.Unlock()

	for i := len(backups) - 1; i >= 0; i-- {
		if backups[i].Path == path {
			b := backups[i]
			// Remove this backup entry
			backups = append(backups[:i], backups[i+1:]...)

			if err := os.WriteFile(b.Path, b.Content, 0644); err != nil {
				return fmt.Errorf("failed to restore %s: %w", b.Path, err)
			}
			return nil
		}
	}

	return fmt.Errorf("no backup found for %s", path)
}
