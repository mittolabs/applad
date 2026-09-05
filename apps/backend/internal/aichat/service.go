// Package aichat provides a streaming AI assistant endpoint that supports
// multiple providers: Anthropic, OpenAI, Gemini, and Ollama.
// Provider, model, and API key are configured via environment variables:
//
//	AI_PROVIDER = anthropic | openai | gemini | ollama  (default: anthropic)
//	AI_API_KEY  = your API key
//	AI_MODEL    = model override (optional)
//	AI_BASE_URL = endpoint override (optional, useful for Ollama)
package aichat

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const systemPrompt = `You are Applad AI, an intelligent assistant built into the Applad developer console.

Applad is a self-hosted backend-as-a-service (BaaS) platform. You help developers who are actively using the Applad console.

What Applad provides:
- Auth: user accounts, sessions, OAuth2 (15+ providers), MFA (TOTP), magic links, anonymous sessions
- Databases: PostgreSQL-backed tables with typed columns, indexes, relationships, row CRUD, cursor pagination, schema-scoped SQL
- Storage: file buckets, image transforms (resize, format), chunked uploads, S3 or local driver, optional ClamAV antivirus
- Functions: container-based serverless in Node.js, Bun, Python, Go, Dart, Rust, Ruby, PHP, or any Dockerfile
- Realtime: WebSocket pub/sub — auto-publishes on every database and storage change via PostgreSQL LISTEN/NOTIFY
- Messaging: SMTP email, Twilio SMS, FCM push notifications, topics & subscribers
- Deploy: deployment lifecycle management with Docker-based executor for sites, containers, mobile, and desktop
- Workflows: native DAG engine with HTTP, transform, condition, delay, code, and email nodes plus webhook triggers
- Teams: team CRUD, memberships, role-based access

SDKs available: JavaScript/TypeScript (client), Dart (client + server), Node.js (server), Go (server), Python (server).

Your role:
- Help developers understand and use Applad's features
- Write code using Applad's SDKs on request
- Help debug issues with functions, deployments, and workflows
- Guide resource creation and configuration through the console
- Answer questions about self-hosting, configuration, and environment variables
- Use the available tools to look up live data from the user's Applad instance

Be concise, technical, and practical. Format code in markdown code blocks with the language specified.`

// Message is a single chat turn.
type Message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// Service manages the AI provider and exposes streaming chat.
type Service struct {
	provider Provider
}

// NewService creates a Service from config values.
// provider: "anthropic" | "openai" | "gemini" | "ollama"
// apiKey:   unified API key
// model:    optional model override
// baseURL:  optional endpoint override
func NewService(provider, apiKey, model, baseURL string) *Service {
	client := &http.Client{Timeout: 120 * time.Second}

	var p Provider
	switch strings.ToLower(provider) {
	case "anthropic":
		p = newAnthropic(apiKey, model, baseURL, client)
	case "openai":
		p = newOpenAI(apiKey, model, baseURL, client)
	case "gemini":
		p = newGemini(apiKey, model, baseURL, client)
	case "ollama":
		p = newOllama(model, baseURL, client)
	default:
		// No provider configured — IsConfigured() will return false.
		p = &unconfiguredProvider{}
	}

	return &Service{provider: p}
}

// ModelName returns the human-readable label for the configured model.
func (s *Service) ModelName() string {
	return s.provider.Name()
}

// IsConfigured reports whether the AI feature is enabled (provider + key are set).
// A missing AI_MODEL is caught at chat time with a clear message, not here.
func (s *Service) IsConfigured() bool {
	switch p := s.provider.(type) {
	case *unconfiguredProvider:
		return false
	case *ollamaProvider:
		return true // local, no key required
	case *anthropicProvider:
		return p.apiKey != ""
	case *openaiProvider:
		return p.apiKey != ""
	case *geminiProvider:
		return p.apiKey != ""
	}
	return false
}

// modelName returns the raw model ID string from the active provider, or "" if unset.
func (s *Service) modelName() string {
	switch p := s.provider.(type) {
	case *anthropicProvider:
		return p.model
	case *openaiProvider:
		return p.model
	case *geminiProvider:
		return p.model
	case *ollamaProvider:
		return p.model
	}
	return ""
}

// toolSchemas returns the tool definitions in the format expected by the active provider.
func (s *Service) toolSchemas() []interface{} {
	switch s.provider.(type) {
	case *anthropicProvider:
		return ToAnthropicTools()
	case *openaiProvider, *ollamaProvider:
		return ToOpenAITools()
	case *geminiProvider:
		return ToGeminiTools()
	}
	return nil
}

