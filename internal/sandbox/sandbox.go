package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

var (
	// Enabled controls whether sandbox mode is active
	Enabled bool

	// Image is the container image to use (default: alpine:latest)
	Image = "alpine:latest"

	// runtime caches the detected container runtime
	runtime     string
	runtimeOnce sync.Once

	// imageReady tracks whether the image has been verified/pulled
	imageReady     bool
	imageReadyOnce sync.Once
	imageErr       error
)

// Detect returns "podman", "docker", or "" if neither is available.
// The result is cached after first call.
func Detect() string {
	runtimeOnce.Do(func() {
		// Prefer podman (Fedora default, rootless)
		if _, err := exec.LookPath("podman"); err == nil {
			runtime = "podman"
			return
		}
		if _, err := exec.LookPath("docker"); err == nil {
			runtime = "docker"
			return
		}
		runtime = ""
	})
	return runtime
}

// Available returns true if a container runtime was detected
func Available() bool {
	return Detect() != ""
}

// EnsureImage checks if the sandbox image exists locally, pulls it if not.
// Returns nil on success, error with clear instructions on failure.
func EnsureImage() error {
	imageReadyOnce.Do(func() {
		rt := Detect()
		if rt == "" {
			imageErr = fmt.Errorf("no container runtime found (install podman or docker)")
			return
		}

		// Check if image exists locally
		check := exec.Command(rt, "image", "inspect", Image)
		if check.Run() == nil {
			imageReady = true
			return
		}

		// Try to pull
		pull := exec.Command(rt, "pull", Image)
		var stderr bytes.Buffer
		pull.Stderr = &stderr
		if err := pull.Run(); err != nil {
			imageErr = fmt.Errorf(
				"sandbox image '%s' not found locally and pull failed:\n%s\n\n"+
					"Fix: run '%s pull %s' manually, or use 'make sandbox-setup'",
				Image, strings.TrimSpace(stderr.String()), rt, Image)
			return
		}
		imageReady = true
	})
	return imageErr
}

// ResetImageCheck allows retrying the image check (e.g. after manual pull)
func ResetImageCheck() {
	imageReadyOnce = sync.Once{}
	imageReady = false
	imageErr = nil
}

// Run executes a command inside a container, mounting cwd as /workspace.
// Auto-pulls the image on first use if needed.
// Returns stdout+stderr combined and the exit code.
func Run(ctx context.Context, command string, cwd string) (output string, exitCode int, err error) {
	rt := Detect()
	if rt == "" {
		return "", 1, fmt.Errorf("no container runtime found (install podman or docker)")
	}

	// Ensure image is available (auto-pull on first use)
	if err := EnsureImage(); err != nil {
		return "", 1, err
	}

	args := []string{
		"run", "--rm",
		"-v", cwd + ":/workspace:Z",
		"-w", "/workspace",
		"--network=none",
		"--read-only",
		"--tmpfs", "/tmp:rw,size=64m",
		Image,
		"sh", "-c", command,
	}

	cmd := exec.CommandContext(ctx, rt, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	combined := stdout.String()
	if stderr.Len() > 0 {
		if combined != "" {
			combined += "\n"
		}
		combined += stderr.String()
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return combined, exitErr.ExitCode(), nil
		}
		return combined, 1, err
	}

	return combined, 0, nil
}
