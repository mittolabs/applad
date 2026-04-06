package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRunWorkflow_HTTPRequest_MockServer(t *testing.T) {
	// Start a test HTTP server that returns JSON.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"greeting": "hello from server"})
	}))
	defer ts.Close()

	wf := &Workflow{
		Nodes: []Node{
			{
				ID: "req1", Type: "http_request", Label: "Fetch",
				Config: map[string]interface{}{
					"method": "GET",
					"url":    ts.URL,
				},
			},
		},
	}

	logs, err := RunWorkflow(context.Background(), wf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Status != "completed" {
		t.Fatalf("expected completed, got %s (error: %s)", logs[0].Status, logs[0].Error)
	}

	output, ok := logs[0].Output.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map output, got %T", logs[0].Output)
	}
	if sc, ok := output["statusCode"].(int); !ok || sc != 200 {
		t.Errorf("expected statusCode 200, got %v", output["statusCode"])
	}
	body, ok := output["body"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected body to be a map, got %T", output["body"])
	}
	if body["greeting"] != "hello from server" {
		t.Errorf("expected greeting 'hello from server', got %v", body["greeting"])
	}
}

func TestRunWorkflow_SetVariable_Propagates(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{
				ID: "setX", Type: "set_variable", Label: "Set X",
				Config: map[string]interface{}{"key": "x", "value": "hello"},
			},
			{
				ID: "useX", Type: "code", Label: "Use X",
				Config: map[string]interface{}{"expression": "value is {{.x}}"},
			},
		},
		Edges: []Edge{{ID: "e1", Source: "setX", Target: "useX"}},
	}

	logs, err := RunWorkflow(context.Background(), wf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}

	// The code node should see "x" in the execution context.
	output := logs[1].Output.(map[string]interface{})
	result, _ := output["result"].(string)
	if !strings.Contains(result, "hello") {
		t.Errorf("expected code output to contain 'hello', got %q", result)
	}
}

func TestRunWorkflow_IfCondition_FalseBranch(t *testing.T) {
	// Condition evaluates false, so trueBranch should be skipped.
	wf := &Workflow{
		Nodes: []Node{
			{
				ID: "cond", Type: "if_condition", Label: "Check",
				Config: map[string]interface{}{
					"field": "trigger.status", "operator": "eq", "value": "active",
					"trueBranch": "yes", "falseBranch": "no",
				},
			},
			{
				ID: "yes", Type: "set_variable", Label: "True Branch",
				Config: map[string]interface{}{"key": "result", "value": "matched"},
			},
			{
				ID: "no", Type: "set_variable", Label: "False Branch",
				Config: map[string]interface{}{"key": "result", "value": "not matched"},
			},
		},
		Edges: []Edge{
			{Source: "cond", Target: "yes"},
			{Source: "cond", Target: "no"},
		},
	}

	// trigger.status = "inactive" so condition is false.
	triggerData := map[string]interface{}{"status": "inactive"}
	logs, err := RunWorkflow(context.Background(), wf, triggerData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}

	for _, l := range logs {
		if l.NodeID == "yes" && l.Status != "skipped" {
			t.Errorf("expected 'yes' (true branch) to be skipped, got %s", l.Status)
		}
		if l.NodeID == "no" && l.Status != "completed" {
			t.Errorf("expected 'no' (false branch) to complete, got %s", l.Status)
		}
	}
}

