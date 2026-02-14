package llm

import (
	"bufio"
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pedromelo/poly/internal/auth"
	"github.com/pedromelo/poly/internal/tools"
)

const (
	codeAssistEndpoint     = "https://cloudcode-pa.googleapis.com/v1internal"
	defaultCodeAssistModel = "gemini-2.5-pro"
	defaultAPIModel        = "gemini-2.5-flash"
)

var (
	codeAssistProjectID   string
	codeAssistProjectOnce sync.Once
	codeAssistProjectErr  error
)

func init() {
	RegisterProvider(NewGeminiProvider(ProviderConfig{}))
}

type GeminiProvider struct {
	config ProviderConfig
}

func NewGeminiProvider(cfg ProviderConfig) *GeminiProvider {
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel("gemini")
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = GetProviderMaxTokens("gemini")
	}
	return &GeminiProvider{config: cfg}
}

func (p *GeminiProvider) Name() string        { return "gemini" }
func (p *GeminiProvider) DisplayName() string { return "Gemini" }
func (p *GeminiProvider) Color() string       { return "#4285F4" }
func (p *GeminiProvider) ToolFormat() ToolFormat { return ToolFormatGoogle }
func (p *GeminiProvider) SetModel(model string)  { p.config.Model = model }
func (p *GeminiProvider) GetModel() string       { return p.config.Model }
func (p *GeminiProvider) SupportsTools() bool { return true }
func (p *GeminiProvider) IsConfigured() bool     { return auth.GetStorage().IsConnected("gemini") }

func (p *GeminiProvider) Stream(ctx context.Context, messages []Message, toolDefs []ToolDefinition, opts ...StreamOptions) <-chan StreamEvent {
	eventChan := make(chan StreamEvent, 64)

	go func() {
		defer close(eventChan)

		token, err := auth.GetStorage().GetAccessToken("gemini")
		if err != nil || token == "" {
			eventChan <- StreamEvent{Type: "error", Error: errors.New("no API key configured")}
			return
		}

		isAPIKey := strings.HasPrefix(token, "AIza")

		role := GetRole(opts)
		thinkingMode := GetThinkingMode(opts)
		if isAPIKey {
			p.agenticLoopPublicAPI(ctx, messages, toolDefs, token, eventChan, role, thinkingMode)
		} else {
			p.streamCodeAssist(ctx, messages, toolDefs, token, eventChan, role, thinkingMode)
		}
	}()

	return eventChan
}

