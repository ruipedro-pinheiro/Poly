package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePath_EmptyPath(t *testing.T) {
	_, err := ValidatePath("")
	if err == nil {
		t.Error("ValidatePath('') should return error")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Errorf("expected 'path is required', got: %v", err)
	}
}

func TestValidatePath_CwdRelative(t *testing.T) {
	// A relative path inside cwd should be allowed
	abs, err := ValidatePath("main.go")
	if err != nil {
		t.Fatalf("ValidatePath('main.go') error: %v", err)
	}
	cwd, _ := os.Getwd()
	if !strings.HasPrefix(abs, cwd) {
		t.Errorf("expected path under cwd %s, got %s", cwd, abs)
	}
}

func TestValidatePath_CwdAbsolute(t *testing.T) {
	cwd, _ := os.Getwd()
	testPath := filepath.Join(cwd, "somefile.txt")
	abs, err := ValidatePath(testPath)
	if err != nil {
		t.Fatalf("ValidatePath(absolute cwd path) error: %v", err)
	}
	if abs != testPath {
		t.Errorf("expected %s, got %s", testPath, abs)
	}
}

func TestValidatePath_TmpAllowed(t *testing.T) {
	// /tmp/ is in allowedPrefixes
	abs, err := ValidatePath("/tmp/test-poly-file.txt")
	if err != nil {
		t.Fatalf("ValidatePath('/tmp/test-poly-file.txt') error: %v", err)
	}
	if abs != "/tmp/test-poly-file.txt" {
		t.Errorf("expected /tmp/test-poly-file.txt, got %s", abs)
	}
}

func TestValidatePath_DotPolyAllowed(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	polyPath := filepath.Join(home, ".poly", "config.json")
	abs, errV := ValidatePath(polyPath)
	if errV != nil {
		t.Fatalf("ValidatePath(~/.poly/config.json) error: %v", errV)
	}
	if abs != polyPath {
		t.Errorf("expected %s, got %s", polyPath, abs)
	}
}

func TestValidatePath_TraversalBlocked(t *testing.T) {
	// Trying to escape cwd with ../../../etc/passwd
	_, err := ValidatePath("../../../etc/passwd")
	if err == nil {
		t.Error("ValidatePath with traversal should be blocked")
	}
	if !strings.Contains(err.Error(), "Access denied") {
		t.Errorf("expected 'Access denied', got: %v", err)
	}
}

func TestValidatePath_AbsoluteOutsideBlocked(t *testing.T) {
	_, err := ValidatePath("/etc/shadow")
	if err == nil {
		t.Error("ValidatePath('/etc/shadow') should be blocked")
	}
	if !strings.Contains(err.Error(), "Access denied") {
		t.Errorf("expected 'Access denied', got: %v", err)
	}
}

func TestValidatePath_HomeDirectlyBlocked(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	// Home itself (not ~/.poly/) should be blocked
	secretPath := filepath.Join(home, ".ssh", "id_rsa")
	_, errV := ValidatePath(secretPath)
	if errV == nil {
		t.Error("ValidatePath(~/.ssh/id_rsa) should be blocked")
	}
}
