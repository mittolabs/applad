package endpoints

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	htmltemplate "html/template"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/mittolabs/applad/internal/databases"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/workflows"
)

// maxNodesPerEndpoint bounds the size of a single endpoint graph. Tighter than
// the workflow cap because an endpoint runs synchronously on the request path.
const maxNodesPerEndpoint = 200

// DataService is the subset of the databases service the data nodes use. It is
// an interface so the executor can be tested with a fake, and so *databases.Service
// satisfies it directly. Every method takes a userID + roles pair: that is the
// apply-rules toggle in action: the caller's identity for a scoped operation,
// or ("", ["service"]) for an elevated one.
type DataService interface {
	CreateRowWithAuth(ctx context.Context, projectID, databaseID, tableID, rowID string, data map[string]interface{}, permissions []string, userID string, roles []string) (*model.Row, error)
	GetRowWithAuth(ctx context.Context, rowID, tableID, databaseID, projectID, userID string, roles []string) (*model.Row, error)
	ListRowsWithAuth(ctx context.Context, projectID, databaseID, tableID, userID string, roles []string, params databases.ListParams) ([]*model.Row, int, error)
	UpdateRowWithAuth(ctx context.Context, rowID, tableID, databaseID, projectID string, data map[string]interface{}, permissions []string, userID string, roles []string) (*model.Row, error)
	DeleteRowWithAuth(ctx context.Context, rowID, tableID, databaseID, projectID, userID string, roles []string) error
}

// callContext carries the resolved caller identity for a single execution.
type callContext struct {
	projectID string
	userID    string // "" ⇒ anonymous / public
}

// Result is the HTTP response an endpoint execution produced, plus the trace.
type Result struct {
	StatusCode  int
	Body        interface{} // for JSON responses
	Text        string      // for text/html responses
	IsText      bool
	ContentType string // for text responses; JSON responses set it themselves
	Responded   bool   // a response node was reached
	Logs        []workflows.StepLog
	Err         string // a node failed; the caller returns 500
}

type executor struct {
	data DataService
}

func newExecutor(d DataService) *executor {
	return &executor{data: d}
}