// agenticLoopPublicAPI runs the tool-use loop for public Gemini API
func (p *GeminiProvider) agenticLoopPublicAPI(ctx context.Context, initialMessages []Message, toolDefs []ToolDefinition, apiKey string, eventChan chan<- StreamEvent, role string, thinkingMode bool) {
	// Build conversation contents
	contents := make([]map[string]interface{}, 0, len(initialMessages)+2)

	// System context as user/model pair with dynamic prompt
	systemPrompt := BuildSystemPrompt("gemini", role)
	contents = append(contents,
		map[string]interface{}{
			"role":  "user",
			"parts": []map[string]interface{}{{"text": systemPrompt}},
		},
		map[string]interface{}{
			"role":  "model",
			"parts": []map[string]interface{}{{"text": "Understood. I'm Gemini, chatting through Poly."}},
		},
	)

	// Add initial messages with image support
	for _, msg := range initialMessages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]interface{}{
			"role":  role,
			"parts": buildGeminiParts(msg),
		})
	}

	// Build tools (Google format)
	var googleTools []map[string]interface{}
	if len(toolDefs) > 0 {
		functionDeclarations := make([]map[string]interface{}, len(toolDefs))
		for i, tool := range toolDefs {
			functionDeclarations[i] = map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			}
		}
		googleTools = []map[string]interface{}{
			{"functionDeclarations": functionDeclarations},
		}
	}

	model := p.config.Model
	if model == "" {
		model = defaultAPIModel
	}

	var fullContent strings.Builder

	// Agentic loop
	for turn := 0; turn < GetMaxToolTurns(); turn++ {
		genConfig := map[string]interface{}{
			"maxOutputTokens": p.config.MaxTokens,
		}
		if thinkingMode {
			genConfig["thinkingConfig"] = map[string]interface{}{
				"thinkingBudget": 8192,
			}
		}
		body := map[string]interface{}{
			"contents":         contents,
			"generationConfig": genConfig,
		}

		if len(googleTools) > 0 {
			body["tools"] = googleTools
		}

		result, err := p.streamRequestPublicAPI(ctx, body, model, apiKey, eventChan)
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: err}
			return
		}

		fullContent.WriteString(result.content)

		// No function calls? Done
		if len(result.functionCalls) == 0 {
			eventChan <- StreamEvent{
				Type: "done",
				Response: &Response{
					Content:      fullContent.String(),
					Provider:     "gemini",
					Model:        model,
					InputTokens:  result.inputTokens,
					OutputTokens: result.outputTokens,
				},
			}
			return
		}

		// Build model response with function calls
		modelParts := make([]map[string]interface{}, 0)
		if result.content != "" {
			modelParts = append(modelParts, map[string]interface{}{"text": result.content})
		}
		for _, fc := range result.functionCalls {
			modelParts = append(modelParts, map[string]interface{}{
				"functionCall": map[string]interface{}{
					"name": fc.name,
					"args": fc.args,
				},
			})
		}
		contents = append(contents, map[string]interface{}{
			"role":  "model",
			"parts": modelParts,
		})

		// Execute functions and build response
		functionResponses := make([]map[string]interface{}, 0, len(result.functionCalls))
		for _, fc := range result.functionCalls {
			// Emit tool_use event before execution
			eventChan <- StreamEvent{
				Type: "tool_use",
				ToolCall: &ToolCall{
					ID:        fc.name, // Gemini doesn't have IDs, use name
					Name:      fc.name,
					Arguments: fc.args,
				},
			}

			toolResult := tools.Execute(fc.name, fc.args)

			// Emit tool_result event after execution
			eventChan <- StreamEvent{
				Type: "tool_result",
				ToolCall: &ToolCall{
					ID:        fc.name,
					Name:      fc.name,
					Arguments: fc.args,
				},
				ToolResult: &ToolResult{
					ToolUseID: fc.name,
					Content:   toolResult.Content,
					IsError:   toolResult.IsError,
				},
			}

			functionResponses = append(functionResponses, map[string]interface{}{
				"functionResponse": map[string]interface{}{
					"name": fc.name,
					"response": map[string]interface{}{
						"content": toolResult.Content,
					},
				},
			})
		}

		// Add function responses as user message
		contents = append(contents, map[string]interface{}{
			"role":  "user",
			"parts": functionResponses,
		})
	}

	eventChan <- StreamEvent{
		Type:    "content",
		Content: "\n⚠️ Max tool turns reached\n",
	}
	eventChan <- StreamEvent{
		Type: "done",
		Response: &Response{
			Content:  fullContent.String(),
			Provider: "gemini",
			Model:    model,
		},
	}
}

// buildGeminiParts creates message parts with optional images (Google format)
func buildGeminiParts(msg Message) []map[string]interface{} {
	if len(msg.Images) == 0 {
		return []map[string]interface{}{{"text": msg.Content}}
	}

	parts := make([]map[string]interface{}, 0, len(msg.Images)+1)

	for _, img := range msg.Images {
		parts = append(parts, map[string]interface{}{
			"inlineData": map[string]interface{}{
				"mimeType": img.MediaType,
				"data":     base64.StdEncoding.EncodeToString(img.Data),
			},
		})
	}

	if msg.Content != "" {
		parts = append(parts, map[string]interface{}{"text": msg.Content})
	}

	return parts
}

type geminiStreamResult struct {
	content       string
	thinking      string
	functionCalls []geminiFunctionCall
	inputTokens   int
	outputTokens  int
}

type geminiFunctionCall struct {
	name string
	args map[string]interface{}
}

