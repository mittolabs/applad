//go:build integration

package tests

import "testing"

func TestConsoleFlow(t *testing.T) {
	email := "ci-console@example.com"
	password := "supersecret99"

	// Sign up — skip if signup is disabled (non-fresh environment)
	status, body := request(t, "POST", "/console/signup", map[string]string{
		"name": "CI Admin", "email": email, "password": password,
	}, nil)
	if status == 403 {
		t.Skipf("console signup disabled (existing install) — skipping: %v", body["message"])
	}
	if status != 201 {
		t.Fatalf("signup: expected 201, got %d: %v", status, body)
	}

	// Login
	status, body = request(t, "POST", "/console/login", map[string]string{
		"email": email, "password": password,
	}, nil)
	if status != 200 {
		t.Fatalf("login: expected 200, got %d: %v", status, body)
	}
	token, ok := body["token"].(string)
	if !ok || token == "" {
		t.Fatalf("login: missing token in response: %v", body)
	}

	// /me
	status, body = request(t, "GET", "/console/me", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if status != 200 {
		t.Fatalf("/me: expected 200, got %d: %v", status, body)
	}
	if body["email"] != email {
		t.Fatalf("/me: expected email %q, got %v", email, body["email"])
	}

	// Password reset (SMTP not configured → token returned in body)
	status, body = request(t, "POST", "/console/password-reset/request",
		map[string]string{"email": email}, nil)
	if status != 200 {
		t.Fatalf("reset request: expected 200, got %d: %v", status, body)
	}
	resetToken, ok := body["token"].(string)
	if !ok || resetToken == "" {
		t.Fatalf("reset request: expected token in response: %v", body)
	}

	// Confirm reset with new password
	status, body = request(t, "POST", "/console/password-reset/confirm", map[string]string{
		"token": resetToken, "password": "newsecret99",
	}, nil)
	if status != 200 {
		t.Fatalf("reset confirm: expected 200, got %d: %v", status, body)
	}

	// Login with new password
	status, body = request(t, "POST", "/console/login", map[string]string{
		"email": email, "password": "newsecret99",
	}, nil)
	if status != 200 {
		t.Fatalf("login after reset: expected 200, got %d: %v", status, body)
	}
}
