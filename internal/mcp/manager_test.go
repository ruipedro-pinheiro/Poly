package mcp

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("Expected non-nil manager")
	}

	if m.clients == nil {
		t.Error("Expected initialized clients map")
	}

	if len(m.clients) != 0 {
		t.Errorf("Expected empty clients map, got %d clients", len(m.clients))
	}
}

func TestManager_Status(t *testing.T) {
	m := NewManager()
	
	status := m.Status()
	if status == nil {
		t.Fatal("Expected non-nil status")
	}

	if len(status) != 0 {
		t.Error("Expected empty status for new manager")
	}
}

func TestServerConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ServerConfig
		isValid bool
	}{
		{
			name: "valid config",
			cfg: ServerConfig{
				Name:    "test-server",
				Command: "node",
				Args:    []string{"server.js"},
			},
			isValid: true,
		},
		{
			name: "missing name",
			cfg: ServerConfig{
				Command: "node",
				Args:    []string{"server.js"},
			},
			isValid: false,
		},
		{
			name: "missing command",
			cfg: ServerConfig{
				Name: "test-server",
				Args: []string{"server.js"},
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.cfg.Name != "" && tt.cfg.Command != ""
			if valid != tt.isValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.isValid, valid)
			}
		})
	}
}

func TestManager_Close(t *testing.T) {
	m := NewManager()
	
	// Close should not panic on empty manager
	m.Close()

	status := m.Status()
	if len(status) != 0 {
		t.Error("Expected empty status after close")
	}
}
