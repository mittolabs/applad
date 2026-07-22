//go:build integration

package tests

import "testing"

func TestAuthFlow(t *testing.T) {
	projectID, _ := projectWithKey(t, "auth-test")
	h := map[string]string{"X-Applad-Project": projectID}

	// Create account
	status, body := request(t, "POST", "/account", map[string]string{
		"userId": "unique()", "email": "test@example.com",
		"password": "testpassword123", "name": "Test User",
	}, h)
	if status != 201 {
		t.Fatalf("create account: expected 201, got %d: %v", status, body)
	}

	// Login
	status, body = request(t, "POST", "/account/sessions/email", map[string]string{
		"email": "test@example.com", "password": "testpassword123",
	}, h)
	if status != 201 {
		t.Fatalf("login: expected 201, got %d: %v", status, body)
	}

	// Wrong password
	status, _ = request(t, "POST", "/account/sessions/email", map[string]string{
		"email": "test@example.com", "password": "wrongpassword",
	}, h)
	if status != 401 {
		t.Fatalf("bad login: expected 401, got %d", status)
	}
}
