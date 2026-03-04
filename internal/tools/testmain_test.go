package tools

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// For tests, we must allow /tmp as many tests use t.TempDir()
	AddAllowedPrefix("/tmp")
	
	os.Exit(m.Run())
}
