package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"
)

const maxReconnectAttempts = 5

// Client communicates with a single MCP server
type Client struct {
	config ServerConfig
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner

	mu                sync.Mutex
	nextID            int
	tools             []MCPTool
	info              serverInfo
	alive             bool
	reconnectAttempts int
}

// NewClient creates a new MCP client for a server config
func NewClient(cfg ServerConfig) *Client {
	return &Client{
		config: cfg,
		nextID: 1,
	}
}

// Connect starts the MCP server process and performs handshake
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config.Type != "stdio" {
		return fmt.Errorf("unsupported transport: %s (only stdio supported)", c.config.Type)
	}

	// Start subprocess
	c.cmd = exec.Command(c.config.Command, c.config.Args...)

	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdoutPipe, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	c.stdout = bufio.NewScanner(stdoutPipe)
	c.stdout.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	// Initialize handshake
	result, err := c.callLocked("initialize", initializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    clientCapabilities{},
		ClientInfo: clientInfo{
			Name:    "poly",
			Version: "1.0.0",
		},
	})
	if err != nil {
		_ = c.cmd.Process.Kill()
		return fmt.Errorf("initialize: %w", err)
	}

	var initResult initializeResult
	if err := json.Unmarshal(result, &initResult); err != nil {
		_ = c.cmd.Process.Kill()
		return fmt.Errorf("parse init result: %w", err)
	}

	c.info = initResult.ServerInfo

	// Send initialized notification
	c.sendNotificationLocked("notifications/initialized", nil)

	// Discover tools
	if err := c.discoverToolsLocked(); err != nil {
		// Non-fatal - server might not have tools
		c.tools = nil
	}

	c.alive = true
	return nil
}

// Close shuts down the MCP server
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeLocked()
}

// closeLocked shuts down the MCP server (caller must hold c.mu)
func (c *Client) closeLocked() {
	c.alive = false
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}
	c.stdin = nil
	c.stdout = nil
	c.cmd = nil
}

// Reconnect closes the current connection and attempts to re-establish it
// with exponential backoff (1s, 2s, 4s, 8s, 16s). Returns nil on success.
func (c *Client) Reconnect() error {
	c.mu.Lock()
	c.closeLocked()
	c.mu.Unlock()

	for attempt := 0; attempt < maxReconnectAttempts; attempt++ {
		backoff := time.Second * time.Duration(1<<uint(attempt))
		if backoff > 16*time.Second {
			backoff = 16 * time.Second
		}

		log.Printf("[MCP] Reconnecting %s (attempt %d/%d, backoff %v)",
			c.config.Name, attempt+1, maxReconnectAttempts, backoff)
		time.Sleep(backoff)

		if err := c.Connect(); err != nil {
			log.Printf("[MCP] Reconnect %s failed: %v", c.config.Name, err)
			continue
		}

		c.mu.Lock()
		c.reconnectAttempts = 0
		c.mu.Unlock()
		log.Printf("[MCP] Reconnected %s successfully (%d tools)", c.config.Name, len(c.Tools()))
		return nil
	}

	c.mu.Lock()
	c.alive = false
	c.mu.Unlock()
	return fmt.Errorf("reconnect failed after %d attempts", maxReconnectAttempts)
}

// IsAlive returns true if the server is connected
func (c *Client) IsAlive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.alive
}

// Tools returns the tools provided by this server
func (c *Client) Tools() []MCPTool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tools
}

// ServerName returns the server's reported name
func (c *Client) ServerName() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.info.Name != "" {
		return c.info.Name
	}
	return c.config.Name
}

// CallTool executes a tool on this server.
// If the call fails due to a broken pipe, it attempts a single reconnect.
func (c *Client) CallTool(name string, args map[string]interface{}) (string, error) {
	result, err := c.callToolOnce(name, args)
	if err != nil && !c.IsAlive() {
		// Connection lost - attempt reconnect
		log.Printf("[MCP] CallTool %s/%s failed, attempting reconnect: %v", c.config.Name, name, err)
		if reconnErr := c.Reconnect(); reconnErr != nil {
			return "", fmt.Errorf("call failed and reconnect failed: %w", reconnErr)
		}
		// Retry the call after successful reconnect
		return c.callToolOnce(name, args)
	}
	return result, err
}

func (c *Client) callToolOnce(name string, args map[string]interface{}) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.alive {
		return "", fmt.Errorf("server not connected")
	}

	result, err := c.callLocked("tools/call", toolCallParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}

	var callResult toolCallResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return string(result), nil // Return raw if can't parse
	}

	// Combine all text content
	var text string
	for _, c := range callResult.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}

	if callResult.IsError {
		return "", fmt.Errorf("%s", text)
	}

	return text, nil
}

// --- Internal methods ---

func (c *Client) discoverToolsLocked() error {
	result, err := c.callLocked("tools/list", nil)
	if err != nil {
		return err
	}

	var listResult toolsListResult
	if err := json.Unmarshal(result, &listResult); err != nil {
		return err
	}

	c.tools = listResult.Tools
	return nil
}

func (c *Client) callLocked(method string, params interface{}) (json.RawMessage, error) {
	id := c.nextID
	c.nextID++

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Write request (newline-delimited JSON)
	if _, err := fmt.Fprintf(c.stdin, "%s\n", data); err != nil {
		c.alive = false
		return nil, fmt.Errorf("write: %w", err)
	}

	// Read response
	for c.stdout.Scan() {
		line := c.stdout.Text()
		if line == "" {
			continue
		}

		var resp jsonRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue // Skip non-JSON lines (stderr leaking?)
		}

		// Check if this is our response (matching ID)
		if resp.ID == id {
			if resp.Error != nil {
				return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
			}
			return resp.Result, nil
		}
		// Otherwise it's a notification or mismatched response, skip
	}

	c.alive = false
	return nil, fmt.Errorf("connection closed")
}

func (c *Client) sendNotificationLocked(method string, params interface{}) {
	// JSON-RPC 2.0: notifications MUST NOT have an "id" field
	type notification struct {
		JSONRPC string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params,omitempty"`
	}
	req := notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	data, _ := json.Marshal(req)
	fmt.Fprintf(c.stdin, "%s\n", data)
}
