package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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

// Run executes a command inside a container, mounting cwd as /workspace.
// Returns stdout+stderr combined and the exit code.
func Run(ctx context.Context, command string, cwd string) (output string, exitCode int, err error) {
	rt := Detect()
	if rt == "" {
		return "", 1, fmt.Errorf("no container runtime found (install podman or docker)")
	}

	args := []string{
		"run", "--rm",
		"-v", cwd + ":/workspace",
		"-w", "/workspace",
		"--network=host",
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
