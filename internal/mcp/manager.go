package mcp

import (
	"fmt"
	"log"
	"sync"

	"github.com/pedromelo/poly/internal/config"
	"github.com/pedromelo/poly/internal/tools"
)

// Global is the package-level MCP manager instance
var Global *Manager

// Manager manages multiple MCP server connections
type Manager struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

// NewManager creates a new MCP manager
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
	}
}

// Init creates the global manager and connects all configured MCP servers.
// Errors are logged but non-fatal (MCP is optional).
func Init() {
	Global = NewManager()

	servers := config.GetMCPServers()
	if len(servers) == 0 {
		return
	}

	var configs []ServerConfig
	for name, srv := range servers {
		t := srv.Type
		if t == "" {
			t = "stdio"
		}
		configs = append(configs, ServerConfig{
			Name:    name,
			Type:    t,
			Command: srv.Command,
			Args:    srv.Args,
			URL:     srv.URL,
		})
	}

	for _, cfg := range configs {
		if err := Global.Connect(cfg); err != nil {
			log.Printf("[MCP] Failed to connect %s: %v", cfg.Name, err)
		} else {
			toolCount := 0
			if c, ok := Global.GetClient(cfg.Name); ok {
				toolCount = len(c.Tools())
			}
			log.Printf("[MCP] Connected: %s (%d tools)", cfg.Name, toolCount)
		}
	}
}

// ConnectAll connects to all configured MCP servers
func (m *Manager) ConnectAll(configs []ServerConfig) {
	for _, cfg := range configs {
		m.Connect(cfg)
	}
}

// Connect starts a single MCP server and registers its tools
func (m *Manager) Connect(cfg ServerConfig) error {
	client := NewClient(cfg)

	if err := client.Connect(); err != nil {
		m.mu.Lock()
		m.clients[cfg.Name] = client // Store even if failed (for status reporting)
		m.mu.Unlock()
		return fmt.Errorf("MCP %s: %w", cfg.Name, err)
	}

	m.mu.Lock()
	m.clients[cfg.Name] = client
	m.mu.Unlock()

	// Register tools from this server
	m.registerTools(cfg.Name, client)

	return nil
}

// registerTools registers MCP tools as Poly tools with namespaced names
func (m *Manager) registerTools(serverName string, client *Client) {
	for _, mcpTool := range client.Tools() {
		// Create a namespaced tool that bridges to the MCP server
		tool := &mcpToolBridge{
			serverName: serverName,
			mcpTool:    mcpTool,
			client:     client,
		}
		tools.Register(tool)
	}
}

// Close shuts down all MCP servers
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, client := range m.clients {
		client.Close()
	}
	m.clients = make(map[string]*Client)
}

// Status returns the status of all MCP servers
func (m *Manager) Status() []ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]ServerStatus, 0, len(m.clients))
	for name, client := range m.clients {
		status := ServerStatus{
			Name:      name,
			Connected: client.IsAlive(),
			ToolCount: len(client.Tools()),
		}
		statuses = append(statuses, status)
	}
	return statuses
}

// GetClient returns a specific MCP client
func (m *Manager) GetClient(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.clients[name]
	return c, ok
}

// --- MCP Tool Bridge ---
// Wraps an MCP tool as a Poly tool

type mcpToolBridge struct {
	serverName string
	mcpTool    MCPTool
	client     *Client
}

func (t *mcpToolBridge) Name() string {
	return "mcp_" + t.serverName + "_" + t.mcpTool.Name
}

func (t *mcpToolBridge) Description() string {
	desc := t.mcpTool.Description
	if desc == "" {
		desc = "MCP tool from " + t.serverName
	}
	return fmt.Sprintf("[MCP:%s] %s", t.serverName, desc)
}

func (t *mcpToolBridge) Parameters() map[string]interface{} {
	if t.mcpTool.InputSchema != nil {
		return t.mcpTool.InputSchema
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *mcpToolBridge) Execute(args map[string]interface{}) tools.ToolResult {
	result, err := t.client.CallTool(t.mcpTool.Name, args)
	if err != nil {
		return tools.ToolResult{
			Content: fmt.Sprintf("MCP error (%s/%s): %v", t.serverName, t.mcpTool.Name, err),
			IsError: true,
		}
	}
	return tools.ToolResult{Content: result}
}