// Run walks an endpoint graph synchronously against request and returns the
// response. It never returns an error: a node failure is captured in Result.Err
// so the handler can shape a 500 while still recording the trace.
func (x *executor) Run(ctx context.Context, ep *Endpoint, request map[string]interface{}, cc callContext) *Result {
	res := &Result{StatusCode: 200}

	if len(ep.Nodes) == 0 {
		res.StatusCode = 200
		res.Body = map[string]interface{}{}
		return res
	}
	if len(ep.Nodes) > maxNodesPerEndpoint {
		res.Err = fmt.Sprintf("endpoint exceeds the maximum of %d nodes", maxNodesPerEndpoint)
		res.StatusCode = 500
		return res
	}

	// Validate the request body against the endpoint's declared input schema
	// before running anything. A 400 here describes the caller's own input, so
	// it is safe to return in full (unlike a node's internal error).
	if problems := validateInput(ep.InputSchema, request["body"]); len(problems) > 0 {
		res.Responded = true
		res.StatusCode = 400
		res.Body = map[string]interface{}{"error": "Request validation failed", "fields": problems}
		return res
	}

	execCtx := map[string]interface{}{
		"request": request,
		"trigger": request, // alias, so workflow-style {{.trigger.*}} refs also work
		// consumed by any reused applad_* workflow node
		"__projectId__": cc.projectID,
	}

	nodeMap := make(map[string]*workflows.Node, len(ep.Nodes))
	for i := range ep.Nodes {
		nodeMap[ep.Nodes[i].ID] = &ep.Nodes[i]
	}
	adj := map[string][]string{}
	incoming := map[string][]string{}
	for _, e := range ep.Edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
		incoming[e.Target] = append(incoming[e.Target], e.Source)
	}

	order := workflows.TopoSort(ep.Nodes, ep.Edges)
	skipped := map[string]bool{}

	for _, nodeID := range order {
		node := nodeMap[nodeID]
		if node == nil {
			continue
		}

		// A node reachable only via not-taken branches is skipped; this
		// propagates a branch decision to its whole downstream subtree. Mirrors
		// the workflow runner. Topological order guarantees predecessors are
		// already decided here.
		if !skipped[nodeID] && len(incoming[nodeID]) > 0 {
			allDead := true
			for _, src := range incoming[nodeID] {
				if !skipped[src] {
					allDead = false
					break
				}
			}
			if allDead {
				skipped[nodeID] = true
			}
		}
		if skipped[nodeID] {
			res.Logs = append(res.Logs, workflows.StepLog{
				NodeID: node.ID, NodeType: node.Type, Label: node.Label, Status: "skipped",
			})
			continue
		}

		// Honor a cancelled/timed-out request context before running the node.
		if err := ctx.Err(); err != nil {
			res.Err = err.Error()
			res.StatusCode = 504
			return res
		}

		start := time.Now()
		var output interface{}
		var skipTargets []string
		var err error
		responded := false

		switch node.Type {
		case "endpoint_handler":
			output = request
		case "endpoint_response":
			applyResponse(node.Config, execCtx, res)
			output = map[string]interface{}{"status": res.StatusCode}
			responded = true
		case "endpoint_data":
			output, err = x.runData(ctx, node.Config, execCtx, cc)
		default:
			// Logic, transform, integration nodes: reuse the hardened workflow
			// executor. Skip-target conventions match, including the filter
			// all-downstream sentinel.
			output, skipTargets, err = workflows.ExecNode(ctx, node, execCtx)
		}

		log := workflows.StepLog{
			NodeID: node.ID, NodeType: node.Type, Label: node.Label,
			Output: output, DurationMs: time.Since(start).Milliseconds(),
		}
		if err != nil {
			log.Status = "failed"
			log.Error = err.Error()
			res.Logs = append(res.Logs, log)
			res.Err = err.Error()
			res.StatusCode = statusForError(err)
			return res
		}
		log.Status = "completed"
		res.Logs = append(res.Logs, log)

		execCtx[node.ID] = output
		if responded {
			return res
		}

		for _, t := range skipTargets {
			if t == workflows.AllDownstreamSentinel {
				for _, succ := range adj[nodeID] {
					skipped[succ] = true
				}
				continue
			}
			skipped[t] = true
		}
	}

	// The graph ran to the end without a response node. Return whatever the last
	// node produced, defaulting to an empty 200, so an endpoint always answers.
	if !res.Responded {
		res.StatusCode = 200
		res.Body = map[string]interface{}{}
	}
	return res
}

// applyResponse shapes the HTTP response from a response node's config.
//
//	status      HTTP status code (default 200)
//	mode        "json" (default) | "text" | "html" | "error"
//	bodyField   a dot path into the execution context whose value becomes the body
//	body        a template string; for json mode it is parsed as JSON when possible
func applyResponse(config map[string]interface{}, execCtx map[string]interface{}, res *Result) {
	res.Responded = true
	res.StatusCode = intFromConfig(config, "status", 200)
	mode, _ := config["mode"].(string)
	if mode == "" {
		mode = "json"
	}

	if field, _ := config["bodyField"].(string); strings.TrimSpace(field) != "" {
		val := resolveField(field, execCtx)
		switch mode {
		case "text":
			res.IsText = true
			res.ContentType = "text/plain; charset=utf-8"
			res.Text = fmt.Sprintf("%v", val)
		case "html":
			// The value is served as HTML, so escape it: it may hold
			// caller-supplied data and would otherwise be an XSS sink.
			res.IsText = true
			res.ContentType = "text/html; charset=utf-8"
			res.Text = html.EscapeString(fmt.Sprintf("%v", val))
		default:
			res.Body = val
		}
		return
	}

	bodyTmpl, _ := config["body"].(string)

	switch mode {
	case "text":
		res.IsText = true
		res.ContentType = "text/plain; charset=utf-8"
		res.Text = resolveTemplate(bodyTmpl, execCtx)
	case "html":
		// html/template context-escapes every interpolated value, so an author
		// echoing request data into markup can't be turned into an XSS vector.
		res.IsText = true
		res.ContentType = "text/html; charset=utf-8"
		res.Text = resolveHTMLTemplate(bodyTmpl, execCtx)
	case "error":
		if res.StatusCode < 400 {
			res.StatusCode = 400
		}
		res.Body = map[string]interface{}{"error": resolveTemplate(bodyTmpl, execCtx)}
	default: // json
		resolved := resolveTemplate(bodyTmpl, execCtx)
		var parsed interface{}
		if resolved != "" && json.Unmarshal([]byte(resolved), &parsed) == nil {
			res.Body = parsed
		} else {
			res.Body = map[string]interface{}{"message": resolved}
		}
	}
}

