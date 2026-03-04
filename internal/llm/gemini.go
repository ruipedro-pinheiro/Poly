package llm

import (
	"bufio"
	"bytes"
	"context"
	cryptoRand "crypto/rand"
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
	"github.com/pedromelo/poly/internal/security"
)

const (
	codeAssistEndpoint     = "https://cloudcode-pa.googleapis.com/v1internal"
	defaultCodeAssistModel = "gemini-2.5-pro"
	defaultAPIModel        = "gemini-2.5-flash"
)

var (
	codeAssistProjectID string
	codeAssistProjectMu sync.Mutex
	codeAssistResolved  bool
)

type GeminiProvider struct {
	config     ProviderConfig
	httpClient *http.Client
}

func NewGeminiProvider(cfg ProviderConfig) *GeminiProvider {
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel("gemini")
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = GetProviderMaxTokens("gemini")
	}
	return &GeminiProvider{config: cfg, httpClient: newStreamHTTPClient()}
}

func (p *GeminiProvider) Name() string           { return "gemini" }
func (p *GeminiProvider) DisplayName() string    { return "Gemini" }
func (p *GeminiProvider) Color() string          { return "#4285F4" }
func (p *GeminiProvider) ToolFormat() ToolFormat { return ToolFormatGoogle }
func (p *GeminiProvider) SetModel(model string)  { p.config.Model = model }
func (p *GeminiProvider) GetModel() string       { return p.config.Model }
func (p *GeminiProvider) SupportsTools() bool    { return true }
func (p *GeminiProvider) IsConfigured() bool     { return auth.GetStorage().IsConnected("gemini") }

func (p *GeminiProvider) Stream(ctx context.Context, messages []Message, toolDefs []ToolDefinition, opts ...StreamOptions) <-chan StreamEvent {
	eventChan := make(chan StreamEvent, 64)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				defer func() { recover() }() //nolint:errcheck // protect against closed channel
				eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("internal error (panic recovered)")}
			}
			close(eventChan)
		}()

		token, err := auth.GetStorage().GetAccessToken("gemini")
		if err != nil || token == "" {
			eventChan <- StreamEvent{Type: "error", Error: errors.New("no API key configured")}
			return
		}

		isAPIKey := strings.HasPrefix(token, "AIza")
		thinkingMode := GetThinkingMode(opts)
		role := GetRole(opts)

		// Resolve project ID for Code Assist if needed
		var projectID string
		if !isAPIKey {
			pid, err := p.resolveCodeAssistProjectID(token)
			if err != nil {
				eventChan <- StreamEvent{Type: "error", Error: fmt.Errorf("failed to get Code Assist project: %w", err)}
				return
			}
			projectID = pid
		}

		handler := func(ctx context.Context, history []Message, internalChan chan<- StreamEvent) (*SingleTurnResult, error) {
			gemHistory := p.buildGeminiHistory(history, role)
			gemTools := GemToolDefsFromPoly(toolDefs)

			model := p.config.Model
			if isAPIKey {
				if model == "" {
					model = defaultAPIModel
				}
				genConfig := &GemGenerationConfig{MaxOutputTokens: p.config.MaxTokens}
				if thinkingMode {
					genConfig.ThinkingConfig = &GemThinkingConfig{ThinkingBudget: 8192}
				}
				body := GemRequestBody{
					Contents:         gemHistory,
					GenerationConfig: genConfig,
					Tools:            gemTools,
				}
				res, err := p.streamRequestPublicAPI(ctx, body, model, token, internalChan)
				if err != nil {
					return nil, err
				}
				return &SingleTurnResult{
					Content:      res.content,
					Thinking:     res.thinking,
					ToolCalls:    p.toToolCalls(res.functionCalls),
					InputTokens:  res.inputTokens,
					OutputTokens: res.outputTokens,
				}, nil
			} else {
				if model == "" || model == GetDefaultModels()["gemini"] {
					model = defaultCodeAssistModel
				}
				innerReq := GemCodeAssistInnerRequest{
					Contents: gemHistory,
					Tools:    gemTools,
				}
				if thinkingMode {
					innerReq.GenerationConfig = &GemGenerationConfig{
						ThinkingConfig: &GemThinkingConfig{ThinkingBudget: 8192},
					}
				}
				body := GemCodeAssistBody{
					Model:        model,
					Project:      projectID,
					UserPromptID: generateUUID(),
					Request:      innerReq,
				}
				res, err := p.streamRequestCodeAssist(ctx, body, token, internalChan)
				if err != nil {
					return nil, err
				}
				return &SingleTurnResult{
					Content:      res.content,
					Thinking:     res.thinking,
					ToolCalls:    p.toToolCalls(res.functionCalls),
					InputTokens:  res.inputTokens,
					OutputTokens: res.outputTokens,
				}, nil
			}
		}

		RunAgenticLoop(ctx, "gemini", p.config.Model, messages, toolDefs, eventChan, handler)
	}()

	return eventChan
}

