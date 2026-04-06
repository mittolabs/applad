package workflows

import (
	"context"
	"testing"
)

func TestTopoSort_Linear(t *testing.T) {
	nodes := []Node{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}
	edges := []Edge{
		{ID: "e1", Source: "a", Target: "b"},
		{ID: "e2", Source: "b", Target: "c"},
	}
	order := topoSort(nodes, edges)
	if len(order) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(order))
	}
	if order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Errorf("expected [a b c], got %v", order)
	}
}

func TestTopoSort_Diamond(t *testing.T) {
	nodes := []Node{
		{ID: "start"}, {ID: "left"}, {ID: "right"}, {ID: "end"},
	}
	edges := []Edge{
		{Source: "start", Target: "left"},
		{Source: "start", Target: "right"},
		{Source: "left", Target: "end"},
		{Source: "right", Target: "end"},
	}
	order := topoSort(nodes, edges)
	if len(order) != 4 {
		t.Fatalf("expected 4, got %d", len(order))
	}
	if order[0] != "start" {
		t.Errorf("expected start first, got %s", order[0])
	}
	if order[3] != "end" {
		t.Errorf("expected end last, got %s", order[3])
	}
}

func TestTopoSort_NoEdges(t *testing.T) {
	nodes := []Node{{ID: "a"}, {ID: "b"}}
	order := topoSort(nodes, nil)
	if len(order) != 2 {
		t.Fatalf("expected 2, got %d", len(order))
	}
}

func TestResolveTemplate(t *testing.T) {
	data := map[string]interface{}{
		"trigger": map[string]interface{}{
			"name": "Alice",
		},
	}
	result := resolveTemplate("Hello {{.trigger.name}}", data)
	if result != "Hello Alice" {
		t.Errorf("expected 'Hello Alice', got %q", result)
	}
}

func TestResolveTemplate_NoTemplate(t *testing.T) {
	result := resolveTemplate("plain string", nil)
	if result != "plain string" {
		t.Errorf("expected 'plain string', got %q", result)
	}
}

func TestResolveFieldValue(t *testing.T) {
	data := map[string]interface{}{
		"trigger": map[string]interface{}{
			"status": "active",
		},
	}
	val := resolveFieldValue("trigger.status", data)
	if val != "active" {
		t.Errorf("expected 'active', got %v", val)
	}
}

func TestResolveFieldValue_Missing(t *testing.T) {
	data := map[string]interface{}{}
	val := resolveFieldValue("missing.field", data)
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		actual   interface{}
		op       string
		expected interface{}
		want     bool
	}{
		{"hello", "eq", "hello", true},
		{"hello", "eq", "world", false},
		{"hello", "neq", "world", true},
		{"hello world", "contains", "world", true},
		{"hello world", "not_contains", "xyz", true},
		{"hello", "starts_with", "hel", true},
		{"hello", "ends_with", "llo", true},
		{"", "empty", "", true},
		{"hello", "not_empty", "", true},
		{nil, "empty", "", true},
	}

	for _, tt := range tests {
		got := evaluateCondition(tt.actual, tt.op, tt.expected)
		if got != tt.want {
			t.Errorf("evaluateCondition(%v, %q, %v) = %v, want %v",
				tt.actual, tt.op, tt.expected, got, tt.want)
		}
	}
}

func TestRunWorkflow_EmptyNodes(t *testing.T) {
	wf := &Workflow{Nodes: []Node{}, Edges: []Edge{}}
	logs, err := RunWorkflow(context.Background(), wf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

func TestRunWorkflow_SetVariable(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{ID: "n1", Type: "set_variable", Label: "Set Name", Config: map[string]interface{}{
				"key": "myVar", "value": "hello",
			}},
		},
		Edges: []Edge{},
	}
	logs, err := RunWorkflow(context.Background(), wf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Status != "completed" {
		t.Errorf("expected completed, got %s", logs[0].Status)
	}
}

func TestRunWorkflow_CodeNode(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{ID: "n1", Type: "code", Label: "Eval", Config: map[string]interface{}{
				"expression": "Result: {{.trigger.x}}",
			}},
		},
		Edges: []Edge{},
	}
	triggerData := map[string]interface{}{"x": "42"}
	logs, err := RunWorkflow(context.Background(), wf, triggerData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 1 || logs[0].Status != "completed" {
		t.Fatalf("expected 1 completed log, got %v", logs)
	}
	output := logs[0].Output.(map[string]interface{})
	if output["result"] != "Result: 42" {
		t.Errorf("expected 'Result: 42', got %v", output["result"])
	}
}

func TestRunWorkflow_IfCondition(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{ID: "cond", Type: "if_condition", Label: "Check", Config: map[string]interface{}{
				"field": "trigger.status", "operator": "eq", "value": "active",
				"trueBranch": "yes", "falseBranch": "no",
			}},
			{ID: "yes", Type: "set_variable", Label: "Yes", Config: map[string]interface{}{
				"key": "result", "value": "matched",
			}},
			{ID: "no", Type: "set_variable", Label: "No", Config: map[string]interface{}{
				"key": "result", "value": "not matched",
			}},
		},
		Edges: []Edge{
			{Source: "cond", Target: "yes"},
			{Source: "cond", Target: "no"},
		},
	}
	triggerData := map[string]interface{}{"status": "active"}
	logs, err := RunWorkflow(context.Background(), wf, triggerData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}
	// "no" branch should be skipped
	for _, l := range logs {
		if l.NodeID == "no" && l.Status != "skipped" {
			t.Errorf("expected 'no' node to be skipped, got %s", l.Status)
		}
		if l.NodeID == "yes" && l.Status != "completed" {
			t.Errorf("expected 'yes' node to complete, got %s", l.Status)
		}
	}
}

func TestRunWorkflow_Delay(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{ID: "d", Type: "delay", Label: "Short Delay", Config: map[string]interface{}{
				"durationMs": float64(10), // 10ms
			}},
		},
	}
	logs, err := RunWorkflow(context.Background(), wf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logs[0].Status != "completed" {
		t.Errorf("expected completed, got %s", logs[0].Status)
	}
}

func TestRunWorkflow_ChainedNodes(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{ID: "a", Type: "set_variable", Label: "Set X", Config: map[string]interface{}{
				"key": "x", "value": "hello",
			}},
			{ID: "b", Type: "set_variable", Label: "Set Y", Config: map[string]interface{}{
				"key": "y", "value": "world",
			}},
		},
		Edges: []Edge{{Source: "a", Target: "b"}},
	}
	logs, err := RunWorkflow(context.Background(), wf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
	for _, l := range logs {
		if l.Status != "completed" {
			t.Errorf("node %s: expected completed, got %s", l.NodeID, l.Status)
		}
	}
}
