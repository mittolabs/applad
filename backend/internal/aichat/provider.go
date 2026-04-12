package aichat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Provider is the common interface every AI backend must implement.
type Provider interface {
	// Name returns a short human-readable label, e.g. "Claude Sonnet 4.5".
	Name() string
	// Chat sends one turn (with optional prior tool results) and returns the
	// AI's response: either a text delta stream or tool calls to execute.
	// When tool calls are present they are sent to out as a sentinel JSON blob
	// and the caller must handle the tool-call / result loop.
	StreamChat(ctx context.Context, system string, messages []ProviderMessage, tools []interface{}, out chan<- StreamEvent) error
}

// StreamEvent is emitted on the out channel during streaming.
type StreamEvent struct {
	Delta     string     // text token
	ToolCalls []ToolCall // non-empty when the model wants to call tools
	Done      bool
}

// ProviderMessage is a turn in the conversation (may include tool results).
type ProviderMessage struct {
	Role       string      // user | assistant | tool
	Content    string      // text content
	ToolCalls  []ToolCall  // populated for assistant messages that called tools
	ToolResult *ToolResult // populated for tool result messages
}

// ── Anthropic ─────────────────────────────────────────────────────────────────

type anthropicProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func newAnthropic(apiKey, model, baseURL string, client *http.Client) Provider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &anthropicProvider{apiKey: apiKey, model: model, baseURL: baseURL, client: client}
}

func (p *anthropicProvider) Name() string { return humanModelName(p.model, "Claude") }

func (p *anthropicProvider) StreamChat(ctx context.Context, system string, msgs []ProviderMessage, tools []interface{}, out chan<- StreamEvent) error {
	type content struct {
		Type       string      `json:"type"`
		Text       string      `json:"text,omitempty"`
		ID         string      `json:"id,omitempty"`
		Name       string      `json:"name,omitempty"`
		Input      interface{} `json:"input,omitempty"`
		ToolUseID  string      `json:"tool_use_id,omitempty"`
		Content    interface{} `json:"content,omitempty"`
	}
	type msg struct {
		Role    string    `json:"role"`
		Content interface{} `json:"content"`
	}

	var built []msg
	for _, m := range msgs {
		switch m.Role {
		case "user":
			built = append(built, msg{Role: "user", Content: m.Content})
		case "assistant":
			if len(m.ToolCalls) > 0 {
				var parts []content
				if m.Content != "" {
					parts = append(parts, content{Type: "text", Text: m.Content})
				}
				for _, tc := range m.ToolCalls {
					parts = append(parts, content{Type: "tool_use", ID: tc.ID, Name: tc.Name, Input: tc.Args})
				}
				built = append(built, msg{Role: "assistant", Content: parts})
			} else {
				built = append(built, msg{Role: "assistant", Content: m.Content})
			}
		case "tool":
			if m.ToolResult != nil {
				built = append(built, msg{
					Role: "user",
					Content: []content{{
						Type:      "tool_result",
						ToolUseID: m.ToolResult.ToolCallID,
						Content:   m.ToolResult.Content,
					}},
				})
			}
		}
	}

	body := map[string]interface{}{
		"model":      p.model,
		"max_tokens": 4096,
		"system":     system,
		"messages":   built,
		"stream":     true,
	}
	if len(tools) > 0 {
		body["tools"] = tools
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("anthropic: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// Track tool_use blocks being assembled
	type pendingTool struct {
		id    string
		name  string
		input strings.Builder
	}
	var pending []pendingTool
	var curIdx int = -1

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		raw := strings.TrimPrefix(line, "data: ")
		if raw == "[DONE]" {
			break
		}

		var ev struct {
			Type  string `json:"type"`
			Index int    `json:"index"`
			Delta *struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta"`
			ContentBlock *struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block"`
		}
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			continue
		}

		switch ev.Type {
		case "content_block_start":
			if ev.ContentBlock != nil && ev.ContentBlock.Type == "tool_use" {
				pending = append(pending, pendingTool{
					id:   ev.ContentBlock.ID,
					name: ev.ContentBlock.Name,
				})
				curIdx = len(pending) - 1
			}
		case "content_block_delta":
			if ev.Delta == nil {
				continue
			}
			if ev.Delta.Type == "text_delta" {
				out <- StreamEvent{Delta: ev.Delta.Text}
			} else if ev.Delta.Type == "input_json_delta" && curIdx >= 0 {
				pending[curIdx].input.WriteString(ev.Delta.PartialJSON)
			}
		case "message_delta":
			// stop_reason may be "tool_use"
		case "message_stop":
			// assemble tool calls
			if len(pending) > 0 {
				calls := make([]ToolCall, len(pending))
				for i, pt := range pending {
					var args map[string]interface{}
					json.Unmarshal([]byte(pt.input.String()), &args)
					calls[i] = ToolCall{ID: pt.id, Name: pt.name, Args: args}
				}
				out <- StreamEvent{ToolCalls: calls}
			}
		}
	}
	return scanner.Err()
}

