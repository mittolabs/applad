package endpoints

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mittolabs/applad/internal/databases"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/workflows"
)

// fakeData records the identity every call was made under, so a test can assert
// the apply-rules toggle chose the right role.
type fakeData struct {
	lastUserID string
	lastRoles  []string
	created    map[string]interface{}
	getRow     *model.Row
	listRows   []*model.Row
	err        error
}

func (f *fakeData) CreateRowWithAuth(_ context.Context, _, _, _, rowID string, data map[string]interface{}, _ []string, userID string, roles []string) (*model.Row, error) {
	f.lastUserID, f.lastRoles, f.created = userID, roles, data
	if f.err != nil {
		return nil, f.err
	}
	return &model.Row{ID: firstOr(rowID, "row_new"), Data: data}, nil
}
func (f *fakeData) GetRowWithAuth(_ context.Context, rowID, _, _, _, userID string, roles []string) (*model.Row, error) {
	f.lastUserID, f.lastRoles = userID, roles
	if f.err != nil {
		return nil, f.err
	}
	if f.getRow != nil {
		return f.getRow, nil
	}
	return &model.Row{ID: rowID, Data: map[string]interface{}{"name": "Ada"}}, nil
}
func (f *fakeData) ListRowsWithAuth(_ context.Context, _, _, _, userID string, roles []string, _ databases.ListParams) ([]*model.Row, int, error) {
	f.lastUserID, f.lastRoles = userID, roles
	return f.listRows, len(f.listRows), f.err
}
func (f *fakeData) UpdateRowWithAuth(_ context.Context, rowID, _, _, _ string, data map[string]interface{}, _ []string, userID string, roles []string) (*model.Row, error) {
	f.lastUserID, f.lastRoles = userID, roles
	return &model.Row{ID: rowID, Data: data}, f.err
}
func (f *fakeData) DeleteRowWithAuth(_ context.Context, _, _, _, _, userID string, roles []string) error {
	f.lastUserID, f.lastRoles = userID, roles
	return f.err
}

func firstOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func run(ep *Endpoint, req map[string]interface{}, cc callContext, data DataService) *Result {
	return newExecutor(data).Run(context.Background(), ep, req, cc)
}

// A minimal handler → response endpoint returns the templated JSON body.
func TestRun_HandlerToJSONResponse(t *testing.T) {
	ep := &Endpoint{
		Nodes: []workflows.Node{
			{ID: "h", Type: "endpoint_handler"},
			{ID: "r", Type: "endpoint_response", Config: map[string]interface{}{
				"status": float64(201),
				"body":   `{"hello":"{{.request.body.name}}"}`,
			}},
		},
		Edges: []workflows.Edge{{Source: "h", Target: "r"}},
	}
	req := map[string]interface{}{"body": map[string]interface{}{"name": "world"}}
	res := run(ep, req, callContext{projectID: "p1"}, &fakeData{})

	if res.Err != "" {
		t.Fatalf("unexpected error: %s", res.Err)
	}
	if res.StatusCode != 201 {
		t.Fatalf("status = %d, want 201", res.StatusCode)
	}
	body, _ := res.Body.(map[string]interface{})
	if body["hello"] != "world" {
		t.Fatalf("body = %#v, want hello=world", res.Body)
	}
}

// A data create node runs, and a response node returns its output via bodyField.
func TestRun_DataCreate_BodyField(t *testing.T) {
	fd := &fakeData{}
	ep := &Endpoint{
		Nodes: []workflows.Node{
			{ID: "h", Type: "endpoint_handler"},
			{ID: "c", Type: "endpoint_data", Config: map[string]interface{}{
				"action": "create", "databaseId": "db1", "tableId": "users",
				"data": map[string]interface{}{"email": "{{.request.body.email}}"},
			}},
			{ID: "r", Type: "endpoint_response", Config: map[string]interface{}{
				"status": float64(200), "bodyField": "c",
			}},
		},
		Edges: []workflows.Edge{{Source: "h", Target: "c"}, {Source: "c", Target: "r"}},
	}
	req := map[string]interface{}{"body": map[string]interface{}{"email": "a@b.com"}}
	res := run(ep, req, callContext{projectID: "p1", userID: "u1"}, fd)

	if res.Err != "" {
		t.Fatalf("unexpected error: %s", res.Err)
	}
	if fd.created["email"] != "a@b.com" {
		t.Fatalf("template not resolved into data: %#v", fd.created)
	}
	body, _ := res.Body.(map[string]interface{})
	if body["email"] != "a@b.com" {
		t.Fatalf("bodyField did not surface created row: %#v", res.Body)
	}
}

