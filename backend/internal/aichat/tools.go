package aichat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ── Tool definitions ──────────────────────────────────────────────────────────

// ToolDef describes a callable tool in provider-agnostic terms.
type ToolDef struct {
	Name        string
	Description string
	Properties  map[string]ToolProp
	Required    []string
}

// ToolProp is a single input parameter schema.
type ToolProp struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ToolCall is a tool invocation requested by the AI.
type ToolCall struct {
	ID   string
	Name string
	Args map[string]interface{}
}

// ToolResult is the outcome of executing a tool, returned to the AI.
type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
}

// AllTools is the complete set of tools available to the AI assistant.
var AllTools = []ToolDef{
	{
		Name:        "list_projects",
		Description: "List all projects the user has access to. Returns project IDs, names, and descriptions.",
		Properties:  map[string]ToolProp{},
	},
	{
		Name:        "get_project_usage",
		Description: "Get usage statistics for a project: request counts, active users, storage used, function executions.",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID ($id field from list_projects)"},
		},
		Required: []string{"project_id"},
	},
	{
		Name:        "search_project",
		Description: "Search for resources (databases, functions, buckets, workflows, users, deployments) within a project by name or keyword.",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID"},
			"query":      {Type: "string", Description: "Search keyword"},
		},
		Required: []string{"project_id", "query"},
	},
	{
		Name:        "list_databases",
		Description: "List all databases in a project.",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID"},
		},
		Required: []string{"project_id"},
	},
	{
		Name:        "list_tables",
		Description: "List all tables in a specific database, including column counts and row counts.",
		Properties: map[string]ToolProp{
			"project_id":  {Type: "string", Description: "The project ID"},
			"database_id": {Type: "string", Description: "The database ID"},
		},
		Required: []string{"project_id", "database_id"},
	},
	{
		Name:        "list_users",
		Description: "List users registered in a project. Returns name, email, and registration date.",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID"},
			"limit":      {Type: "string", Description: "Max results (default 20, max 100)"},
		},
		Required: []string{"project_id"},
	},
	{
		Name:        "list_functions",
		Description: "List all serverless functions in a project, including runtime, status, and last execution time.",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID"},
		},
		Required: []string{"project_id"},
	},
	{
		Name:        "list_workflows",
		Description: "List all workflows in a project, including trigger type and last execution status.",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID"},
		},
		Required: []string{"project_id"},
	},
	{
		Name:        "trigger_workflow",
		Description: "Manually trigger a workflow execution.",
		Properties: map[string]ToolProp{
			"project_id":  {Type: "string", Description: "The project ID"},
			"workflow_id": {Type: "string", Description: "The workflow ID to execute"},
		},
		Required: []string{"project_id", "workflow_id"},
	},
	{
		Name:        "list_buckets",
		Description: "List all storage buckets in a project.",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID"},
		},
		Required: []string{"project_id"},
	},
	{
		Name:        "list_api_keys",
		Description: "List API keys for a project (key values are masked).",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID"},
		},
		Required: []string{"project_id"},
	},
	{
		Name:        "list_deployments",
		Description: "List deployments (deploy targets) in a project and their current status.",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID"},
		},
		Required: []string{"project_id"},
	},
	{
		Name:        "list_platforms",
		Description: "List registered platforms (web, iOS, Android, desktop) for a project.",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID"},
		},
		Required: []string{"project_id"},
	},
	{
		Name:        "get_auth_config",
		Description: "Get the authentication configuration for a project: enabled providers, session settings.",
		Properties: map[string]ToolProp{
			"project_id": {Type: "string", Description: "The project ID"},
		},
		Required: []string{"project_id"},
	},
}

// ── Tool executor ─────────────────────────────────────────────────────────────

// ToolExecutor makes internal API calls to execute AI tool requests.
// It calls Applad's own REST endpoints using the console JWT for auth.
type ToolExecutor struct {
	baseURL string // e.g. "http://localhost:8080/v1"
	client  *http.Client
}

// NewToolExecutor creates an executor that calls the given internal API base URL.
func NewToolExecutor(port string, client *http.Client) *ToolExecutor {
	return &ToolExecutor{
		baseURL: fmt.Sprintf("http://localhost:%s/v1", port),
		client:  client,
	}
}

// Execute runs a tool by name, using the console JWT and optional project context.
func (e *ToolExecutor) Execute(ctx context.Context, token, name string, args map[string]interface{}) ToolResult {
	result, err := e.run(ctx, token, name, args)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error: %s", err.Error()), IsError: true}
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{Content: string(b)}
}