// ── OpenAI ────────────────────────────────────────────────────────────────────

type openaiProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func newOpenAI(apiKey, model, baseURL string, client *http.Client) Provider {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &openaiProvider{apiKey: apiKey, model: model, baseURL: baseURL, client: client}
}

func (p *openaiProvider) Name() string { return humanModelName(p.model, "GPT") }

func (p *openaiProvider) StreamChat(ctx context.Context, system string, msgs []ProviderMessage, tools []interface{}, out chan<- StreamEvent) error {
	type fnCall struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	}
	type tc struct {
		ID       string `json:"id,omitempty"`
		Type     string `json:"type,omitempty"`
		Function fnCall `json:"function,omitempty"`
	}
	type msg struct {
		Role       string  `json:"role"`
		Content    string  `json:"content,omitempty"`
		ToolCalls  []tc    `json:"tool_calls,omitempty"`
		ToolCallID string  `json:"tool_call_id,omitempty"`
		Name       string  `json:"name,omitempty"`
	}

	built := []msg{{Role: "system", Content: system}}
	for _, m := range msgs {
		switch m.Role {
		case "user":
			built = append(built, msg{Role: "user", Content: m.Content})
		case "assistant":
			am := msg{Role: "assistant", Content: m.Content}
			for _, call := range m.ToolCalls {
				b, _ := json.Marshal(call.Args)
				am.ToolCalls = append(am.ToolCalls, tc{
					ID:   call.ID,
					Type: "function",
					Function: fnCall{Name: call.Name, Arguments: string(b)},
				})
			}
			built = append(built, am)
		case "tool":
			if m.ToolResult != nil {
				built = append(built, msg{
					Role:       "tool",
					Content:    m.ToolResult.Content,
					ToolCallID: m.ToolResult.ToolCallID,
				})
			}
		}
	}

	body := map[string]interface{}{
		"model":    p.model,
		"messages": built,
		"stream":   true,
	}
	if len(tools) > 0 {
		body["tools"] = tools
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("openai: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// pending tool calls indexed by tool_call index
	type pendingTC struct {
		id   string
		name string
		args strings.Builder
	}
	pending := map[int]*pendingTC{}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		raw := strings.TrimPrefix(line, "data: ")
		if raw == "[DONE]" {
			break
		}
		var ev struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			continue
		}
		if len(ev.Choices) == 0 {
			continue
		}
		d := ev.Choices[0].Delta
		if d.Content != "" {
			out <- StreamEvent{Delta: d.Content}
		}
		for _, tcd := range d.ToolCalls {
			if _, ok := pending[tcd.Index]; !ok {
				pending[tcd.Index] = &pendingTC{id: tcd.ID, name: tcd.Function.Name}
			}
			pt := pending[tcd.Index]
			if tcd.ID != "" {
				pt.id = tcd.ID
			}
			if tcd.Function.Name != "" {
				pt.name = tcd.Function.Name
			}
			pt.args.WriteString(tcd.Function.Arguments)
		}
		if ev.Choices[0].FinishReason == "tool_calls" {
			calls := make([]ToolCall, 0, len(pending))
			for _, pt := range pending {
				var args map[string]interface{}
				json.Unmarshal([]byte(pt.args.String()), &args)
				calls = append(calls, ToolCall{ID: pt.id, Name: pt.name, Args: args})
			}
			out <- StreamEvent{ToolCalls: calls}
		}
	}
	return scanner.Err()
}

