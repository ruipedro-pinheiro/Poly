package tui

import (
	"os/exec"
	"runtime"
	"strings"
)

// getClipboardContent retrieves content from system clipboard synchronously
func getClipboardContent() string {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		// Try wl-paste first (Wayland), then xclip (X11)
		if _, err := exec.LookPath("wl-paste"); err == nil {
			cmd = exec.Command("wl-paste", "--no-newline")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
		}
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "windows":
		cmd = exec.Command("powershell", "-command", "Get-Clipboard")
	default:
		return ""
	}

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// getClipboardImage retrieves image data from system clipboard
// Returns image data, media type, and success bool
func getClipboardImage() ([]byte, string, bool) {
	var cmd *exec.Cmd
	mediaType := "image/png"

	switch runtime.GOOS {
	case "linux":
		// Try wl-paste first (Wayland), then xclip (X11)
		if _, err := exec.LookPath("wl-paste"); err == nil {
			// Check if clipboard has image
			checkCmd := exec.Command("wl-paste", "--list-types")
			output, err := checkCmd.Output()
			if err != nil {
				return nil, "", false
			}
			types := string(output)
			if strings.Contains(types, "image/png") {
				cmd = exec.Command("wl-paste", "--type", "image/png")
				mediaType = "image/png"
			} else if strings.Contains(types, "image/jpeg") {
				cmd = exec.Command("wl-paste", "--type", "image/jpeg")
				mediaType = "image/jpeg"
			} else {
				return nil, "", false
			}
		} else {
			// xclip fallback
			cmd = exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o")
		}
	case "darwin":
		// macOS - use osascript to check for image, then pngpaste
		if _, err := exec.LookPath("pngpaste"); err == nil {
			cmd = exec.Command("pngpaste", "-")
		} else {
			return nil, "", false
		}
	default:
		return nil, "", false
	}

	if cmd == nil {
		return nil, "", false
	}

	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil, "", false
	}

	return output, mediaType, true
}
