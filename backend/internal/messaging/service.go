// Package messaging implements Applad's messaging service:
// email via SMTP, SMS via Twilio, and push notifications via FCM.
package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"sync"
)

// Provider is the common interface for all messaging providers.
type Provider interface {
	Send(ctx context.Context, to, subject, body string) error
}

// Config holds settings for all messaging providers.
type Config struct {
	// SMTP
	Host     string
	Port     string
	Username string
	Password string
	From     string
	// Twilio SMS
	TwilioSID   string
	TwilioToken string
	TwilioFrom  string
	// FCM push notifications
	FCMServerKey string
}

// Service handles email, SMS, and push notification sending.
type Service struct {
	cfg Config

	mu     sync.RWMutex
	topics map[string]*Topic // topicId -> Topic
}

// Topic represents a messaging topic with subscribers.
type Topic struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Subscribers []string `json:"subscribers"` // email addresses, phone numbers, or FCM tokens
}

// NewService creates a new messaging Service.
func NewService(cfg Config) *Service {
	return &Service{
		cfg:    cfg,
		topics: make(map[string]*Topic),
	}
}

// ---------------------------------------------------------------------------
// Email (SMTP)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// SMS (Twilio)
// ---------------------------------------------------------------------------

// SendSMS sends an SMS via the Twilio API.
func (s *Service) SendSMS(ctx context.Context, to, body string) error {
	if s.cfg.TwilioSID == "" || s.cfg.TwilioToken == "" {
		return fmt.Errorf("messaging: Twilio not configured")
	}

	apiURL := fmt.Sprintf(
		"https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json",
		s.cfg.TwilioSID,
	)

	form := url.Values{}
	form.Set("From", s.cfg.TwilioFrom)
	form.Set("To", to)
	form.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("messaging: create twilio request: %w", err)
	}
	req.SetBasicAuth(s.cfg.TwilioSID, s.cfg.TwilioToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("messaging: twilio request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("messaging: twilio error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Push notifications (FCM v1)
// ---------------------------------------------------------------------------

// SendPush sends a push notification via Firebase Cloud Messaging.
func (s *Service) SendPush(ctx context.Context, token, title, body string) error {
	if s.cfg.FCMServerKey == "" {
		return fmt.Errorf("messaging: FCM not configured")
	}

	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"token": token,
			"notification": map[string]string{
				"title": title,
				"body":  body,
			},
		},
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("messaging: marshal fcm payload: %w", err)
	}

	// FCM v1 endpoint — the server key is used as a Bearer token.
	fcmURL := "https://fcm.googleapis.com/v1/projects/_/messages:send"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fcmURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("messaging: create fcm request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.FCMServerKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("messaging: fcm request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("messaging: fcm error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Topics / subscribers (in-memory MVP)
// ---------------------------------------------------------------------------

// CreateTopic creates a new messaging topic.
func (s *Service) CreateTopic(name string) *Topic {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("topic_%d", len(s.topics)+1)
	t := &Topic{
		ID:          id,
		Name:        name,
		Subscribers: []string{},
	}
	s.topics[id] = t
	return t
}

// AddSubscriber adds a subscriber (target) to a topic.
func (s *Service) AddSubscriber(topicID, target string) (*Topic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.topics[topicID]
	if !ok {
		return nil, fmt.Errorf("messaging: topic %q not found", topicID)
	}
	t.Subscribers = append(t.Subscribers, target)
	return t, nil
}

// GetTopic returns a topic by ID.
func (s *Service) GetTopic(topicID string) (*Topic, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.topics[topicID]
	if !ok {
		return nil, fmt.Errorf("messaging: topic %q not found", topicID)
	}
	return t, nil
}

// SendToTopic sends a message to all subscribers of a topic.
// It attempts delivery to every subscriber and returns the first error encountered.
func (s *Service) SendToTopic(ctx context.Context, topicID, subject, body string) error {
	s.mu.RLock()
	t, ok := s.topics[topicID]
	if !ok {
		s.mu.RUnlock()
		return fmt.Errorf("messaging: topic %q not found", topicID)
	}
	// Copy subscribers so we can release the lock.
	subs := make([]string, len(t.Subscribers))
	copy(subs, t.Subscribers)
	s.mu.RUnlock()

	var firstErr error
	for _, sub := range subs {
		var err error
		switch {
		case strings.HasPrefix(sub, "+"):
			// Phone number — send SMS
			err = s.SendSMS(ctx, sub, body)
		case strings.Contains(sub, "@"):
			// Email address — send email
			err = s.SendEmail(ctx, []string{sub}, subject, body)
		default:
			// Assume FCM device token — send push
			err = s.SendPush(ctx, sub, subject, body)
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