// ── Gemini ────────────────────────────────────────────────────────────────────

type geminiProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func newGemini(apiKey, model, baseURL string, client *http.Client) Provider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	return &geminiProvider{apiKey: apiKey, model: model, baseURL: baseURL, client: client}
}

func (p *geminiProvider) Name() string { return humanModelName(p.model, "Gemini") }

func (p *geminiProvider) StreamChat(ctx context.Context, system string, msgs []ProviderMessage, tools []interface{}, out chan<- StreamEvent) error {
	type part struct {
		Text         string      `json:"text,omitempty"`
		FunctionCall interface{} `json:"functionCall,omitempty"`
		FunctionResp interface{} `json:"functionResponse,omitempty"`
	}
	type content struct {
		Role  string `json:"role"`
		Parts []part `json:"parts"`
	}

	var contents []content
	for _, m := range msgs {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		switch m.Role {
		case "user":
			contents = append(contents, content{Role: "user", Parts: []part{{Text: m.Content}}})
		case "assistant":
			var parts []part
			if m.Content != "" {
				parts = append(parts, part{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				parts = append(parts, part{FunctionCall: map[string]interface{}{
					"name": tc.Name, "args": tc.Args,
				}})
			}
			contents = append(contents, content{Role: "model", Parts: parts})
		case "tool":
			if m.ToolResult != nil {
				var resp interface{}
				json.Unmarshal([]byte(m.ToolResult.Content), &resp)
				contents = append(contents, content{
					Role: "user",
					Parts: []part{{FunctionResp: map[string]interface{}{
						"name":     m.ToolResult.ToolCallID,
						"response": resp,
					}}},
				})
			}
		}
	}

	body := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []part{{Text: system}},
		},
		"contents": contents,
	}
	if len(tools) > 0 {
		body["tools"] = tools
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?key=%s&alt=sse",
		p.baseURL, p.model, p.apiKey)
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("gemini: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gemini error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text         string `json:"text"`
						FunctionCall *struct {
							Name string                 `json:"name"`
							Args map[string]interface{} `json:"args"`
						} `json:"functionCall"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
			continue
		}
		if len(ev.Candidates) == 0 {
			continue
		}
		var toolCalls []ToolCall
		for _, pt := range ev.Candidates[0].Content.Parts {
			if pt.Text != "" {
				out <- StreamEvent{Delta: pt.Text}
			}
			if pt.FunctionCall != nil {
				toolCalls = append(toolCalls, ToolCall{
					ID:   pt.FunctionCall.Name,
					Name: pt.FunctionCall.Name,
					Args: pt.FunctionCall.Args,
				})
			}
		}
		if len(toolCalls) > 0 {
			out <- StreamEvent{ToolCalls: toolCalls}
		}
	}
	return scanner.Err()
}

// ── Ollama ────────────────────────────────────────────────────────────────────

type ollamaProvider struct {
	model   string
	baseURL string
	client  *http.Client
}

func newOllama(model, baseURL string, client *http.Client) Provider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &ollamaProvider{model: model, baseURL: baseURL, client: client}
}

func (p *ollamaProvider) Name() string { return humanModelName(p.model, "Ollama") }

func (p *ollamaProvider) StreamChat(ctx context.Context, system string, msgs []ProviderMessage, tools []interface{}, out chan<- StreamEvent) error {
	type fnCall struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	type tc struct {
		ID       string `json:"id,omitempty"`
		Type     string `json:"type"`
		Function fnCall `json:"function"`
	}
	type msg struct {
		Role       string `json:"role"`
		Content    string `json:"content,omitempty"`
		ToolCalls  []tc   `json:"tool_calls,omitempty"`
		ToolCallID string  `json:"tool_call_id,omitempty"`
	}

	built := []msg{{Role: "system", Content: system}}
	for _, m := range msgs {
		switch m.Role {
		case "user":
			built = append(built, msg{Role: "user", Content: m.Content})
		case "assistant":
			am := msg{Role: "assistant", Content: m.Content}
			for _, call := range m.ToolCalls {
				am.ToolCalls = append(am.ToolCalls, tc{
					ID: call.ID, Type: "function",
					Function: fnCall{Name: call.Name, Arguments: call.Args},
				})
			}
			built = append(built, am)
		case "tool":
			if m.ToolResult != nil {
				built = append(built, msg{
					Role: "tool", Content: m.ToolResult.Content,
					ToolCallID: m.ToolResult.ToolCallID,
				})
			}
		}
	}

	body := map[string]interface{}{
		"model":    p.model,
		"messages": built,
		"stream":   true,
	}
	if len(tools) > 0 {
		body["tools"] = tools
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var ev struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Function struct {
						Name      string                 `json:"name"`
						Arguments map[string]interface{} `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := decoder.Decode(&ev); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if ev.Message.Content != "" {
			out <- StreamEvent{Delta: ev.Message.Content}
		}
		if len(ev.Message.ToolCalls) > 0 {
			calls := make([]ToolCall, len(ev.Message.ToolCalls))
			for i, tc := range ev.Message.ToolCalls {
				calls[i] = ToolCall{
					ID:   fmt.Sprintf("call_%d", i),
					Name: tc.Function.Name,
					Args: tc.Function.Arguments,
				}
			}
			out <- StreamEvent{ToolCalls: calls}
		}
		if ev.Done {
			break
		}
	}
	return nil
}

