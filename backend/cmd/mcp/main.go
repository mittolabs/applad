// Command mcp starts the Applad Model Context Protocol (MCP) server.
// It exposes Applad services as MCP tools that LLMs and AI coding assistants
// can call via the MCP JSON-RPC protocol over stdio or HTTP.
//
// Supported tools:
//   - databases_list, databases_query, databases_create_row, databases_update_row, databases_delete_row
//   - auth_list_users, auth_get_user, auth_create_user
//   - storage_list_files, storage_get_file
//   - deploy_list, deploy_trigger
//   - workflows_list, workflows_execute
//   - search_query
//   - analytics_track
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// JSON-RPC 2.0 structures
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool describes a single MCP tool.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// Server is the MCP server.
type Server struct {
	baseURL    string
	projectID  string
	apiKey     string
	httpClient *http.Client
}

func main() {
	s := &Server{
		baseURL:    getenv("APPLAD_URL", "http://localhost:80/v1"),
		projectID:  getenv("APPLAD_PROJECT", ""),
		apiKey:     getenv("APPLAD_KEY", ""),
		httpClient: &http.Client{},
	}

	mode := getenv("MCP_TRANSPORT", "stdio")
	if mode == "http" {
		port := getenv("MCP_PORT", "3100")
		log.Printf("MCP server listening on :%s (HTTP transport)", port)
		http.HandleFunc("/", s.httpHandler)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal(err)
		}
		return
	}

	log.Printf("MCP server started (stdio transport). Project: %s, URL: %s", s.projectID, s.baseURL)
	s.stdioLoop()
}

// stdioLoop reads JSON-RPC requests from stdin and writes responses to stdout.
func (s *Server) stdioLoop() {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			encoder.Encode(response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}}) //nolint:errcheck
			continue
		}
		res := s.handle(&req)
		if res != nil { // nil = notification (no response expected)
			encoder.Encode(res) //nolint:errcheck
		}
	}
}

// httpHandler handles MCP requests over HTTP POST.
func (s *Server) httpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}}) //nolint:errcheck
		return
	}
	res := s.handle(&req)
	if res == nil { // notification — no response body
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res) //nolint:errcheck
}

// handle dispatches a JSON-RPC request to the appropriate tool.
func (s *Server) handle(req *request) *response {
	switch req.Method {

	// ── MCP protocol ──────────────────────────────────────────────────────────
	case "initialize":
		return s.ok(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo": map[string]interface{}{
				"name":    "applad-mcp",
				"version": "1.0.0",
			},
		})

	case "tools/list":
		return s.ok(req.ID, map[string]interface{}{"tools": s.toolList()})

	case "tools/call":
		return s.callTool(req)

	// ── Notifications (fire-and-forget) ───────────────────────────────────────
	case "notifications/initialized":
		return nil

	default:
		return &response{
			JSONRPC: "2.0", ID: req.ID,
			Error: &rpcError{Code: -32601, Message: "method not found"},
		}
	}
}

func (s *Server) callTool(req *request) *response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.err(req.ID, -32602, "invalid params")
	}

	var args map[string]interface{}
	json.Unmarshal(params.Arguments, &args) //nolint:errcheck

	result, err := s.executeTool(params.Name, args)
	if err != nil {
		return s.err(req.ID, -32000, err.Error())
	}

	body, _ := json.MarshalIndent(result, "", "  ")
	return s.ok(req.ID, map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": string(body)},
		},
	})
}