// resolveHTMLTemplate evaluates a body template with html/template, which
// context-escapes every interpolated value. A parse or execution failure
// returns the input unchanged, matching resolveTemplate; that is safe because a
// failed execution interpolates no caller data.
func resolveHTMLTemplate(tmplStr string, data map[string]interface{}) string {
	if !strings.Contains(tmplStr, "{{") {
		return tmplStr
	}
	t, err := htmltemplate.New("").Parse(tmplStr)
	if err != nil {
		return tmplStr
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return tmplStr
	}
	return buf.String()
}

// validateInput checks a request body against an endpoint's input schema, a
// small JSON-Schema subset: a "required" list of field names and a "properties"
// map of {name: {type}}. It returns a field->problem map (empty when valid).
func validateInput(schema map[string]interface{}, body interface{}) map[string]string {
	if len(schema) == 0 {
		return nil
	}
	obj, _ := body.(map[string]interface{})
	if obj == nil {
		obj = map[string]interface{}{}
	}
	problems := map[string]string{}

	if required, ok := schema["required"].([]interface{}); ok {
		for _, f := range required {
			name, _ := f.(string)
			if name == "" {
				continue
			}
			v, present := obj[name]
			if !present || v == nil || v == "" {
				problems[name] = "is required"
			}
		}
	}

	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for name, spec := range props {
			sm, _ := spec.(map[string]interface{})
			typ, _ := sm["type"].(string)
			v, present := obj[name]
			if !present || typ == "" {
				continue
			}
			if !typeMatches(typ, v) {
				problems[name] = "must be a " + typ
			}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return problems
}

func typeMatches(typ string, v interface{}) bool {
	switch typ {
	case "string":
		_, ok := v.(string)
		return ok
	case "number":
		_, ok := v.(float64) // JSON numbers decode to float64
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "object":
		_, ok := v.(map[string]interface{})
		return ok
	case "array":
		_, ok := v.([]interface{})
		return ok
	default:
		return true
	}
}

// runData executes a data node against the databases service. The apply-rules
// toggle chooses the identity: ON (default) runs as the caller under RLS; OFF
// runs as the elevated service role.
func (x *executor) runData(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}, cc callContext) (interface{}, error) {
	if x.data == nil {
		return nil, fmt.Errorf("endpoint_data: data service not configured")
	}
	action, _ := config["action"].(string)
	databaseID := resolveTemplate(getStr(config, "databaseId"), execCtx)
	tableID := resolveTemplate(getStr(config, "tableId"), execCtx)
	if databaseID == "" || tableID == "" {
		return nil, fmt.Errorf("endpoint_data: databaseId and tableId are required")
	}

	// Apply-rules ON (default): the caller's identity, so the node runs under
	// row security exactly as a client SDK call would (anonymous when the caller
	// is unauthenticated). OFF: the trusted server identity, the same elevated
	// access a server API key gets. The databases layer grants the "users" role
	// and the _authenticated Postgres role to any non-empty user id, so an empty
	// id (as we used before) is actually LESS privileged than the caller, not
	// more. serviceIdentity is that non-empty trusted id.
	userID, roles := cc.userID, []string(nil)
	if !applyRules(config) {
		userID, roles = serviceIdentity, nil
	}

	switch action {
	case "create":
		data := resolveDataMap(config["data"], execCtx)
		perms := stringSlice(config["permissions"])
		row, err := x.data.CreateRowWithAuth(ctx, cc.projectID, databaseID, tableID, getStr(config, "rowId"), data, perms, userID, roles)
		if err != nil {
			return nil, err
		}
		return rowToMap(row), nil
	case "get":
		rowID := resolveTemplate(getStr(config, "rowId"), execCtx)
		row, err := x.data.GetRowWithAuth(ctx, rowID, tableID, databaseID, cc.projectID, userID, roles)
		if err != nil {
			return nil, err
		}
		return rowToMap(row), nil
	case "list", "query":
		params := databases.ListParams{
			Limit:   intFromConfig(config, "limit", 25),
			Offset:  intFromConfig(config, "offset", 0),
			Queries: buildQueries(config, execCtx),
		}
		rows, total, err := x.data.ListRowsWithAuth(ctx, cc.projectID, databaseID, tableID, userID, roles, params)
		if err != nil {
			return nil, err
		}
		items := make([]interface{}, 0, len(rows))
		for _, r := range rows {
			items = append(items, rowToMap(r))
		}
		return map[string]interface{}{"rows": items, "total": total}, nil
	case "update":
		rowID := resolveTemplate(getStr(config, "rowId"), execCtx)
		data := resolveDataMap(config["data"], execCtx)
		row, err := x.data.UpdateRowWithAuth(ctx, rowID, tableID, databaseID, cc.projectID, data, nil, userID, roles)
		if err != nil {
			return nil, err
		}
		return rowToMap(row), nil
	case "delete":
		rowID := resolveTemplate(getStr(config, "rowId"), execCtx)
		if err := x.data.DeleteRowWithAuth(ctx, rowID, tableID, databaseID, cc.projectID, userID, roles); err != nil {
			return nil, err
		}
		return map[string]interface{}{"deleted": true, "$id": rowID}, nil
	default:
		return nil, fmt.Errorf("endpoint_data: unknown action %q", action)
	}
}

