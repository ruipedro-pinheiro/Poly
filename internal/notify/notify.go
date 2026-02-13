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
		exec.Command("notify-send", title, body).Start()
	case "darwin":
		exec.Command("osascript", "-e",
			`display notification "`+body+`" with title "`+title+`"`).Start()
	}
}