func (p *GeminiProvider) buildGeminiHistory(messages []Message, role string) []GemContent {
	systemPrompt := BuildSystemPrompt("gemini", role)
	contents := make([]GemContent, 0, len(messages)+2)
	contents = append(contents,
		GemContent{Role: "user", Parts: []GemPart{NewGemTextPart(systemPrompt)}},
		GemContent{Role: "model", Parts: []GemPart{NewGemTextPart("Understood. I'm Gemini, chatting through Poly.")}},
	)

	for _, msg := range messages {
		r := "user"
		if msg.Role == "assistant" {
			r = "model"
		}
		parts := BuildGemPartsFromMessage(msg)
		if len(parts) > 0 {
			contents = append(contents, GemContent{Role: r, Parts: parts})
		}
	}
	return contents
}

func (p *GeminiProvider) toToolCalls(fcs []geminiFunctionCall) []ToolCall {
	calls := make([]ToolCall, len(fcs))
	for i, fc := range fcs {
		calls[i] = ToolCall{
			ID:        fmt.Sprintf("call_%s_%d", fc.name, i),
			Name:      fc.name,
			Arguments: fc.args,
		}
	}
	return calls
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

func (p *GeminiProvider) streamRequestPublicAPI(ctx context.Context, body interface{}, model, apiKey string, eventChan chan<- StreamEvent) (*geminiStreamResult, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":streamGenerateContent?alt=sse&key=" + apiKey

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoWithRetry(ctx, p.httpClient, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
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

	return result, scanner.Err()
}

func (p *GeminiProvider) streamRequestCodeAssist(ctx context.Context, body interface{}, token string, eventChan chan<- StreamEvent) (*geminiStreamResult, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := codeAssistEndpoint + ":streamGenerateContent?alt=sse"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := DoWithRetry(ctx, p.httpClient, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseCodeAssistStreamWithTools(resp.Body, eventChan)
}

func (p *GeminiProvider) parseCodeAssistStreamWithTools(body io.Reader, eventChan chan<- StreamEvent) (*geminiStreamResult, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	result := &geminiStreamResult{}
	var sseBuffer []string
	var sentText, sentThinking string

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
			text, thinking, functionCalls := extractCodeAssistContent(payload)
			if thinking != "" {
				delta := thinking
				if strings.HasPrefix(thinking, sentThinking) {
					delta = thinking[len(sentThinking):]
				}
				sentThinking = thinking
				if delta != "" {
					result.thinking += delta
					eventChan <- StreamEvent{Type: "thinking", Thinking: delta}
				}
			}
			if text != "" {
				delta := text
				if strings.HasPrefix(text, sentText) {
					delta = text[len(sentText):]
				}
				sentText = text
				if delta != "" {
					result.content += delta
					eventChan <- StreamEvent{Type: "content", Content: delta}
				}
			}
			result.functionCalls = append(result.functionCalls, functionCalls...)
		}
	}
	return result, scanner.Err()
}

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

	var text, thinking strings.Builder
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
				functionCalls = append(functionCalls, geminiFunctionCall{name: name, args: args})
			}
		}
	}
	return text.String(), thinking.String(), functionCalls
}

func (p *GeminiProvider) resolveCodeAssistProjectID(token string) (string, error) {
	codeAssistProjectMu.Lock()
	defer codeAssistProjectMu.Unlock()
	if codeAssistResolved {
		return codeAssistProjectID, nil
	}
	envProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if envProject == "" {
		envProject = os.Getenv("GOOGLE_CLOUD_PROJECT_ID")
	}
	if envProject == "" {
		envProject = os.Getenv("GCLOUD_PROJECT")
	}
	body := GemLoadCodeAssistBody{
		CloudAICompanionProject: envProject,
		Metadata: GemLoadCodeAssistMetadata{
			IDEType:     "IDE_UNSPECIFIED",
			Platform:    "PLATFORM_UNSPECIFIED",
			PluginType:  "GEMINI",
			DuetProject: envProject,
		},
	}
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", codeAssistEndpoint+":loadCodeAssist", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("loadCodeAssist failed (%d): %s", resp.StatusCode, security.SanitizeResponseBody(bodyBytes))
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if project, ok := result["cloudaicompanionProject"].(string); ok && project != "" {
		codeAssistProjectID = project
		codeAssistResolved = true
		return codeAssistProjectID, nil
	}
	if envProject != "" {
		codeAssistProjectID = envProject
		codeAssistResolved = true
		return codeAssistProjectID, nil
	}
	return "", errors.New("Code Assist requires GOOGLE_CLOUD_PROJECT environment variable")
}

func generateUUID() string {
	b := make([]byte, 16)
	if _, err := cryptoRand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
