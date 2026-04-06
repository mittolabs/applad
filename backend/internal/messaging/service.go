// Package messaging implements Applad's messaging service:
// email via SMTP providers.
package messaging

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// Config holds SMTP settings.
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// Service handles email sending.
type Service struct {
	cfg Config
}

// NewService creates a new messaging Service.
func NewService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// SendEmail sends an email via SMTP.
func (s *Service) SendEmail(ctx context.Context, to []string, subject, htmlBody string) error {
	if s.cfg.Host == "" {
		return fmt.Errorf("messaging: SMTP not configured")
	}

	addr := s.cfg.Host + ":" + s.cfg.Port
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	headers := []string{
		fmt.Sprintf("From: %s", s.cfg.From),
		fmt.Sprintf("To: %s", strings.Join(to, ", ")),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}
	msg := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + htmlBody)

	return smtp.SendMail(addr, auth, s.cfg.From, to, msg)
}

// SendPasswordReset sends a password reset email.
func (s *Service) SendPasswordReset(ctx context.Context, email, resetURL string) error {
	body := fmt.Sprintf(`
		<h2>Password Reset</h2>
		<p>Click the link below to reset your password:</p>
		<p><a href="%s">Reset Password</a></p>
		<p>If you did not request this, please ignore this email.</p>
	`, resetURL)
	return s.SendEmail(ctx, []string{email}, "Reset Your Password", body)
}

// SendWelcome sends a welcome email.
func (s *Service) SendWelcome(ctx context.Context, email, name string) error {
	greeting := "there"
	if name != "" {
		greeting = name
	}
	body := fmt.Sprintf(`
		<h2>Welcome to Applad!</h2>
		<p>Hi %s,</p>
		<p>Your account has been created successfully.</p>
	`, greeting)
	return s.SendEmail(ctx, []string{email}, "Welcome to Applad", body)
}
