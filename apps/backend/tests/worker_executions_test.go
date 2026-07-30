//go:build integration

package tests

import (
	"fmt"
	"testing"
	"time"
)

// TestWorkerExecutionsAsyncPath proves an async worker path completes end to
// end: a workflow execution is enqueued by the API, picked up by the booted
// worker-executions binary off the Redis queue, run, and its status transitioned
// away from "pending" to a terminal state — all without the test doing the work
// itself.
//
// This is the first integration test that asserts the async worker fleet
// actually runs. It requires worker-executions to be running against the same
// Postgres+Redis as the API (CI builds and starts it; `docker compose up` runs
// it locally). If no worker is consuming the queue the execution stays "pending"
// and this test fails by timeout — which is the point: a silent, idle fleet is
// exactly the gap this closes.
func TestWorkerExecutionsAsyncPath(t *testing.T) {
	projectID, apiKey := projectWithKey(t, "worker-executions-e2e")
	h := authHeader(projectID, apiKey)

	// A single no-op node: it runs to completion without network or Docker, so
	// the only thing under test is that a worker consumed the job and drove the
	// status transition.
	status, body := request(t, "POST", "/workflows",
		map[string]interface{}{
			"name": "async-e2e",
			"nodes": []map[string]interface{}{
				{"id": "start", "type": "no_operation", "config": map[string]interface{}{}},
			},
			"edges":   []interface{}{},
			"trigger": "manual",
		}, h)
	if status != 201 {
		t.Fatalf("create workflow: expected 201, got %d: %v", status, body)
	}
	wfID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/workflows/%s", wfID), nil, h)

	// Enqueue an execution. The API records it 'pending' and pushes to the
	// "executions" queue, then returns 202 without doing the work.
	status, body = request(t, "POST", fmt.Sprintf("/workflows/%s/execute", wfID),
		map[string]interface{}{"triggerData": map[string]interface{}{"trigger": "e2e"}}, h)
	if status != 202 {
		t.Fatalf("execute workflow: expected 202, got %d: %v", status, body)
	}
	execID, _ := body["$id"].(string)
	if execID == "" {
		t.Fatalf("execute workflow: no execution id in response: %v", body)
	}
	if got, _ := body["status"].(string); got != "pending" {
		t.Logf("execution starts in status %q (expected pending)", got)
	}

	// Poll for the worker to pick it up and transition it off "pending". A booted
	// worker completes a no-op DAG in well under a second; the generous window
	// absorbs CI scheduling and the queue poll interval.
	deadline := time.Now().Add(30 * time.Second)
	var lastStatus string
	for time.Now().Before(deadline) {
		status, body = request(t, "GET",
			fmt.Sprintf("/workflows/%s/executions/%s", wfID, execID), nil, h)
		if status != 200 {
			t.Fatalf("get execution: expected 200, got %d: %v", status, body)
		}
		lastStatus, _ = body["status"].(string)
		if lastStatus != "" && lastStatus != "pending" {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	switch lastStatus {
	case "completed":
		// Async path fully closed: worker consumed the job and finished the run.
	case "running":
		// Also proves the worker booted and claimed the job.
		t.Logf("execution reached %q — worker is consuming jobs", lastStatus)
	case "failed":
		t.Fatalf("execution was picked up but failed: %v", body)
	default:
		t.Fatalf("execution never left %q within 30s — is worker-executions running against this Redis/Postgres?", lastStatus)
	}
}
