package permission

import "testing"

func TestClassifyBashCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    Level
	}{
		// Safe commands
		{"ls is safe", "ls -la", Allow},
		{"cat is safe", "cat file.txt", Allow},
		{"pwd is safe", "pwd", Allow},
		{"git status is safe", "git status", Allow},
		{"git log is safe", "git log --oneline", Allow},
		{"git diff is safe", "git diff HEAD", Allow},
		{"go version is safe", "go version", Allow},
		{"echo is safe", "echo hello", Allow},
		{"whoami is safe", "whoami", Allow},

		// Banned commands
		{"rm -rf / is banned", "rm -rf /", Deny},
		{"rm -rf ~ is banned", "rm -rf ~", Deny},
		{"sudo rm is banned", "sudo rm -rf /tmp", Deny},
		{"fork bomb is banned", ":(){:|:&};:", Deny},
		{"curl pipe bash is banned", "curl | bash", Deny},
		{"sudo shutdown is banned", "sudo shutdown -h now", Deny},
		{"dd is banned", "dd if=/dev/zero of=/dev/sda", Deny},
		{"shutdown is banned", "shutdown now", Deny},
		{"reboot is banned", "reboot", Deny},

		// Ask commands (neither safe nor banned)
		{"go build requires ask", "go build ./...", Ask},
		{"make requires ask", "make install", Ask},
		{"git push requires ask", "git push origin main", Ask},
		{"npm install requires ask", "npm install express", Ask},
		{"mkdir requires ask", "mkdir -p /tmp/test", Ask},
		{"unknown command requires ask", "some-random-command", Ask},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyBashCommand(tt.command)
			if got != tt.want {
				t.Errorf("ClassifyBashCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestClassifyBashCommandCaseInsensitive(t *testing.T) {
	// Verify case insensitivity
	if ClassifyBashCommand("LS -la") != Allow {
		t.Error("ClassifyBashCommand should be case insensitive for safe commands")
	}
	if ClassifyBashCommand("RM -RF /") != Deny {
		t.Error("ClassifyBashCommand should be case insensitive for banned commands")
	}
}

func TestClassifyBashCommandWhitespace(t *testing.T) {
	if ClassifyBashCommand("  ls -la  ") != Allow {
		t.Error("ClassifyBashCommand should handle leading/trailing whitespace")
	}
}

func TestClassifyTool(t *testing.T) {
	tests := []struct {
		name string
		tool string
		want Level
	}{
		{"read_file is Allow", "read_file", Allow},
		{"glob is Allow", "glob", Allow},
		{"grep is Allow", "grep", Allow},
		{"git_status is Allow", "git_status", Allow},
		{"bash is Ask", "bash", Ask},
		{"write_file is Ask", "write_file", Ask},
		{"edit_file is Ask", "edit_file", Ask},
		{"web_fetch is Ask", "web_fetch", Ask},
		{"unknown tool is Ask", "unknown_tool", Ask},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyTool(tt.tool)
			if got != tt.want {
				t.Errorf("ClassifyTool(%q) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}

func TestPolicyShouldAsk(t *testing.T) {
	t.Run("normal mode asks for write tools", func(t *testing.T) {
		p := DefaultPolicy()
		if !p.ShouldAsk("bash") {
			t.Error("Default policy should ask for bash")
		}
		if p.ShouldAsk("read_file") {
			t.Error("Default policy should not ask for read_file")
		}
	})

	t.Run("yolo mode never asks", func(t *testing.T) {
		p := &Policy{YoloMode: true}
		if p.ShouldAsk("bash") {
			t.Error("YOLO mode should never ask, even for bash")
		}
		if p.ShouldAsk("write_file") {
			t.Error("YOLO mode should never ask, even for write_file")
		}
	})
}

func TestIsReadOnly(t *testing.T) {
	if !IsReadOnly("read_file") {
		t.Error("read_file should be read-only")
	}
	if !IsReadOnly("glob") {
		t.Error("glob should be read-only")
	}
	if IsReadOnly("bash") {
		t.Error("bash should not be read-only")
	}
	if IsReadOnly("write_file") {
		t.Error("write_file should not be read-only")
	}
}