func (e *ToolExecutor) run(ctx context.Context, token, name string, args map[string]interface{}) (interface{}, error) {
	get := func(path string) (interface{}, error) { return e.get(ctx, token, path) }

	pid, _ := args["project_id"].(string)

	switch name {
	case "list_projects":
		return get("/projects")

	case "get_project_usage":
		return get(fmt.Sprintf("/projects/%s/usage", pid))

	case "search_project":
		q, _ := args["query"].(string)
		return get(fmt.Sprintf("/projects/%s/search?q=%s&limit=20", pid, q))

	case "list_databases":
		return get(fmt.Sprintf("/databases?projectId=%s", pid))

	case "list_tables":
		dbID, _ := args["database_id"].(string)
		return get(fmt.Sprintf("/databases/%s/tables?projectId=%s", dbID, pid))

	case "list_users":
		limit, _ := args["limit"].(string)
		if limit == "" {
			limit = "20"
		}
		return get(fmt.Sprintf("/users?projectId=%s&limit=%s", pid, limit))

	case "list_functions":
		return get(fmt.Sprintf("/functions?projectId=%s", pid))

	case "list_workflows":
		return get(fmt.Sprintf("/workflows?projectId=%s", pid))

	case "trigger_workflow":
		wfID, _ := args["workflow_id"].(string)
		return e.post(ctx, token, fmt.Sprintf("/workflows/%s/execute?projectId=%s", wfID, pid), nil)

	case "list_buckets":
		return get(fmt.Sprintf("/storage?projectId=%s", pid))

	case "list_api_keys":
		return get(fmt.Sprintf("/projects/%s/keys", pid))

	case "list_deployments":
		return get(fmt.Sprintf("/deploy?projectId=%s", pid))

	case "list_platforms":
		return get(fmt.Sprintf("/projects/%s/platforms", pid))

	case "get_auth_config":
		return get(fmt.Sprintf("/auth/settings?projectId=%s", pid))

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (e *ToolExecutor) get(ctx context.Context, token, path string) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", e.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), nil
	}
	return result, nil
}

func (e *ToolExecutor) post(ctx context.Context, token, path string, payload interface{}) (interface{}, error) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = strings.NewReader(string(b))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var result interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return string(raw), nil
	}
	return result, nil
}

// ── Provider-agnostic tool schema helpers ─────────────────────────────────────

// ToAnthropicTools converts AllTools to Anthropic's tool schema format.
func ToAnthropicTools() []interface{} {
	out := make([]interface{}, len(AllTools))
	for i, t := range AllTools {
		props := map[string]interface{}{}
		for k, v := range t.Properties {
			props[k] = map[string]interface{}{
				"type":        v.Type,
				"description": v.Description,
			}
			if len(v.Enum) > 0 {
				props[k].(map[string]interface{})["enum"] = v.Enum
			}
		}
		out[i] = map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"input_schema": map[string]interface{}{
				"type":       "object",
				"properties": props,
				"required":   t.Required,
			},
		}
	}
	return out
}

// ToOpenAITools converts AllTools to OpenAI's function-calling tool format.
// Ollama uses the same format.
func ToOpenAITools() []interface{} {
	out := make([]interface{}, len(AllTools))
	for i, t := range AllTools {
		props := map[string]interface{}{}
		for k, v := range t.Properties {
			props[k] = map[string]interface{}{
				"type":        v.Type,
				"description": v.Description,
			}
			if len(v.Enum) > 0 {
				props[k].(map[string]interface{})["enum"] = v.Enum
			}
		}
		out[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": props,
					"required":   t.Required,
				},
			},
		}
	}
	return out
}

// ToGeminiTools converts AllTools to Gemini's function declarations format.
func ToGeminiTools() []interface{} {
	decls := make([]interface{}, len(AllTools))
	for i, t := range AllTools {
		props := map[string]interface{}{}
		for k, v := range t.Properties {
			props[k] = map[string]interface{}{
				"type":        strings.ToUpper(v.Type),
				"description": v.Description,
			}
		}
		decls[i] = map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"parameters": map[string]interface{}{
				"type":       "OBJECT",
				"properties": props,
				"required":   t.Required,
			},
		}
	}
	return []interface{}{
		map[string]interface{}{"function_declarations": decls},
	}
}
