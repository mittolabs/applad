// Package aichat provides an AI assistant endpoint that proxies to the
// Anthropic Claude API. The assistant knows about Applad's features and
// can answer questions, help debug, and guide resource creation.
package aichat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const systemPrompt = `You are Applad AI, an intelligent assistant built into the Applad developer console.

Applad is a self-hosted backend-as-a-service (BaaS) platform that gives developers:
- **Auth**: user accounts, sessions, OAuth2 (15+ providers), MFA, magic links
- **Databases**: PostgreSQL-backed schema orchestration, table/row CRUD, row-level security
- **Storage**: file buckets, image transforms, chunked uploads, S3 or local driver
- **Functions**: serverless functions in Node, Python, Go, Dart, Bun, Ruby, PHP, Rust with container execution
- **Messaging**: SMTP email, Twilio SMS, FCM push notifications
- **Deploy**: Sites, Containers, Mobile, Desktop deployments with Docker
- **Workflows**: native DAG-based workflow engine with HTTP, transform, condition, code nodes
- **Realtime**: WebSocket pub/sub with PostgreSQL LISTEN/NOTIFY

You help developers:
- Understand and query their project's resources
- Create and configure Applad resources via the console
- Debug issues with functions, deployments, and workflows
- Write code using Applad's SDKs (Dart, JS/TS, Node.js, Go, Python)
- Understand best practices for BaaS architecture

Be concise, practical, and technical. Format code in markdown code blocks.`

// Message is a single chat turn.
type Message struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// Service proxies chat messages to the Anthropic Claude API.
type Service struct {
	apiKey string
	client *http.Client
}

// NewService creates a new AI chat service.
func NewService(apiKey string) *Service {
	return &Service{
		apiKey: apiKey,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// Chat sends a conversation to Claude and returns the assistant's reply.
// If no API key is configured, returns a placeholder message.
func (s *Service) Chat(ctx context.Context, messages []Message, pageContext string) (*Message, error) {
	if s.apiKey == "" {
		return &Message{
			Role:    "assistant",
			Content: "AI chat is not configured. Add `ANTHROPIC_API_KEY` to your environment variables to enable it.",
		}, nil
	}

	sys := systemPrompt
	if pageContext != "" {
		sys += "\n\nThe user is currently on: " + pageContext
	}

	type anthropicMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	msgs := make([]anthropicMsg, len(messages))
	for i, m := range messages {
		msgs[i] = anthropicMsg{Role: m.Role, Content: m.Content}
	}

	body := map[string]interface{}{
		"model":      "claude-sonnet-4-6",
		"max_tokens": 2048,
		"system":     sys,
		"messages":   msgs,
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("aichat: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aichat: api request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("aichat: parse response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("aichat: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("aichat: empty response from API")
	}

	return &Message{Role: "assistant", Content: result.Content[0].Text}, nil
}