// The apply-rules toggle picks the caller identity (ON) or the service role (OFF).
func TestRun_ApplyRules_TogglesRole(t *testing.T) {
	base := func(applyRules interface{}) (*fakeData, *Result) {
		fd := &fakeData{}
		cfg := map[string]interface{}{"action": "get", "databaseId": "db1", "tableId": "t", "rowId": "x"}
		if applyRules != nil {
			cfg["applyRules"] = applyRules
		}
		ep := &Endpoint{
			Nodes: []workflows.Node{
				{ID: "h", Type: "endpoint_handler"},
				{ID: "g", Type: "endpoint_data", Config: cfg},
				{ID: "r", Type: "endpoint_response", Config: map[string]interface{}{"bodyField": "g"}},
			},
			Edges: []workflows.Edge{{Source: "h", Target: "g"}, {Source: "g", Target: "r"}},
		}
		return fd, run(ep, map[string]interface{}{}, callContext{projectID: "p1", userID: "caller-1"}, fd)
	}

	// Default (absent) ⇒ apply rules ON ⇒ runs as the caller.
	fd, res := base(nil)
	if res.Err != "" {
		t.Fatalf("err: %s", res.Err)
	}
	if fd.lastUserID != "caller-1" || fd.lastRoles != nil {
		t.Fatalf("apply-rules ON: got userID=%q roles=%v, want caller-1/nil", fd.lastUserID, fd.lastRoles)
	}

	// Explicit ON.
	fd, _ = base(true)
	if fd.lastUserID != "caller-1" || fd.lastRoles != nil {
		t.Fatalf("apply-rules true: got userID=%q roles=%v", fd.lastUserID, fd.lastRoles)
	}

	// OFF ⇒ the trusted server identity (a non-empty id, like an API key), which
	// the databases layer elevates to the "users" role. An empty id would be the
	// anonymous role, i.e. LESS privileged than the caller.
	fd, _ = base(false)
	if fd.lastUserID != serviceIdentity || fd.lastRoles != nil {
		t.Fatalf("apply-rules OFF: got userID=%q roles=%v, want %q/nil", fd.lastUserID, fd.lastRoles, serviceIdentity)
	}
}

func TestStatusForError(t *testing.T) {
	cases := map[string]int{
		"create row: ERROR: new row violates row-level security policy (SQLSTATE 42501)": 403,
		"permission denied for table messages":                                           403,
		"ERROR: column \"text\" does not exist (SQLSTATE 42703)":                         500,
		"endpoint_data: databaseId and tableId are required":                             500,
	}
	for msg, want := range cases {
		if got := statusForError(fmt.Errorf("%s", msg)); got != want {
			t.Errorf("statusForError(%q) = %d, want %d", msg, got, want)
		}
	}
}

// A condition node skips the not-taken branch and its downstream response.
func TestRun_ConditionBranchSkips(t *testing.T) {
	ep := &Endpoint{
		Nodes: []workflows.Node{
			{ID: "h", Type: "endpoint_handler"},
			{ID: "cond", Type: "if_condition", Config: map[string]interface{}{
				"field": "request.body.admin", "operator": "eq", "value": "true",
				"trueBranch": "yes", "falseBranch": "no",
			}},
			{ID: "yes", Type: "endpoint_response", Config: map[string]interface{}{"status": float64(200), "body": `{"role":"admin"}`}},
			{ID: "no", Type: "endpoint_response", Config: map[string]interface{}{"status": float64(403), "body": `{"role":"guest"}`}},
		},
		Edges: []workflows.Edge{
			{Source: "h", Target: "cond"},
			{Source: "cond", Target: "yes"},
			{Source: "cond", Target: "no"},
		},
	}
	// admin=true ⇒ the "no" branch is skipped, "yes" responds 200.
	res := run(ep, map[string]interface{}{"body": map[string]interface{}{"admin": "true"}}, callContext{projectID: "p1"}, &fakeData{})
	if res.StatusCode != 200 {
		t.Fatalf("admin path: status = %d, want 200 (logs: %+v)", res.StatusCode, res.Logs)
	}

	// admin=false ⇒ the "yes" branch is skipped, "no" responds 403.
	res = run(ep, map[string]interface{}{"body": map[string]interface{}{"admin": "false"}}, callContext{projectID: "p1"}, &fakeData{})
	if res.StatusCode != 403 {
		t.Fatalf("guest path: status = %d, want 403", res.StatusCode)
	}
}

