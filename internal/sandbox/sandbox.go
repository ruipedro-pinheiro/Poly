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
	imageReady bool
	imageMu    sync.Mutex
	imageErr   error
)

// Detect returns "podman", "docker", or "" if neither is available.
func Detect() string {
	runtimeOnce.Do(func() {
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
func EnsureImage() error {
	imageMu.Lock()
	defer imageMu.Unlock()

	if imageReady {
		return nil
	}
	if imageErr != nil {
		return imageErr
	}

	rt := Detect()
	if rt == "" {
		imageErr = fmt.Errorf("no container runtime found (install podman or docker)")
		return imageErr
	}

	check := exec.Command(rt, "image", "inspect", Image)
	if check.Run() == nil {
		imageReady = true
		return nil
	}

	pull := exec.Command(rt, "pull", Image)
	var stderr bytes.Buffer
	pull.Stderr = &stderr
	if err := pull.Run(); err != nil {
		imageErr = fmt.Errorf(
			"sandbox image '%s' not found locally and pull failed:\n%s\n\n"+
				"Fix: run '%s pull %s' manually",
			Image, strings.TrimSpace(stderr.String()), rt, Image)
		return imageErr
	}
	imageReady = true
	return nil
}

// ResetImageCheck allows retrying the image check
func ResetImageCheck() {
	imageMu.Lock()
	defer imageMu.Unlock()
	imageReady = false
	imageErr = nil
}

// Run executes a command inside a hardened container.
func Run(ctx context.Context, command string, cwd string) (output string, exitCode int, err error) {
	rt := Detect()
	if rt == "" {
		return "", 1, fmt.Errorf("no container runtime found")
	}

	if err := EnsureImage(); err != nil {
		return "", 1, err
	}

	// Hardened sandbox arguments
	args := []string{
		"run", "--rm",
		"-v", cwd + ":/workspace:Z",
		"-w", "/workspace",
		"--network=none",           // No internet access
		"--read-only",              // Read-only root filesystem
		"--tmpfs", "/tmp:rw,size=64m", // Small writable /tmp
		"--cap-drop=all",           // Drop all capabilities
		"--security-opt", "no-new-privileges", // Prevent privilege escalation
		"--memory=512m",            // Max 512MB RAM
		"--cpus=1",                 // Max 1 CPU core
		"--pids-limit=50",          // Prevent fork bombs
	}

	// Podman-specific hardening: run as the same user as the host
	if rt == "podman" {
		args = append(args, "--userns=keep-id")
	}

	args = append(args, Image, "sh", "-c", command)

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
