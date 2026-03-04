package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// WebFetchTool fetches content from a URL
type WebFetchTool struct{}

func (t *WebFetchTool) Name() string { return "web_fetch" }

func (t *WebFetchTool) Description() string {
	return "Fetches content from a URL. Returns the text content (HTML tags stripped). Max 30KB. Blocks localhost/private IPs for security."
}

func (t *WebFetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "The URL to fetch (must be https://)",
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Execute(args map[string]interface{}) ToolResult {
	rawURL, _ := args["url"].(string)
	if rawURL == "" {
		return ToolResult{Content: "Error: url is required", IsError: true}
	}

	// Parse and validate URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: invalid URL: %v", err), IsError: true}
	}

	// Block non-HTTP(S) schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ToolResult{Content: "Error: only http/https URLs allowed", IsError: true}
	}

	// Block private/localhost IPs
	if isPrivateHost(parsed.Hostname()) {
		return ToolResult{Content: "Error: cannot fetch from private/localhost addresses", IsError: true}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}
	req.Header.Set("User-Agent", "Poly/1.0 (AI Terminal Tool)")

	resp, err := client.Do(req)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ToolResult{Content: fmt.Sprintf("Error: HTTP %d %s", resp.StatusCode, resp.Status), IsError: true}
	}

	// Read up to 30KB
	limited := io.LimitReader(resp.Body, 30*1024)
	body, err := io.ReadAll(limited)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error reading body: %v", err), IsError: true}
	}

	content := string(body)

	// Strip HTML tags for readability
	content = stripHTML(content)

	// Truncate if needed
	if len(content) > 30000 {
		content = content[:30000] + "\n... (truncated)"
	}

	return ToolResult{Content: fmt.Sprintf("Fetched %s (%d bytes):\n\n%s", rawURL, len(content), content)}
}

// WebSearchTool searches the web using DuckDuckGo
type WebSearchTool struct{}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Description() string {
	return "Searches the web using DuckDuckGo. Returns titles, URLs, and snippets for the query."
}

func (t *WebSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query",
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(args map[string]interface{}) ToolResult {
	query, _ := args["query"].(string)
	if query == "" {
		return ToolResult{Content: "Error: query is required", IsError: true}
	}

	// Use DuckDuckGo Instant Answers API (free, no key)
	searchURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1",
		url.QueryEscape(query))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error creating request: %v", err), IsError: true}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error reading response: %v", err), IsError: true}
	}

	// Parse the JSON response manually (avoid importing encoding/json for simple extraction)
	content := string(body)
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))

	// Extract Abstract
	if abstract := extractJSONField(content, "Abstract"); abstract != "" {
		result.WriteString("Summary: " + abstract + "\n")
		if abstractURL := extractJSONField(content, "AbstractURL"); abstractURL != "" {
			result.WriteString("Source: " + abstractURL + "\n")
		}
		result.WriteString("\n")
	}

	// Extract RelatedTopics (basic parsing)
	if strings.Contains(content, "RelatedTopics") {
		result.WriteString("Related:\n")
		// Simple text extraction from RelatedTopics
		parts := strings.Split(content, "\"Text\":\"")
		count := 0
		for i := 1; i < len(parts) && count < 8; i++ {
			endIdx := strings.Index(parts[i], "\"")
			if endIdx > 0 {
				text := parts[i][:endIdx]
				if text != "" && len(text) > 5 {
					result.WriteString("  - " + text + "\n")
					count++
				}
			}
		}
	}

	if result.Len() < 50 {
		result.WriteString("No instant answer available. Try web_fetch on a specific URL.")
	}

	return ToolResult{Content: result.String()}
}

// isPrivateHost checks if a hostname resolves to a private/localhost IP
func isPrivateHost(host string) bool {
	// Check common private hostnames
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "127.0.0.1" || lower == "::1" || lower == "0.0.0.0" {
		return true
	}

	// Resolve and check
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return true
		}
	}

	return false
}

// stripHTML removes HTML tags from content
func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(s, " ")
	// Collapse whitespace
	spaceRe := regexp.MustCompile(`\s+`)
	cleaned = spaceRe.ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}

// extractJSONField extracts a simple string field from JSON
func extractJSONField(json, field string) string {
	key := fmt.Sprintf(`"%s":"`, field)
	idx := strings.Index(json, key)
	if idx < 0 {
		return ""
	}
	start := idx + len(key)
	end := strings.Index(json[start:], `"`)
	if end < 0 {
		return ""
	}
	return json[start : start+end]
}
