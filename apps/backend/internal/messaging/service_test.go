package messaging

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

// captureSMTP starts a throwaway in-process SMTP server that speaks just enough
// of the protocol for net/smtp.SendMail, records the DATA payload it receives,
// and returns its host:port. The captured message is sent on msgCh.
func captureSMTP(t *testing.T) (host, port string, msgCh <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ch := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		defer ln.Close()
		r := bufio.NewReader(conn)
		w := bufio.NewWriter(conn)
		write := func(s string) { w.WriteString(s + "\r\n"); w.Flush() }
		write("220 test ESMTP")
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if strings.TrimRight(line, "\r\n") == "." {
					inData = false
					write("250 OK")
					ch <- data.String()
					continue
				}
				data.WriteString(line)
				continue
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				write("250 localhost")
			case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
				write("250 OK")
			case strings.HasPrefix(cmd, "DATA"):
				write("354 end data with <CRLF>.<CRLF>")
				inData = true
			case strings.HasPrefix(cmd, "QUIT"):
				write("221 bye")
				return
			default:
				write("250 OK")
			}
		}
	}()
	h, p, _ := net.SplitHostPort(ln.Addr().String())
	return h, p, ch
}

// TestSendEmail_SubjectHeaderInjection_Stripped proves a CR/LF-bearing subject
// cannot inject an extra header: the composed message that reaches the SMTP
// server carries no injected "Bcc:" header, and the subject is folded to one
// line.
func TestSendEmail_SubjectHeaderInjection_Stripped(t *testing.T) {
	host, port, msgCh := captureSMTP(t)
	svc := NewService(nil, Config{Host: host, Port: port, From: "noreply@test.com"})
	subject := "Hello\r\nBcc: attacker@evil.com"
	if err := svc.SendEmail(nil, []string{"user@test.com"}, subject, "<p>body</p>"); err != nil {
		t.Fatalf("SendEmail: %v", err)
	}
	msg := <-msgCh
	if !strings.Contains(msg, "Subject: Hello") {
		t.Fatalf("expected folded subject line, got:\n%s", msg)
	}
	// No line in the header block may be a standalone injected header. The
	// attacker text folds onto the single Subject line (inert), but it must not
	// appear as its own "Bcc:" header line.
	headerBlock := msg
	if idx := strings.Index(msg, "\r\n\r\n"); idx >= 0 {
		headerBlock = msg[:idx]
	}
	for _, line := range strings.Split(headerBlock, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Bcc:") {
			t.Fatalf("injected Bcc header line present:\n%s", headerBlock)
		}
	}
}

// TestSendEmail_RecipientNewline_Rejected proves a recipient address carrying a
// newline is refused rather than composed into the message.
func TestSendEmail_RecipientNewline_Rejected(t *testing.T) {
	svc := NewService(nil, Config{Host: "smtp.test.com", Port: "25", From: "noreply@test.com"})
	err := svc.SendEmail(nil, []string{"user@test.com\r\nBcc: attacker@evil.com"}, "subject", "body")
	if err == nil {
		t.Fatal("expected send to be rejected for recipient with newline")
	}
	if !strings.Contains(err.Error(), "newline") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSendEmail_LegitimateSend_Succeeds proves the sanitization does not break a
// normal send: a clean subject and recipient reach the server intact.
func TestSendEmail_LegitimateSend_Succeeds(t *testing.T) {
	host, port, msgCh := captureSMTP(t)
	svc := NewService(nil, Config{Host: host, Port: port, From: "noreply@test.com"})
	if err := svc.SendEmail(nil, []string{"user@test.com"}, "Welcome aboard", "<p>hi</p>"); err != nil {
		t.Fatalf("SendEmail: %v", err)
	}
	msg := <-msgCh
	if !strings.Contains(msg, "Subject: Welcome aboard") {
		t.Fatalf("subject missing/altered:\n%s", msg)
	}
	if !strings.Contains(msg, "To: user@test.com") {
		t.Fatalf("recipient missing/altered:\n%s", msg)
	}
	if !strings.Contains(msg, "<p>hi</p>") {
		t.Fatalf("body missing:\n%s", msg)
	}
}

func TestStripHeaderNewlines(t *testing.T) {
	got := stripHeaderNewlines("a\r\nb\nc\rd")
	if got != "abcd" {
		t.Fatalf("expected abcd, got %q", got)
	}
}

func TestNewService_StoresConfig(t *testing.T) {
	cfg := Config{
		Host:         "smtp.example.com",
		Port:         "587",
		Username:     "user",
		Password:     "pass",
		From:         "noreply@example.com",
		TwilioSID:    "AC123",
		TwilioToken:  "tok123",
		TwilioFrom:   "+1234567890",
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
	err := svc.SendPush(nil, "device-token", "title", "body", nil)
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
