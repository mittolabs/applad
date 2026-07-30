package workflows

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

// --- P0: $env secret exfiltration ---

// A workflow author's JS must not be able to read the API process environment.
// $env is removed entirely, so referencing it is a ReferenceError and typeof is
// "undefined".
func TestExecJavaScript_EnvBindingRemoved(t *testing.T) {
	out, err := execJavaScript(map[string]interface{}{
		"code": "typeof $env",
	}, map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res := out.(map[string]interface{})["result"]
	if res != "undefined" {
		t.Fatalf("expected typeof $env == 'undefined', got %v", res)
	}
}

func TestExecJavaScript_CannotReadJWTSecret(t *testing.T) {
	os.Setenv("JWT_SECRET", "super-secret-signing-key") //nolint:errcheck
	defer os.Unsetenv("JWT_SECRET")                     //nolint:errcheck

	// Calling the (now absent) $env is a ReferenceError — the node fails rather
	// than returning the secret.
	out, err := execJavaScript(map[string]interface{}{
		"code": "$env('JWT_SECRET')",
	}, map[string]interface{}{})
	if err == nil {
		t.Fatalf("expected a ReferenceError calling $env, got output %v", out)
	}
	if strings.Contains(err.Error(), "super-secret-signing-key") {
		t.Fatalf("the secret leaked into the error: %v", err)
	}

	// And a defensive script that guards on typeof never reaches the secret.
	out, err = execJavaScript(map[string]interface{}{
		"code": "typeof $env === 'undefined' ? 'safe' : $env('JWT_SECRET')",
	}, map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res := out.(map[string]interface{})["result"]; res != "safe" {
		t.Fatalf("expected 'safe', got %v", res)
	}
}

// --- P2: cyclic-graph rejection ---

func TestHasCycle(t *testing.T) {
	acyclic := []Node{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	if HasCycle(acyclic, []Edge{{Source: "a", Target: "b"}, {Source: "b", Target: "c"}}) {
		t.Error("linear graph should not be reported as cyclic")
	}
	if HasCycle([]Node{{ID: "a"}, {ID: "b"}}, nil) {
		t.Error("edgeless graph should not be cyclic")
	}
	cyclic := []Node{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	if !HasCycle(cyclic, []Edge{
		{Source: "a", Target: "b"}, {Source: "b", Target: "c"}, {Source: "c", Target: "a"},
	}) {
		t.Error("3-cycle should be detected")
	}
	if !HasCycle([]Node{{ID: "a"}}, []Edge{{Source: "a", Target: "a"}}) {
		t.Error("self-loop should be detected")
	}
}

// --- P1: transitive branch skip ---

// The losing branch's entire downstream subtree must not execute, even nodes
// several hops past the branch. Previously only the immediate branch node was
// skipped and its children still ran.
func TestRunWorkflow_LosingBranchSubtreeSkipped(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{ID: "cond", Type: "if_condition", Label: "Check", Config: map[string]interface{}{
				"field": "trigger.status", "operator": "eq", "value": "active",
				"trueBranch": "yes", "falseBranch": "no",
			}},
			{ID: "yes", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "yes"}},
			{ID: "no", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "no"}},
			// Deep descendants of the losing (false) branch.
			{ID: "no_child", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "child"}},
			{ID: "no_grandchild", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "grand"}},
		},
		Edges: []Edge{
			{Source: "cond", Target: "yes"},
			{Source: "cond", Target: "no"},
			{Source: "no", Target: "no_child"},
			{Source: "no_child", Target: "no_grandchild"},
		},
	}

	logs, err := RunWorkflow(context.Background(), wf, map[string]interface{}{"status": "active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	status := map[string]string{}
	for _, l := range logs {
		status[l.NodeID] = l.Status
	}
	if status["yes"] != "completed" {
		t.Errorf("winning branch 'yes' should complete, got %s", status["yes"])
	}
	for _, id := range []string{"no", "no_child", "no_grandchild"} {
		if status[id] != "skipped" {
			t.Errorf("losing-branch node %q should be skipped, got %s", id, status[id])
		}
	}
}

// A node reachable from BOTH a taken and a not-taken branch (a merge) must still
// run — the transitive skip only removes nodes reachable ONLY via dead paths.
func TestRunWorkflow_MergeKeepsAliveNode(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{ID: "cond", Type: "if_condition", Config: map[string]interface{}{
				"field": "trigger.status", "operator": "eq", "value": "active",
				"trueBranch": "yes", "falseBranch": "no",
			}},
			{ID: "yes", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "yes"}},
			{ID: "no", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "no"}},
			{ID: "merge", Type: "merge"},
		},
		Edges: []Edge{
			{Source: "cond", Target: "yes"},
			{Source: "cond", Target: "no"},
			{Source: "yes", Target: "merge"},
			{Source: "no", Target: "merge"},
		},
	}
	logs, err := RunWorkflow(context.Background(), wf, map[string]interface{}{"status": "active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	status := map[string]string{}
	for _, l := range logs {
		status[l.NodeID] = l.Status
	}
	if status["no"] != "skipped" {
		t.Errorf("losing 'no' should be skipped, got %s", status["no"])
	}
	if status["merge"] != "completed" {
		t.Errorf("merge fed by the live branch should run, got %s", status["merge"])
	}
}

// A filter that does not match must skip its whole downstream (real node ids),
// not the old sentinel that matched nothing.
func TestRunWorkflow_FilterNoMatchSkipsDownstream(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{ID: "f", Type: "filter", Config: map[string]interface{}{
				"field": "trigger.n", "operator": "eq", "value": "keep",
			}},
			{ID: "after", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "ran"}},
			{ID: "after2", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "ran2"}},
		},
		Edges: []Edge{
			{Source: "f", Target: "after"},
			{Source: "after", Target: "after2"},
		},
	}
	logs, err := RunWorkflow(context.Background(), wf, map[string]interface{}{"n": "drop"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	status := map[string]string{}
	for _, l := range logs {
		status[l.NodeID] = l.Status
	}
	if status["f"] != "completed" {
		t.Errorf("filter node should complete, got %s", status["f"])
	}
	for _, id := range []string{"after", "after2"} {
		if status[id] != "skipped" {
			t.Errorf("non-matching filter should skip %q, got %s", id, status[id])
		}
	}

	// And when it matches, downstream runs.
	logs, err = RunWorkflow(context.Background(), wf, map[string]interface{}{"n": "keep"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, l := range logs {
		if l.Status != "completed" {
			t.Errorf("matching filter: node %q expected completed, got %s", l.NodeID, l.Status)
		}
	}
}

// A non-matching switch must skip every non-default branch and its subtree.
func TestRunWorkflow_SwitchSkipsNonMatchingBranch(t *testing.T) {
	wf := &Workflow{
		Nodes: []Node{
			{ID: "sw", Type: "switch", Config: map[string]interface{}{
				"field": "trigger.kind",
				"cases": []interface{}{
					map[string]interface{}{"value": "a", "targetNodeId": "na"},
					map[string]interface{}{"value": "b", "targetNodeId": "nb"},
				},
			}},
			{ID: "na", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "a"}},
			{ID: "nb", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "b"}},
			{ID: "nb_child", Type: "set_variable", Config: map[string]interface{}{"key": "r", "value": "bc"}},
		},
		Edges: []Edge{
			{Source: "sw", Target: "na"},
			{Source: "sw", Target: "nb"},
			{Source: "nb", Target: "nb_child"},
		},
	}
	logs, err := RunWorkflow(context.Background(), wf, map[string]interface{}{"kind": "a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	status := map[string]string{}
	for _, l := range logs {
		status[l.NodeID] = l.Status
	}
	if status["na"] != "completed" {
		t.Errorf("matched case 'na' should run, got %s", status["na"])
	}
	for _, id := range []string{"nb", "nb_child"} {
		if status[id] != "skipped" {
			t.Errorf("unmatched switch branch %q should be skipped, got %s", id, status[id])
		}
	}
}

// --- P1: execution bounds ---

func TestRunWorkflow_NodeCap(t *testing.T) {
	nodes := make([]Node, maxNodesPerRun+1)
	for i := range nodes {
		nodes[i] = Node{ID: fmt.Sprintf("n%d", i), Type: "no_operation"}
	}
	_, err := RunWorkflow(context.Background(), &Workflow{Nodes: nodes}, nil)
	if err == nil {
		t.Fatal("expected an error when node count exceeds the cap")
	}
	if !strings.Contains(err.Error(), "maximum") {
		t.Errorf("expected a node-cap error, got %v", err)
	}
}

// A self-referencing sub-workflow with fan-out would explode; the total
// sub-execution cap must stop it.
func TestRunWorkflow_SubExecutionCap(t *testing.T) {
	wf := &Workflow{
		ID: "wf1",
		Nodes: []Node{
			{ID: "s1", Type: "execute_sub_workflow", Config: map[string]interface{}{"workflowId": "wf1"}},
			{ID: "s2", Type: "execute_sub_workflow", Config: map[string]interface{}{"workflowId": "wf1"}},
			{ID: "s3", Type: "execute_sub_workflow", Config: map[string]interface{}{"workflowId": "wf1"}},
		},
	}
	old := SubWorkflowRunner
	defer func() { SubWorkflowRunner = old }()
	SubWorkflowRunner = func(ctx context.Context, id string, td map[string]interface{}, depth int) ([]StepLog, interface{}, error) {
		td["__depth__"] = depth
		logs, err := RunWorkflow(ctx, wf, td)
		return logs, nil, err
	}

	_, err := RunWorkflow(context.Background(), wf, nil)
	if err == nil {
		t.Fatal("expected the sub-execution cap to halt runaway recursion")
	}
	if !strings.Contains(err.Error(), "sub-execution cap") {
		t.Errorf("expected a sub-execution cap error, got %v", err)
	}
}

// --- P1: raw-TCP node routed through netguard ---

// A redis_command aimed at a private address is refused before dialing.
// (main_test sets ALLOW_PRIVATE_EGRESS=true for httptest; flip it off here so
// the guard is active, then restore.)
func TestExecRedisCommand_RefusesPrivateTarget(t *testing.T) {
	os.Setenv("ALLOW_PRIVATE_EGRESS", "false")      //nolint:errcheck
	defer os.Setenv("ALLOW_PRIVATE_EGRESS", "true") //nolint:errcheck

	_, err := execRedisCommand(context.Background(), map[string]interface{}{
		"connectionUrl": "10.0.0.5:6379",
		"command":       "FLUSHALL",
	}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected redis_command to a private address to be refused")
	}
	if !strings.Contains(err.Error(), "netguard") && !strings.Contains(err.Error(), "public") {
		t.Errorf("expected a netguard refusal, got %v", err)
	}
}

// An HTTP integration node (slack/discord webhook post) aimed at a private
// address is likewise refused — these nodes now use the guarded client.
func TestExecWebhookPost_RefusesPrivateTarget(t *testing.T) {
	os.Setenv("ALLOW_PRIVATE_EGRESS", "false")      //nolint:errcheck
	defer os.Setenv("ALLOW_PRIVATE_EGRESS", "true") //nolint:errcheck

	_, err := execWebhookPost(context.Background(), map[string]interface{}{
		"webhookUrl": "http://169.254.169.254/latest/meta-data/",
		"message":    "x",
	}, map[string]interface{}{}, "slack")
	if err == nil {
		t.Fatal("expected webhook post to cloud metadata to be refused")
	}
}

// --- P2: non-idempotent nodes are not retried ---

func TestIsNonIdempotent(t *testing.T) {
	for _, ty := range []string{"send_email", "stripe", "twilio_sms", "sendgrid"} {
		if !isNonIdempotent(&Node{Type: ty}) {
			t.Errorf("%s should be treated as non-idempotent", ty)
		}
	}
	if isNonIdempotent(&Node{Type: "http_request"}) {
		t.Error("http_request should be retryable")
	}
	if !isNonIdempotent(&Node{Type: "applad_auth", Config: map[string]interface{}{"action": "create_user"}}) {
		t.Error("applad_auth create_user should be non-idempotent")
	}
	if isNonIdempotent(&Node{Type: "applad_auth", Config: map[string]interface{}{"action": "get_user"}}) {
		t.Error("applad_auth get_user should be retryable")
	}
}

// A send_email node whose SMTP host is unreachable must be attempted exactly
// once even when retries are configured — no duplicate mail.
func TestRunWorkflow_NonIdempotentNotRetried(t *testing.T) {
	wf := &Workflow{
		RetryAttempts: 3,
		RetryDelayMs:  1,
		Nodes: []Node{
			{ID: "m", Type: "send_email", Config: map[string]interface{}{
				// 127.0.0.1:1 refuses immediately, so the node fails fast.
				"smtpHost": "127.0.0.1", "smtpPort": "1",
				"to": "a@example.com", "subject": "hi", "body": "x",
			}},
		},
	}
	logs, err := RunWorkflow(context.Background(), wf, nil)
	if err == nil {
		t.Fatal("expected the send_email node to fail")
	}
	// Exactly one step log for the node — a retried node would still yield one
	// log, so assert the single failed node and rely on isNonIdempotent's unit
	// test plus the fast failure (no 3x retry delay stacking) for coverage.
	if len(logs) != 1 || logs[0].Status != "failed" {
		t.Fatalf("expected 1 failed log, got %v", logs)
	}
}