// A failing data node surfaces as a 500 with the error captured in the trace.
func TestRun_NodeFailure_Is500(t *testing.T) {
	fd := &fakeData{err: context.DeadlineExceeded}
	ep := &Endpoint{
		Nodes: []workflows.Node{
			{ID: "h", Type: "endpoint_handler"},
			{ID: "g", Type: "endpoint_data", Config: map[string]interface{}{"action": "get", "databaseId": "d", "tableId": "t", "rowId": "1"}},
		},
		Edges: []workflows.Edge{{Source: "h", Target: "g"}},
	}
	res := run(ep, map[string]interface{}{}, callContext{projectID: "p1", userID: "u"}, fd)
	if res.Err == "" || res.StatusCode != 500 {
		t.Fatalf("want 500 with error, got status=%d err=%q", res.StatusCode, res.Err)
	}
}

func TestValidateInput(t *testing.T) {
	schema := map[string]interface{}{
		"required":   []interface{}{"email"},
		"properties": map[string]interface{}{"email": map[string]interface{}{"type": "string"}, "age": map[string]interface{}{"type": "number"}},
	}
	// Missing required field.
	if p := validateInput(schema, map[string]interface{}{}); p["email"] == "" {
		t.Errorf("missing email should be flagged, got %#v", p)
	}
	// Wrong type.
	if p := validateInput(schema, map[string]interface{}{"email": "a@b.com", "age": "not-a-number"}); p["age"] == "" {
		t.Errorf("string age should be flagged, got %#v", p)
	}
	// Valid.
	if p := validateInput(schema, map[string]interface{}{"email": "a@b.com", "age": float64(30)}); p != nil {
		t.Errorf("valid body should pass, got %#v", p)
	}
	// No schema ⇒ always valid.
	if p := validateInput(nil, "anything"); p != nil {
		t.Errorf("no schema should pass, got %#v", p)
	}
}

// An endpoint with an input schema rejects a bad body with a 400 before running.
func TestRun_InputSchema_Rejects(t *testing.T) {
	ep := &Endpoint{
		InputSchema: map[string]interface{}{"required": []interface{}{"email"}},
		Nodes: []workflows.Node{
			{ID: "h", Type: "endpoint_handler"},
			{ID: "r", Type: "endpoint_response", Config: map[string]interface{}{"status": float64(200), "body": `{"ok":true}`}},
		},
		Edges: []workflows.Edge{{Source: "h", Target: "r"}},
	}
	res := run(ep, map[string]interface{}{"body": map[string]interface{}{}}, callContext{projectID: "p1"}, &fakeData{})
	if res.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
}

// HTML response mode escapes caller-supplied data (XSS defence).
func TestApplyResponse_HTMLEscapes(t *testing.T) {
	res := &Result{}
	execCtx := map[string]interface{}{
		"request": map[string]interface{}{"body": map[string]interface{}{"name": "<script>alert(1)</script>"}},
	}
	applyResponse(map[string]interface{}{
		"mode": "html", "body": "<h1>Hi {{.request.body.name}}</h1>",
	}, execCtx, res)

	if !res.IsText || res.ContentType != "text/html; charset=utf-8" {
		t.Fatalf("html mode: IsText=%v ct=%q", res.IsText, res.ContentType)
	}
	if strings.Contains(res.Text, "<script>") {
		t.Fatalf("html mode did not escape caller input: %q", res.Text)
	}
	if !strings.Contains(res.Text, "&lt;script&gt;") {
		t.Fatalf("expected escaped script tag, got %q", res.Text)
	}
}

// Text response mode is served as text/plain, not text/html.
func TestApplyResponse_TextIsPlain(t *testing.T) {
	res := &Result{}
	applyResponse(map[string]interface{}{"mode": "text", "body": "hello"}, map[string]interface{}{}, res)
	if res.ContentType != "text/plain; charset=utf-8" {
		t.Fatalf("text mode content-type = %q, want text/plain", res.ContentType)
	}
}

func TestAuthorized(t *testing.T) {
	cases := []struct {
		auth   string
		userID string
		isKey  bool
		want   bool
	}{
		{AuthPublic, "", false, true},
		{AuthPublic, "u", true, true},
		{AuthSession, "", false, false},
		{AuthSession, "u", false, true},
		{AuthAPIKey, "u", false, false},
		{AuthAPIKey, "", true, true},
		{AuthEither, "", false, false},
		{AuthEither, "u", false, true},
		{AuthEither, "", true, true},
	}
	for _, c := range cases {
		if got := authorized(c.auth, c.userID, c.isKey); got != c.want {
			t.Errorf("authorized(%q, user=%q, key=%v) = %v, want %v", c.auth, c.userID, c.isKey, got, c.want)
		}
	}
}