// StreamChat sends the conversation to the provider and streams token deltas
// to out. It closes out when streaming is complete or on error.
// The caller must drain out until it is closed.
// token is the console JWT used by the ToolExecutor to call internal APIs.
// executor may be nil when tools are not needed.
func (s *Service) StreamChat(ctx context.Context, messages []Message, pageContext string, token string, executor *ToolExecutor, out chan<- string) error {
	defer close(out)

	if !s.IsConfigured() {
		out <- "AI chat is not configured. Set `AI_PROVIDER` and `AI_API_KEY` in your environment to enable it."
		return nil
	}

	if s.modelName() == "" {
		out <- "No model is selected. Set `AI_MODEL` in your environment — see `.env.example` for the list of options for your provider."
		return nil
	}

	sys := systemPrompt
	if pageContext != "" {
		sys += "\n\nThe user is currently on the " + pageContext + " page of the Applad console."
	}

	tools := s.toolSchemas()

	// Convert []Message → []ProviderMessage
	provMsgs := make([]ProviderMessage, len(messages))
	for i, m := range messages {
		provMsgs[i] = ProviderMessage{Role: m.Role, Content: m.Content}
	}

	// Tool-calling loop — runs at most 8 rounds to prevent runaway recursion.
	for round := 0; round < 8; round++ {
		evCh := make(chan StreamEvent, 64)
		errCh := make(chan error, 1)

		go func(msgs []ProviderMessage) {
			errCh <- s.provider.StreamChat(ctx, sys, msgs, tools, evCh)
		}(provMsgs)

		var assistantText strings.Builder
		var pendingCalls []ToolCall
		done := false

		for !done {
			select {
			case ev, ok := <-evCh:
				if !ok {
					done = true
					break
				}
				if ev.Delta != "" {
					assistantText.WriteString(ev.Delta)
					out <- ev.Delta
				}
				if len(ev.ToolCalls) > 0 {
					pendingCalls = append(pendingCalls, ev.ToolCalls...)
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := <-errCh; err != nil {
			return err
		}

		// No tool calls — conversation turn is complete.
		if len(pendingCalls) == 0 {
			return nil
		}

		// Append the assistant turn (may include reasoning text + tool calls).
		provMsgs = append(provMsgs, ProviderMessage{
			Role:      "assistant",
			Content:   assistantText.String(),
			ToolCalls: pendingCalls,
		})

		// Execute each tool and stream a brief inline notification.
		for _, tc := range pendingCalls {
			label := toolLabel(tc.Name)
			out <- "\n\n_" + label + "..._\n\n"

			var result ToolResult
			if executor != nil && token != "" {
				result = executor.Execute(ctx, token, tc.Name, tc.Args)
			} else {
				result = ToolResult{Content: `{"error":"tool executor not available"}`}
			}
			result.ToolCallID = tc.ID

			provMsgs = append(provMsgs, ProviderMessage{
				Role:       "tool",
				ToolResult: &result,
			})
		}

		// Reset for the next provider turn.
		pendingCalls = nil
	}

	return nil
}

// Chat is a non-streaming convenience wrapper (collects full response).
func (s *Service) Chat(ctx context.Context, messages []Message, pageContext string, token string, executor *ToolExecutor) (*Message, error) {
	out := make(chan string, 256)
	var sb strings.Builder
	errCh := make(chan error, 1)

	go func() {
		errCh <- s.StreamChat(ctx, messages, pageContext, token, executor, out)
	}()

	for delta := range out {
		sb.WriteString(delta)
	}
	if err := <-errCh; err != nil {
		return nil, err
	}
	return &Message{Role: "assistant", Content: sb.String()}, nil
}

// Explain runs a single, tool-free completion: a system instruction and a user
// prompt in, the assistant's text out. It is the one-shot cousin of Chat, for
// callers that want a summary rather than a conversation.
func (s *Service) Explain(ctx context.Context, system, user string) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("ai is not configured")
	}
	msg, err := s.Chat(ctx, []Message{{Role: "user", Content: user}}, system, "", nil)
	if err != nil {
		return "", err
	}
	return msg.Content, nil
}

// toolLabel returns a human-readable status string for a tool being executed.
func toolLabel(name string) string {
	labels := map[string]string{
		"list_projects":     "Listing your projects",
		"get_project_usage": "Fetching project usage",
		"search_project":    "Searching project",
		"list_databases":    "Listing databases",
		"list_tables":       "Listing tables",
		"list_users":        "Listing users",
		"list_functions":    "Listing functions",
		"list_workflows":    "Listing workflows",
		"trigger_workflow":  "Triggering workflow",
		"list_buckets":      "Listing storage buckets",
		"list_api_keys":     "Listing API keys",
		"list_deployments":  "Listing deployments",
		"list_platforms":    "Listing platforms",
		"get_auth_config":   "Fetching auth configuration",
	}
	if l, ok := labels[name]; ok {
		return l
	}
	return "Using " + name
}
