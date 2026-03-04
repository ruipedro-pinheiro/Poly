package tools

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SystemInfoTool returns system information
type SystemInfoTool struct{}

func (t *SystemInfoTool) Name() string {
	return "system_info"
}

func (t *SystemInfoTool) Description() string {
	return "Get system information (OS, architecture, memory, disk, shell, git info, installed languages). Use this to understand the user's environment."
}

func (t *SystemInfoTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *SystemInfoTool) Execute(args map[string]interface{}) ToolResult {
	var sb strings.Builder

	// OS / Architecture
	sb.WriteString("=== System ===\n")
	sb.WriteString(fmt.Sprintf("OS: %s\n", runtime.GOOS))
	sb.WriteString(fmt.Sprintf("Arch: %s\n", runtime.GOARCH))

	if kernel := cmdOutput("uname", "-r"); kernel != "" {
		sb.WriteString(fmt.Sprintf("Kernel: %s\n", kernel))
	}
	if hostname := cmdOutput("hostname"); hostname != "" {
		sb.WriteString(fmt.Sprintf("Hostname: %s\n", hostname))
	}

	// Shell & paths
	sb.WriteString(fmt.Sprintf("Shell: %s\n", os.Getenv("SHELL")))
	sb.WriteString(fmt.Sprintf("Home: %s\n", os.Getenv("HOME")))
	if cwd, err := os.Getwd(); err == nil {
		sb.WriteString(fmt.Sprintf("CWD: %s\n", cwd))
	}
	sb.WriteString(fmt.Sprintf("User: %s\n", os.Getenv("USER")))

	// Memory (Linux only, /proc/meminfo)
	if runtime.GOOS == "linux" {
		if mem := linuxMemInfo(); mem != "" {
			sb.WriteString("\n=== Memory ===\n")
			sb.WriteString(mem)
		}
	}

	// Disk
	if disk := cmdOutput("df", "-h", "--output=size,avail,pcent,target", "/"); disk != "" {
		sb.WriteString("\n=== Disk (/) ===\n")
		sb.WriteString(disk + "\n")
	}

	// Git info (if in a repo)
	if branch := cmdOutput("git", "branch", "--show-current"); branch != "" {
		sb.WriteString("\n=== Git ===\n")
		sb.WriteString(fmt.Sprintf("Branch: %s\n", branch))

		if remote := cmdOutput("git", "remote", "-v"); remote != "" {
			// Just show first line (fetch)
			lines := strings.Split(remote, "\n")
			if len(lines) > 0 {
				sb.WriteString(fmt.Sprintf("Remote: %s\n", lines[0]))
			}
		}

		if status := cmdOutput("git", "status", "--porcelain"); status == "" {
			sb.WriteString("Status: clean\n")
		} else {
			count := len(strings.Split(strings.TrimSpace(status), "\n"))
			sb.WriteString(fmt.Sprintf("Status: %d modified/untracked files\n", count))
		}
	}

	// Installed languages
	sb.WriteString("\n=== Languages ===\n")
	langs := []struct {
		name string
		cmd  string
		args []string
	}{
		{"Go", "go", []string{"version"}},
		{"Node", "node", []string{"--version"}},
		{"Python", "python3", []string{"--version"}},
		{"Rust", "rustc", []string{"--version"}},
		{"GCC", "gcc", []string{"--version"}},
	}
	for _, l := range langs {
		if v := cmdOutput(l.cmd, l.args...); v != "" {
			// Take first line only (gcc outputs multiple)
			first := strings.Split(v, "\n")[0]
			sb.WriteString(fmt.Sprintf("%s: %s\n", l.name, first))
		}
	}

	return ToolResult{Content: sb.String()}
}

// cmdOutput runs a command and returns trimmed stdout, or empty string on error
func cmdOutput(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// linuxMemInfo reads /proc/meminfo and returns total/available memory
func linuxMemInfo() string {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	var totalKB, availKB uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			_, _ = fmt.Sscanf(line, "MemTotal: %d kB", &totalKB)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			_, _ = fmt.Sscanf(line, "MemAvailable: %d kB", &availKB)
		}
		if totalKB > 0 && availKB > 0 {
			break
		}
	}

	if totalKB == 0 {
		return ""
	}

	totalGB := float64(totalKB) / 1024 / 1024
	availGB := float64(availKB) / 1024 / 1024
	return fmt.Sprintf("Total: %.1f GB\nAvailable: %.1f GB\n", totalGB, availGB)
}
