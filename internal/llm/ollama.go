package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type OllamaProvider struct {
	config     ProviderConfig
	httpClient *http.Client
}

func NewOllamaProvider(cfg ProviderConfig) *OllamaProvider {
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel("ollama")
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = GetProviderMaxTokens("ollama")
	}
	return &OllamaProvider{config: cfg, httpClient: newStreamHTTPClient()}
}

func (p *OllamaProvider) Name() string           { return "ollama" }
func (p *OllamaProvider) DisplayName() string    { return "Ollama" }
func (p *OllamaProvider) Color() string          { return "#FFFFFF" }        // White for Ollama
func (p *OllamaProvider) ToolFormat() ToolFormat { return ToolFormatOpenAI } // Ollama uses OpenAI tool format
func (p *OllamaProvider) SetModel(model string)  { p.config.Model = model }
func (p *OllamaProvider) GetModel() string       { return p.config.Model }
func (p *OllamaProvider) SupportsTools() bool    { return false } // For now
func (p *OllamaProvider) IsConfigured() bool     { return true }  // Always configured for local Ollama

func (p *OllamaProvider) Stream(ctx context.Context, messages []Message, toolDefs []ToolDefinition, opts ...StreamOptions) <-chan StreamEvent {
	eventChan := make(chan StreamEvent, 64)

	go func() {
		defer close(eventChan)

		if len(messages) == 0 {
			eventChan <- StreamEvent{Type: "error", Error: errors.New("no messages to stream")}
			return
		}

		// Convert Poly messages to Ollama messages (OpenAI-compatible format)
		ollamaMessages := make([]OAIMessage, 0, len(messages))
		for _, msg := range messages {
			if msg.Role == "system" {
				continue
			}
			ollamaMessages = append(ollamaMessages, NewOAITextMessage(msg.Role, msg.Content))
		}

		body := OAIRequestBody{
			Model:    p.GetModel(),
			Stream:   true,
			Messages: ollamaMessages,
		}

		jsonBody, err := json.Marshal(body)
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: err}
			return
		}

		url := GetProviderEndpoint("ollama")
		if url == "" {
			url = "http://localhost:11434/api"
		}
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "http://" + url
		}
		if !strings.HasSuffix(url, "/api") {
			url = strings.TrimSuffix(url, "/") + "/api"
		}
		url += "/chat"

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("failed to create ollama request: %w", err)}
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("failed to connect to ollama: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("ollama returned status %d", resp.StatusCode)}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
		var fullContent strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			var ollamaResp struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Done bool `json:"done"`
			}
			if err := json.Unmarshal([]byte(line), &ollamaResp); err != nil {
				continue
			}

			if !ollamaResp.Done {
				eventChan <- StreamEvent{Type: "content", Content: ollamaResp.Message.Content}
				fullContent.WriteString(ollamaResp.Message.Content)
			} else {
				eventChan <- StreamEvent{
					Type: "done",
					Response: &Response{
						Content:  fullContent.String(),
						Provider: p.Name(),
						Model:    p.GetModel(),
					},
				}
				break
			}
		}

		if err := scanner.Err(); err != nil {
			eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("error reading ollama stream: %w", err)}
		}
	}()

	return eventChan
}
