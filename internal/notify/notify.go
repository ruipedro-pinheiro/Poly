package notify

import (
	"os/exec"
	"runtime"
)

// Send sends a desktop notification using the OS notification system.
// Uses .Start() to avoid blocking the caller.
func Send(title, body string) {
	switch runtime.GOOS {
	case "linux":
		_ = exec.Command("notify-send", title, body).Start()
	case "darwin":
		script := `on run argv
	display notification (item 1 of argv) with title (item 2 of argv)
end run`
		_ = exec.Command("osascript", "-e", script, body, title).Start()
	}
}
