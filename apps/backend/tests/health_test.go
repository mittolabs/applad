//go:build integration

package tests

import "testing"

func TestHealthEndpoints(t *testing.T) {
	status, body := request(t, "GET", "/health", nil, nil)
	if status != 200 {
		t.Fatalf("expected 200, got %d", status)
	}
	if body["status"] != "pass" {
		t.Fatalf("expected status=pass, got %v", body["status"])
	}
}