func TestRunWorkflow_MultipleConditions(t *testing.T) {
	// cond1: trigger.a == "x" -> true -> cond2
	// cond2: trigger.b == "y" -> true -> action, false -> fallback
	wf := &Workflow{
		Nodes: []Node{
			{
				ID: "cond1", Type: "if_condition", Label: "Check A",
				Config: map[string]interface{}{
					"field": "trigger.a", "operator": "eq", "value": "x",
					"trueBranch": "cond2", "falseBranch": "skip1",
				},
			},
			{
				ID: "cond2", Type: "if_condition", Label: "Check B",
				Config: map[string]interface{}{
					"field": "trigger.b", "operator": "eq", "value": "y",
					"trueBranch": "action", "falseBranch": "fallback",
				},
			},
			{
				ID: "skip1", Type: "set_variable", Label: "Skipped",
				Config: map[string]interface{}{"key": "out", "value": "should not run"},
			},
			{
				ID: "action", Type: "set_variable", Label: "Action",
				Config: map[string]interface{}{"key": "out", "value": "both matched"},
			},
			{
				ID: "fallback", Type: "set_variable", Label: "Fallback",
				Config: map[string]interface{}{"key": "out", "value": "b did not match"},
			},
		},
		Edges: []Edge{
			{Source: "cond1", Target: "cond2"},
			{Source: "cond1", Target: "skip1"},
			{Source: "cond2", Target: "action"},
			{Source: "cond2", Target: "fallback"},
		},
	}

	triggerData := map[string]interface{}{"a": "x", "b": "y"}
	logs, err := RunWorkflow(context.Background(), wf, triggerData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statusMap := map[string]string{}
	for _, l := range logs {
		statusMap[l.NodeID] = l.Status
	}

	if statusMap["cond1"] != "completed" {
		t.Errorf("expected cond1 completed, got %s", statusMap["cond1"])
	}
	if statusMap["cond2"] != "completed" {
		t.Errorf("expected cond2 completed, got %s", statusMap["cond2"])
	}
	if statusMap["skip1"] != "skipped" {
		t.Errorf("expected skip1 skipped, got %s", statusMap["skip1"])
	}
	if statusMap["action"] != "completed" {
		t.Errorf("expected action completed, got %s", statusMap["action"])
	}
	if statusMap["fallback"] != "skipped" {
		t.Errorf("expected fallback skipped, got %s", statusMap["fallback"])
	}
}

func TestRunWorkflow_FailingNode_StopsExecution(t *testing.T) {
	// http_request to a non-routable URL that will fail.
	wf := &Workflow{
		Nodes: []Node{
			{
				ID: "bad", Type: "http_request", Label: "Bad Request",
				Config: map[string]interface{}{
					"method": "GET",
					"url":    "http://192.0.2.1:1/nonexistent", // RFC 5737 TEST-NET, will fail
				},
			},
			{
				ID: "after", Type: "set_variable", Label: "Should Not Run",
				Config: map[string]interface{}{"key": "x", "value": "nope"},
			},
		},
		Edges: []Edge{{Source: "bad", Target: "after"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	logs, err := RunWorkflow(ctx, wf, nil)
	if err == nil {
		t.Fatal("expected error from failing node, got nil")
	}

	// Only the failing node should appear in logs; "after" should not.
	if len(logs) != 1 {
		t.Errorf("expected 1 log (only the failing node), got %d", len(logs))
	}
	if logs[0].NodeID != "bad" {
		t.Errorf("expected failing node 'bad', got %q", logs[0].NodeID)
	}
	if logs[0].Status != "failed" {
		t.Errorf("expected status 'failed', got %q", logs[0].Status)
	}
}

func TestRunWorkflow_DelayCancellation(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{
				ID: "longdelay", Type: "delay", Label: "Long Delay",
				Config: map[string]interface{}{
					"durationMs": float64(60000), // 60 seconds
				},
			},
		},
	}

	// Very short timeout so the delay is cancelled.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := RunWorkflow(ctx, wf, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("expected context-related error, got: %v", err)
	}
}

func TestRunWorkflow_ComplexDAG(t *testing.T) {
	// DAG shape:
	//          start
	//         /     \
	//      left    right
	//         \     /
	//         merge
	//           |
	//          end
	wf := &Workflow{
		Nodes: []Node{
			{ID: "start", Type: "set_variable", Label: "Start",
				Config: map[string]interface{}{"key": "started", "value": "yes"}},
			{ID: "left", Type: "set_variable", Label: "Left",
				Config: map[string]interface{}{"key": "left_done", "value": "true"}},
			{ID: "right", Type: "set_variable", Label: "Right",
				Config: map[string]interface{}{"key": "right_done", "value": "true"}},
			{ID: "merge", Type: "code", Label: "Merge",
				Config: map[string]interface{}{"expression": "left={{.left_done}} right={{.right_done}}"}},
			{ID: "end", Type: "set_variable", Label: "End",
				Config: map[string]interface{}{"key": "final", "value": "done"}},
		},
		Edges: []Edge{
			{Source: "start", Target: "left"},
			{Source: "start", Target: "right"},
			{Source: "left", Target: "merge"},
			{Source: "right", Target: "merge"},
			{Source: "merge", Target: "end"},
		},
	}

	logs, err := RunWorkflow(context.Background(), wf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 5 {
		t.Fatalf("expected 5 logs, got %d", len(logs))
	}

	// All should be completed.
	for _, l := range logs {
		if l.Status != "completed" {
			t.Errorf("node %s: expected completed, got %s", l.NodeID, l.Status)
		}
	}

	// Verify topological order: start must come before left/right, merge after both, end last.
	orderIndex := map[string]int{}
	for i, l := range logs {
		orderIndex[l.NodeID] = i
	}
	if orderIndex["start"] >= orderIndex["left"] {
		t.Error("start should come before left")
	}
	if orderIndex["start"] >= orderIndex["right"] {
		t.Error("start should come before right")
	}
	if orderIndex["left"] >= orderIndex["merge"] {
		t.Error("left should come before merge")
	}
	if orderIndex["right"] >= orderIndex["merge"] {
		t.Error("right should come before merge")
	}
	if orderIndex["merge"] >= orderIndex["end"] {
		t.Error("merge should come before end")
	}

	// Verify merge node saw the upstream variables.
	mergeOutput := logs[orderIndex["merge"]].Output.(map[string]interface{})
	result, _ := mergeOutput["result"].(string)
	if !strings.Contains(result, "left=true") || !strings.Contains(result, "right=true") {
		t.Errorf("merge node should see both upstream values, got %q", result)
	}
}

func TestEvaluateCondition_AllOperators(t *testing.T) {
	tests := []struct {
		name     string
		actual   interface{}
		op       string
		expected interface{}
		want     bool
	}{
		// eq / == / equals
		{"eq match", "hello", "eq", "hello", true},
		{"eq mismatch", "hello", "eq", "world", false},
		{"== alias", "42", "==", "42", true},
		{"equals alias", "abc", "equals", "abc", true},

		// neq / != / not_equals
		{"neq match", "a", "neq", "b", true},
		{"neq same", "a", "neq", "a", false},
		{"!= alias", "x", "!=", "y", true},
		{"not_equals alias", "x", "not_equals", "x", false},

		// contains
		{"contains yes", "hello world", "contains", "world", true},
		{"contains no", "hello world", "contains", "xyz", false},
		{"contains empty substr", "hello", "contains", "", true},

		// not_contains
		{"not_contains yes", "hello", "not_contains", "xyz", true},
		{"not_contains no", "hello", "not_contains", "ell", false},

		// starts_with
		{"starts_with yes", "hello world", "starts_with", "hello", true},
		{"starts_with no", "hello world", "starts_with", "world", false},
		{"starts_with empty", "hello", "starts_with", "", true},

		// ends_with
		{"ends_with yes", "hello world", "ends_with", "world", true},
		{"ends_with no", "hello world", "ends_with", "hello", false},
		{"ends_with empty", "hello", "ends_with", "", true},

		// empty
		{"empty string", "", "empty", "", true},
		{"empty nil", nil, "empty", "", true},
		{"empty non-empty", "hello", "empty", "", false},

		// not_empty
		{"not_empty has value", "hello", "not_empty", "", true},
		{"not_empty empty string", "", "not_empty", "", false},
		{"not_empty nil", nil, "not_empty", "", false},

		// edge cases: numeric strings
		{"eq numeric", 42, "eq", "42", true},
		{"contains numeric", 12345, "contains", "234", true},

		// edge case: nil compared
		{"nil eq nil", nil, "eq", nil, true},
		{"nil neq string", nil, "neq", "hello", true},

		// default operator falls through to eq
		{"unknown op match", "abc", "unknown_op", "abc", true},
		{"unknown op mismatch", "abc", "unknown_op", "xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateCondition(tt.actual, tt.op, tt.expected)
			if got != tt.want {
				t.Errorf("evaluateCondition(%v, %q, %v) = %v, want %v",
					tt.actual, tt.op, tt.expected, got, tt.want)
			}
		})
	}
}

func TestRunWorkflow_HTTPRequest_PostWithBody(t *testing.T) {
	var receivedBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"created": true}`)
	}))
	defer ts.Close()

	wf := &Workflow{
		Nodes: []Node{
			{
				ID: "post1", Type: "http_request", Label: "Post Data",
				Config: map[string]interface{}{
					"method": "POST",
					"url":    ts.URL,
					"body":   `{"name": "{{.trigger.name}}"}`,
				},
			},
		},
	}

	triggerData := map[string]interface{}{"name": "Alice"}
	logs, err := RunWorkflow(context.Background(), wf, triggerData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logs[0].Status != "completed" {
		t.Fatalf("expected completed, got %s: %s", logs[0].Status, logs[0].Error)
	}
	if !strings.Contains(receivedBody, "Alice") {
		t.Errorf("expected request body to contain 'Alice', got %q", receivedBody)
	}
}

func TestRunWorkflow_HTTPRequest_CustomHeaders(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer ts.Close()

	wf := &Workflow{
		Nodes: []Node{
			{
				ID: "h1", Type: "http_request", Label: "With Auth",
				Config: map[string]interface{}{
					"url": ts.URL,
					"headers": map[string]interface{}{
						"Authorization": "Bearer test-token",
					},
				},
			},
		},
	}

	_, err := RunWorkflow(context.Background(), wf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("expected Authorization header 'Bearer test-token', got %q", gotAuth)
	}
}

func TestRunWorkflow_UnknownNodeType(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{ID: "u1", Type: "nonexistent_type", Label: "Unknown"},
		},
	}

	logs, err := RunWorkflow(context.Background(), wf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unknown node type should still complete (returns a message, no error).
	if len(logs) != 1 || logs[0].Status != "completed" {
		t.Errorf("expected 1 completed log, got %v", logs)
	}
	output := logs[0].Output.(map[string]interface{})
	msg, _ := output["message"].(string)
	if !strings.Contains(msg, "unknown node type") {
		t.Errorf("expected unknown node type message, got %q", msg)
	}
}