func (p *GeminiProvider) streamRequestPublicAPI(ctx context.Context, body map[string]interface{}, model, apiKey string, eventChan chan<- StreamEvent) (*geminiStreamResult, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":streamGenerateContent?alt=sse&key=" + apiKey

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(RetryDelay(attempt - 1)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Minute}
		resp, err = client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusOK {
			break
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))

		if !ShouldRetry(resp.StatusCode) {
			return nil, lastErr
		}
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	result := &geminiStreamResult{}

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text         string `json:"text"`
						Thought      bool   `json:"thought,omitempty"`
						FunctionCall *struct {
							Name string                 `json:"name"`
							Args map[string]interface{} `json:"args"`
						} `json:"functionCall"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
			UsageMetadata *struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
			} `json:"usageMetadata"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.UsageMetadata != nil {
			result.inputTokens = event.UsageMetadata.PromptTokenCount
			result.outputTokens = event.UsageMetadata.CandidatesTokenCount
		}

		if len(event.Candidates) > 0 && len(event.Candidates[0].Content.Parts) > 0 {
			for _, part := range event.Candidates[0].Content.Parts {
				if part.Thought && part.Text != "" {
					result.thinking += part.Text
					eventChan <- StreamEvent{Type: "thinking", Thinking: part.Text}
				} else if part.Text != "" {
					result.content += part.Text
					eventChan <- StreamEvent{Type: "content", Content: part.Text}
				}
				if part.FunctionCall != nil {
					result.functionCalls = append(result.functionCalls, geminiFunctionCall{
						name: part.FunctionCall.Name,
						args: part.FunctionCall.Args,
					})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// streamCodeAssist handles streaming via Google Cloud Code Assist (for OAuth tokens)
// Now with full agentic loop for tool execution
func (p *GeminiProvider) streamCodeAssist(ctx context.Context, messages []Message, toolDefs []ToolDefinition, token string, eventChan chan<- StreamEvent, role string, thinkingMode bool) {
	projectID, err := p.resolveCodeAssistProjectID(token)
	if err != nil {
		eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("failed to get Code Assist project: %w", err)}
		return
	}

	systemPrompt := BuildSystemPrompt("gemini", role)
	contents := make([]map[string]interface{}, 0, len(messages)+2)
	contents = append(contents,
		map[string]interface{}{"role": "user", "parts": []map[string]interface{}{{"text": systemPrompt}}},
		map[string]interface{}{"role": "model", "parts": []map[string]interface{}{{"text": "Understood. I'm Gemini, chatting through Poly."}}},
	)

	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]interface{}{
			"role":  role,
			"parts": buildGeminiParts(msg),
		})
	}

	model := p.config.Model
	if model == "" || model == DefaultModels["gemini"] {
		model = defaultCodeAssistModel
	}

	// Build tools (Google format)
	var googleTools []map[string]interface{}
	if len(toolDefs) > 0 {
		functionDeclarations := make([]map[string]interface{}, len(toolDefs))
		for i, tool := range toolDefs {
			functionDeclarations[i] = map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			}
		}
		googleTools = []map[string]interface{}{
			{"functionDeclarations": functionDeclarations},
		}
	}

	var fullContent strings.Builder
	sessionID := generateUUID()

	// Agentic loop
	for turn := 0; turn < GetMaxToolTurns(); turn++ {
		request := map[string]interface{}{
			"contents":   contents,
			"session_id": sessionID,
		}
		if thinkingMode {
			request["generationConfig"] = map[string]interface{}{
				"thinkingConfig": map[string]interface{}{
					"thinkingBudget": 8192,
				},
			}
		}
		if len(googleTools) > 0 {
			request["tools"] = googleTools
		}
		body := map[string]interface{}{
			"model":          model,
			"project":        projectID,
			"user_prompt_id": generateUUID(),
			"request":        request,
		}

		result, err := p.streamRequestCodeAssist(ctx, body, token, eventChan)
		if err != nil {
			eventChan <- StreamEvent{Type: "error", Error: err}
			return
		}

		fullContent.WriteString(result.content)

		// No function calls? Done
		if len(result.functionCalls) == 0 {
			eventChan <- StreamEvent{
				Type: "done",
				Response: &Response{
					Content:      fullContent.String(),
					Provider:     "gemini",
					Model:        model,
					InputTokens:  result.inputTokens,
					OutputTokens: result.outputTokens,
				},
			}
			return
		}

		// Build model response with function calls
		modelParts := make([]map[string]interface{}, 0)
		if result.content != "" {
			modelParts = append(modelParts, map[string]interface{}{"text": result.content})
		}
		for _, fc := range result.functionCalls {
			modelParts = append(modelParts, map[string]interface{}{
				"functionCall": map[string]interface{}{
					"name": fc.name,
					"args": fc.args,
				},
			})
		}
		contents = append(contents, map[string]interface{}{
			"role":  "model",
			"parts": modelParts,
		})

		// Execute functions and build response
		functionResponses := make([]map[string]interface{}, 0, len(result.functionCalls))
		for _, fc := range result.functionCalls {
			// Emit tool_use event before execution
			eventChan <- StreamEvent{
				Type: "tool_use",
				ToolCall: &ToolCall{
					ID:        fc.name,
					Name:      fc.name,
					Arguments: fc.args,
				},
			}

			toolResult := tools.Execute(fc.name, fc.args)

			// Emit tool_result event after execution
			eventChan <- StreamEvent{
				Type: "tool_result",
				ToolCall: &ToolCall{
					ID:        fc.name,
					Name:      fc.name,
					Arguments: fc.args,
				},
				ToolResult: &ToolResult{
					ToolUseID: fc.name,
					Content:   toolResult.Content,
					IsError:   toolResult.IsError,
				},
			}

			functionResponses = append(functionResponses, map[string]interface{}{
				"functionResponse": map[string]interface{}{
					"name": fc.name,
					"response": map[string]interface{}{
						"content": toolResult.Content,
					},
				},
			})
		}

		// Add function responses as user message
		contents = append(contents, map[string]interface{}{
			"role":  "user",
			"parts": functionResponses,
		})
	}

	eventChan <- StreamEvent{
		Type:    "content",
		Content: "\n⚠️ Max tool turns reached\n",
	}
	eventChan <- StreamEvent{
		Type: "done",
		Response: &Response{
			Content:  fullContent.String(),
			Provider: "gemini",
			Model:    model,
		},
	}
}

