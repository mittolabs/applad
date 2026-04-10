package messaging

import (
	"testing"
)

func TestNewService_StoresConfig(t *testing.T) {
	cfg := Config{
		Host:        "smtp.example.com",
		Port:        "587",
		Username:    "user",
		Password:    "pass",
		From:        "noreply@example.com",
		TwilioSID:   "AC123",
		TwilioToken: "tok123",
		TwilioFrom:  "+1234567890",
		FCMServerKey: "fcm-key",
	}
	svc := NewService(nil, cfg)
	if svc.cfg.Host != "smtp.example.com" {
		t.Fatalf("expected smtp.example.com, got %s", svc.cfg.Host)
	}
	if svc.cfg.TwilioSID != "AC123" {
		t.Fatalf("expected AC123, got %s", svc.cfg.TwilioSID)
	}
	if svc.cfg.FCMServerKey != "fcm-key" {
		t.Fatalf("expected fcm-key, got %s", svc.cfg.FCMServerKey)
	}
}

func TestSendEmail_NoSMTPHost_ReturnsError(t *testing.T) {
	svc := NewService(nil, Config{})
	err := svc.SendEmail(nil, []string{"test@test.com"}, "subject", "body")
	if err == nil {
		t.Fatal("expected error when SMTP not configured")
	}
	if err.Error() != "messaging: SMTP not configured" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendSMS_NoTwilio_ReturnsError(t *testing.T) {
	svc := NewService(nil, Config{})
	err := svc.SendSMS(nil, "+1234567890", "hello")
	if err == nil {
		t.Fatal("expected error when Twilio not configured")
	}
}

func TestSendPush_NoFCM_ReturnsError(t *testing.T) {
	svc := NewService(nil, Config{})
	err := svc.SendPush(nil, "device-token", "title", "body")
	if err == nil {
		t.Fatal("expected error when FCM not configured")
	}
}

func TestPasswordResetTemplate(t *testing.T) {
	svc := NewService(nil, Config{Host: "smtp.test.com", From: "no-reply@test.com"})
	// Just verify it doesn't panic
	err := svc.SendPasswordReset(nil, "user@test.com", "https://reset.url/token123")
	// Will fail because smtp.test.com doesn't exist, but shouldn't panic
	if err == nil {
		t.Log("unexpected success — SMTP server responded")
	}
}