// executeTool routes a tool call to the Applad API.
func (s *Server) executeTool(name string, args map[string]interface{}) (interface{}, error) {
	switch name {

	// ── Databases ──────────────────────────────────────────────────────────────
	case "databases_list":
		return s.apiGet("/databases")

	case "databases_list_tables":
		dbID := strArg(args, "databaseId")
		return s.apiGet("/databases/" + dbID + "/collections")

	case "databases_query":
		dbID := strArg(args, "databaseId")
		tableID := strArg(args, "tableId")
		limit := intArgOrDefault(args, "limit", 25)
		return s.apiGet(fmt.Sprintf("/databases/%s/collections/%s/documents?limit=%d", dbID, tableID, limit))

	case "databases_create_row":
		dbID := strArg(args, "databaseId")
		tableID := strArg(args, "tableId")
		data := mapArg(args, "data")
		return s.apiPost(fmt.Sprintf("/databases/%s/collections/%s/documents", dbID, tableID), data)

	case "databases_get_row":
		dbID := strArg(args, "databaseId")
		tableID := strArg(args, "tableId")
		rowID := strArg(args, "rowId")
		return s.apiGet(fmt.Sprintf("/databases/%s/collections/%s/documents/%s", dbID, tableID, rowID))

	case "databases_update_row":
		dbID := strArg(args, "databaseId")
		tableID := strArg(args, "tableId")
		rowID := strArg(args, "rowId")
		data := mapArg(args, "data")
		return s.apiPatch(fmt.Sprintf("/databases/%s/collections/%s/documents/%s", dbID, tableID, rowID), data)

	case "databases_delete_row":
		dbID := strArg(args, "databaseId")
		tableID := strArg(args, "tableId")
		rowID := strArg(args, "rowId")
		return s.apiDelete(fmt.Sprintf("/databases/%s/collections/%s/documents/%s", dbID, tableID, rowID))

	// ── Auth / Users ──────────────────────────────────────────────────────────
	case "auth_list_users":
		limit := intArgOrDefault(args, "limit", 25)
		return s.apiGet(fmt.Sprintf("/users?limit=%d", limit))

	case "auth_get_user":
		return s.apiGet("/users/" + strArg(args, "userId"))

	case "auth_create_user":
		return s.apiPost("/users", map[string]interface{}{
			"userId": strArg(args, "userId"),
			"email":  strArg(args, "email"),
			"name":   strArg(args, "name"),
		})

	// ── Storage ───────────────────────────────────────────────────────────────
	case "storage_list_buckets":
		return s.apiGet("/storage/buckets")

	case "storage_list_files":
		bucketID := strArg(args, "bucketId")
		return s.apiGet("/storage/buckets/" + bucketID + "/files")

	// ── Deploy ────────────────────────────────────────────────────────────────
	case "deploy_list_targets":
		return s.apiGet("/deploy/targets")

	case "deploy_trigger":
		targetID := strArg(args, "targetId")
		return s.apiPost("/deploy/targets/"+targetID+"/trigger", nil)

	// ── Workflows ─────────────────────────────────────────────────────────────
	case "workflows_list":
		return s.apiGet("/workflows")

	case "workflows_execute":
		workflowID := strArg(args, "workflowId")
		return s.apiPost("/workflows/"+workflowID+"/executions", mapArg(args, "input"))

	// ── Search ────────────────────────────────────────────────────────────────
	case "search_query":
		indexID := strArg(args, "indexId")
		return s.apiPost("/search/indexes/"+indexID+"/query", map[string]interface{}{
			"query": strArg(args, "query"),
			"limit": intArgOrDefault(args, "limit", 20),
		})

	// ── Analytics ─────────────────────────────────────────────────────────────
	case "analytics_track":
		return s.apiPost("/analytics/events", map[string]interface{}{
			"event":      strArg(args, "event"),
			"userId":     strArg(args, "userId"),
			"properties": mapArg(args, "properties"),
		})

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// ── toolList returns the MCP tool catalog ─────────────────────────────────────

func (s *Server) toolList() []Tool {
	tools := []struct {
		name, desc string
		schema     string
	}{
		{"databases_list", "List all databases in the project", `{"type":"object","properties":{}}`},
		{"databases_list_tables", "List tables in a database", `{"type":"object","properties":{"databaseId":{"type":"string","description":"Database ID"}},"required":["databaseId"]}`},
		{"databases_query", "Query rows from a table", `{"type":"object","properties":{"databaseId":{"type":"string"},"tableId":{"type":"string"},"limit":{"type":"integer","default":25}},"required":["databaseId","tableId"]}`},
		{"databases_create_row", "Create a row in a table", `{"type":"object","properties":{"databaseId":{"type":"string"},"tableId":{"type":"string"},"data":{"type":"object"}},"required":["databaseId","tableId","data"]}`},
		{"databases_get_row", "Get a single row by ID", `{"type":"object","properties":{"databaseId":{"type":"string"},"tableId":{"type":"string"},"rowId":{"type":"string"}},"required":["databaseId","tableId","rowId"]}`},
		{"databases_update_row", "Update a row", `{"type":"object","properties":{"databaseId":{"type":"string"},"tableId":{"type":"string"},"rowId":{"type":"string"},"data":{"type":"object"}},"required":["databaseId","tableId","rowId","data"]}`},
		{"databases_delete_row", "Delete a row", `{"type":"object","properties":{"databaseId":{"type":"string"},"tableId":{"type":"string"},"rowId":{"type":"string"}},"required":["databaseId","tableId","rowId"]}`},
		{"auth_list_users", "List project users", `{"type":"object","properties":{"limit":{"type":"integer","default":25}}}`},
		{"auth_get_user", "Get a user by ID", `{"type":"object","properties":{"userId":{"type":"string"}},"required":["userId"]}`},
		{"auth_create_user", "Create a new user", `{"type":"object","properties":{"userId":{"type":"string"},"email":{"type":"string"},"name":{"type":"string"}},"required":["email"]}`},
		{"storage_list_buckets", "List storage buckets", `{"type":"object","properties":{}}`},
		{"storage_list_files", "List files in a bucket", `{"type":"object","properties":{"bucketId":{"type":"string"}},"required":["bucketId"]}`},
		{"deploy_list_targets", "List deploy targets", `{"type":"object","properties":{}}`},
		{"deploy_trigger", "Trigger a deployment", `{"type":"object","properties":{"targetId":{"type":"string"}},"required":["targetId"]}`},
		{"workflows_list", "List workflows", `{"type":"object","properties":{}}`},
		{"workflows_execute", "Execute a workflow", `{"type":"object","properties":{"workflowId":{"type":"string"},"input":{"type":"object"}},"required":["workflowId"]}`},
		{"search_query", "Search an index", `{"type":"object","properties":{"indexId":{"type":"string"},"query":{"type":"string"},"limit":{"type":"integer"}},"required":["indexId","query"]}`},
		{"analytics_track", "Track an analytics event", `{"type":"object","properties":{"event":{"type":"string"},"userId":{"type":"string"},"properties":{"type":"object"}},"required":["event"]}`},
	}

	out := make([]Tool, len(tools))
	for i, t := range tools {
		out[i] = Tool{Name: t.name, Description: t.desc, InputSchema: json.RawMessage(t.schema)}
	}
	return out
}

// ── API helpers ───────────────────────────────────────────────────────────────

func (s *Server) apiGet(path string) (interface{}, error) {
	req, _ := http.NewRequest(http.MethodGet, s.baseURL+path, nil)
	s.setHeaders(req)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out interface{}
	json.NewDecoder(resp.Body).Decode(&out) //nolint:errcheck
	return out, nil
}

func (s *Server) apiPost(path string, body interface{}) (interface{}, error) {
	return s.apiRequest(http.MethodPost, path, body)
}

func (s *Server) apiPatch(path string, body interface{}) (interface{}, error) {
	return s.apiRequest(http.MethodPatch, path, body)
}

func (s *Server) apiDelete(path string) (interface{}, error) {
	return s.apiRequest(http.MethodDelete, path, nil)
}

func (s *Server) apiRequest(method, path string, body interface{}) (interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(b))
	}
	req, _ := http.NewRequest(method, s.baseURL+path, bodyReader)
	s.setHeaders(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out interface{}
	json.NewDecoder(resp.Body).Decode(&out) //nolint:errcheck
	return out, nil
}

func (s *Server) setHeaders(req *http.Request) {
	if s.projectID != "" {
		req.Header.Set("X-Applad-Project", s.projectID)
	}
	if s.apiKey != "" {
		req.Header.Set("X-Applad-Key", s.apiKey)
	}
}

// ── response helpers ──────────────────────────────────────────────────────────

func (s *Server) ok(id, result interface{}) *response {
	return &response{JSONRPC: "2.0", ID: id, Result: result}
}

func (s *Server) err(id interface{}, code int, msg string) *response {
	return &response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
}

// ── arg helpers ───────────────────────────────────────────────────────────────

func strArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intArgOrDefault(args map[string]interface{}, key string, def int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

func mapArg(args map[string]interface{}, key string) map[string]interface{} {
	if v, ok := args[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return map[string]interface{}{}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