// serviceIdentity is the user id a data node runs under when Apply-rules is OFF.
// It mirrors how a server API key is seen by the databases layer (middleware
// sets "api:<keyId>"): a non-empty id, which grants the trusted "users" role and
// the _authenticated Postgres role. It is not a raw RLS bypass — the platform
// always enforces row security — but it is the sanctioned trusted-server access,
// the same the server SDKs get.
const serviceIdentity = "api:endpoint"

// statusForError maps a node failure to an HTTP status. A permission or
// row-security denial is a 403, not a server error; everything else is a 500.
func statusForError(err error) int {
	s := err.Error()
	if strings.Contains(s, "permission denied") ||
		strings.Contains(s, "row-level security") ||
		strings.Contains(s, "42501") {
		return 403
	}
	return 500
}

// applyRules reads the per-node toggle, defaulting to ON (scoped) when absent.
func applyRules(config map[string]interface{}) bool {
	v, ok := config["applyRules"]
	if !ok {
		return true
	}
	b, ok := v.(bool)
	if !ok {
		return true
	}
	return b
}

// buildQueries turns a node's filter list into databases queries.
// filters: [{ "field": "...", "method": "equal", "value": ... }]
func buildQueries(config map[string]interface{}, execCtx map[string]interface{}) []databases.Query {
	raw, ok := config["filters"].([]interface{})
	if !ok {
		return nil
	}
	var out []databases.Query
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		field, _ := m["field"].(string)
		method, _ := m["method"].(string)
		if field == "" || method == "" {
			continue
		}
		val := m["value"]
		if s, ok := val.(string); ok {
			val = resolveTemplate(s, execCtx)
		}
		out = append(out, databases.Query{Field: field, Method: method, Values: val})
	}
	return out
}

// --- value helpers ---

func rowToMap(r *model.Row) map[string]interface{} {
	if r == nil {
		return nil
	}
	m := make(map[string]interface{}, len(r.Data)+3)
	for k, v := range r.Data {
		m[k] = v
	}
	m["$id"] = r.ID
	m["$createdAt"] = r.CreatedAt
	m["$updatedAt"] = r.UpdatedAt
	if r.Permissions != nil {
		m["$permissions"] = r.Permissions
	}
	return m
}

// resolveDataMap deep-resolves string template values in a config data map.
func resolveDataMap(v interface{}, execCtx map[string]interface{}) map[string]interface{} {
	m, ok := v.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(m))
	for k, val := range m {
		if s, ok := val.(string); ok {
			out[k] = resolveTemplate(s, execCtx)
		} else {
			out[k] = val
		}
	}
	return out
}

func stringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func getStr(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}

func intFromConfig(config map[string]interface{}, key string, def int) int {
	switch v := config[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

// resolveTemplate evaluates a Go text/template string against the execution
// context. A malformed template or a failed execution returns the input
// unchanged, matching the workflow engine's forgiving behavior.
func resolveTemplate(tmplStr string, data map[string]interface{}) string {
	if !strings.Contains(tmplStr, "{{") {
		return tmplStr
	}
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return tmplStr
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return tmplStr
	}
	return buf.String()
}

// resolveField navigates a dot-separated path in the execution context.
func resolveField(field string, data map[string]interface{}) interface{} {
	parts := strings.Split(field, ".")
	var current interface{} = data
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}
