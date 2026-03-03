package llm

import (
	"os"
	"testing"
)

// TestMain registers all built-in providers before running tests.
// Required since providers are no longer registered via init().
func TestMain(m *testing.M) {
	RegisterAllProviders()
	os.Exit(m.Run())
}