// streamRequestCodeAssist makes a single request to Code Assist and parses the response
func (p *GeminiProvider) streamRequestCodeAssist(ctx context.Context, body map[string]interface{}, token string, eventChan chan<- StreamEvent) (*geminiStreamResult, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := codeAssistEndpoint + ":streamGenerateContent?alt=sse"

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(RetryDelay(attempt - 1)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{Timeout: 5 * time.Minute}
		resp, err = client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusOK {
			break
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("Code Assist error (%d): %s", resp.StatusCode, string(bodyBytes))

		if !ShouldRetry(resp.StatusCode) {
			return nil, lastErr
		}
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
	}
	defer resp.Body.Close()

	return p.parseCodeAssistStreamWithTools(resp.Body, eventChan)
}

// parseCodeAssistStreamWithTools parses Code Assist response including function calls
func (p *GeminiProvider) parseCodeAssistStreamWithTools(body io.Reader, eventChan chan<- StreamEvent) (*geminiStreamResult, error) {
	scanner := bufio.NewScanner(body)
	result := &geminiStreamResult{}
	var sseBuffer []string
	var sentText string
	var sentThinking string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data: ") {
			sseBuffer = append(sseBuffer, strings.TrimPrefix(line, "data: "))
			continue
		}

		if strings.TrimSpace(line) == "" && len(sseBuffer) > 0 {
			data := strings.Join(sseBuffer, "\n")
			sseBuffer = nil

			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(data), &payload); err != nil {
				continue
			}

			// Extract text, thinking, and function calls
			text, thinking, functionCalls := extractCodeAssistContent(payload)

			// Handle thinking delta
			if thinking != "" {
				var delta string
				if strings.HasPrefix(thinking, sentThinking) {
					delta = thinking[len(sentThinking):]
				} else {
					delta = thinking
				}
				sentThinking = thinking

				if delta != "" {
					result.thinking += delta
					eventChan <- StreamEvent{Type: "thinking", Thinking: delta}
				}
			}

			// Handle text delta
			if text != "" {
				var delta string
				if strings.HasPrefix(text, sentText) {
					delta = text[len(sentText):]
				} else {
					delta = text
				}
				sentText = text

				if delta != "" {
					result.content += delta
					eventChan <- StreamEvent{Type: "content", Content: delta}
				}
			}

			// Handle function calls
			result.functionCalls = append(result.functionCalls, functionCalls...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// extractCodeAssistContent extracts text and function calls from Code Assist response
func extractCodeAssistContent(payload map[string]interface{}) (string, string, []geminiFunctionCall) {
	response, ok := payload["response"].(map[string]interface{})
	if !ok {
		return "", "", nil
	}

	candidates, ok := response["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", "", nil
	}

	candidate, ok := candidates[0].(map[string]interface{})
	if !ok {
		return "", "", nil
	}

	content, ok := candidate["content"].(map[string]interface{})
	if !ok {
		return "", "", nil
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "", "", nil
	}

	var text strings.Builder
	var thinking strings.Builder
	var functionCalls []geminiFunctionCall

	for _, part := range parts {
		p, ok := part.(map[string]interface{})
		if !ok {
			continue
		}

		isThought, _ := p["thought"].(bool)
		if t, ok := p["text"].(string); ok {
			if isThought {
				thinking.WriteString(t)
			} else {
				text.WriteString(t)
			}
		}

		if fc, ok := p["functionCall"].(map[string]interface{}); ok {
			name, _ := fc["name"].(string)
			args, _ := fc["args"].(map[string]interface{})
			if name != "" {
				functionCalls = append(functionCalls, geminiFunctionCall{
					name: name,
					args: args,
				})
			}
		}
	}

	return text.String(), thinking.String(), functionCalls
}

func (p *GeminiProvider) resolveCodeAssistProjectID(token string) (string, error) {
	codeAssistProjectOnce.Do(func() {
		envProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
		if envProject == "" {
			envProject = os.Getenv("GOOGLE_CLOUD_PROJECT_ID")
		}
		if envProject == "" {
			envProject = os.Getenv("GCLOUD_PROJECT")
		}

		body := map[string]interface{}{
			"cloudaicompanionProject": envProject,
			"metadata": map[string]interface{}{
				"ideType":     "IDE_UNSPECIFIED",
				"platform":    "PLATFORM_UNSPECIFIED",
				"pluginType":  "GEMINI",
				"duetProject": envProject,
			},
		}

		jsonBody, err := json.Marshal(body)
		if err != nil {
			codeAssistProjectErr = fmt.Errorf("failed to marshal request: %w", err)
			return
		}
		req, err := http.NewRequest("POST", codeAssistEndpoint+":loadCodeAssist", bytes.NewReader(jsonBody))
		if err != nil {
			codeAssistProjectErr = fmt.Errorf("failed to create request: %w", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			codeAssistProjectErr = err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			codeAssistProjectErr = fmt.Errorf("loadCodeAssist failed (%d): %s", resp.StatusCode, string(bodyBytes))
			return
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			codeAssistProjectErr = fmt.Errorf("failed to decode response: %w", err)
			return
		}

		if project, ok := result["cloudaicompanionProject"].(string); ok && project != "" {
			codeAssistProjectID = project
			return
		}

		if envProject != "" {
			codeAssistProjectID = envProject
			return
		}

		codeAssistProjectErr = errors.New("Code Assist requires GOOGLE_CLOUD_PROJECT environment variable")
	})

	if codeAssistProjectErr != nil {
		return "", codeAssistProjectErr
	}
	return codeAssistProjectID, nil
}

func generateUUID() string {
	b := make([]byte, 16)
	if _, err := cryptoRand.Read(b); err != nil {
		// Fallback to timestamp if crypto/rand fails (should never happen)
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