// ── Unconfigured ──────────────────────────────────────────────────────────────

type unconfiguredProvider struct{}

func (p *unconfiguredProvider) Name() string { return "Not configured" }
func (p *unconfiguredProvider) StreamChat(_ context.Context, _ string, _ []ProviderMessage, _ []interface{}, out chan<- StreamEvent) error {
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func humanModelName(model, fallbackPrefix string) string {
	replacer := strings.NewReplacer(
		"claude-sonnet-4-6", "Claude Sonnet 4.6",
		"claude-sonnet-4-5", "Claude Sonnet 4.5",
		"claude-opus-4-6", "Claude Opus 4.6",
		"claude-haiku-4-5", "Claude Haiku 4.5",
		"claude-3-5-sonnet-20241022", "Claude 3.5 Sonnet",
		"claude-3-5-haiku-20241022", "Claude 3.5 Haiku",
		"claude-3-opus-20240229", "Claude 3 Opus",
		"gpt-4o", "GPT-4o",
		"gpt-4o-mini", "GPT-4o mini",
		"gpt-4-turbo", "GPT-4 Turbo",
		"gpt-4", "GPT-4",
		"gpt-3.5-turbo", "GPT-3.5 Turbo",
		"o1", "o1",
		"o3-mini", "o3-mini",
		"gemini-2.0-flash", "Gemini 2.0 Flash",
		"gemini-1.5-pro", "Gemini 1.5 Pro",
		"gemini-1.5-flash", "Gemini 1.5 Flash",
		"llama3.2", "Llama 3.2",
		"llama3.1", "Llama 3.1",
		"llama3", "Llama 3",
		"mistral", "Mistral",
		"mixtral", "Mixtral",
		"phi3", "Phi-3",
	)
	if renamed := replacer.Replace(model); renamed != model {
		return renamed
	}
	return model
}
