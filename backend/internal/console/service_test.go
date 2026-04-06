package console

import (
	"testing"
)

func TestSignupEnabled_True(t *testing.T) {
	svc := &Service{jwtSecret: "test-secret"}
	enabled, err := svc.SignupEnabled(nil, "true")
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("expected true")
	}
}

func TestSignupEnabled_False(t *testing.T) {
	svc := &Service{jwtSecret: "test-secret"}
	enabled, err := svc.SignupEnabled(nil, "false")
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("expected false")
	}
}

func TestSignJWT_Roundtrip(t *testing.T) {
	svc := &Service{jwtSecret: "test-secret-key-12345"}

	token, err := svc.signJWT("user123", "test@example.com")
	if err != nil {
		t.Fatalf("signJWT failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	userID, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if userID != "user123" {
		t.Errorf("expected user123, got %s", userID)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	svc := &Service{jwtSecret: "test-secret"}

	_, err := svc.ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	svc1 := &Service{jwtSecret: "secret-1"}
	svc2 := &Service{jwtSecret: "secret-2"}

	token, _ := svc1.signJWT("user1", "test@test.com")
	_, err := svc2.ValidateToken(token)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestConsoleClaims_ConsoleFlag(t *testing.T) {
	svc := &Service{jwtSecret: "test-secret"}

	token, err := svc.signJWT("user1", "test@test.com")
	if err != nil {
		t.Fatal(err)
	}

	// Validate returns user ID only if console=true in claims
	userID, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("expected valid console token: %v", err)
	}
	if userID != "user1" {
		t.Errorf("expected user1, got %s", userID)
	}
}
