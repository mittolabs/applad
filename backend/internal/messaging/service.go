// Package messaging implements Applad's messaging service:
// email via SMTP, SMS via Twilio, and push notifications via FCM.
package messaging

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
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
	// Mailgun
	MailgunAPIKey string
	MailgunDomain string
	// Resend
	ResendAPIKey string
	// Vonage SMS
	VonageAPIKey    string
	VonageAPISecret string
	VonageFrom      string
	// MSG91 SMS
	MSG91AuthKey  string
	MSG91SenderID string
	// APNS push notifications
	APNSKeyID    string
	APNSTeamID   string
	APNSKeyPath  string
	APNSBundleID string
}

// Service handles email, SMS, and push notification sending.
type Service struct {
	cfg        Config
	httpClient *http.Client

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
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
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

	resp, err := s.httpClient.Do(req)
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

	resp, err := s.httpClient.Do(req)
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
// Email (Mailgun)
// ---------------------------------------------------------------------------

// SendEmailMailgun sends an email via the Mailgun API.
func (s *Service) SendEmailMailgun(ctx context.Context, to []string, subject, htmlBody string) error {
	if s.cfg.MailgunAPIKey == "" || s.cfg.MailgunDomain == "" {
		return fmt.Errorf("messaging: Mailgun not configured")
	}

	apiURL := fmt.Sprintf("https://api.mailgun.net/v3/%s/messages", s.cfg.MailgunDomain)

	form := url.Values{}
	form.Set("from", s.cfg.From)
	form.Set("to", strings.Join(to, ", "))
	form.Set("subject", subject)
	form.Set("html", htmlBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("messaging: create mailgun request: %w", err)
	}
	req.SetBasicAuth("api", s.cfg.MailgunAPIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("messaging: mailgun request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("messaging: mailgun error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Email (Resend)
// ---------------------------------------------------------------------------

// SendEmailResend sends an email via the Resend API.
func (s *Service) SendEmailResend(ctx context.Context, to []string, subject, htmlBody string) error {
	if s.cfg.ResendAPIKey == "" {
		return fmt.Errorf("messaging: Resend not configured")
	}

	payload := map[string]interface{}{
		"from":    s.cfg.From,
		"to":      to,
		"subject": subject,
		"html":    htmlBody,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("messaging: marshal resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("messaging: create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("messaging: resend request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("messaging: resend error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ---------------------------------------------------------------------------
// SMS (Vonage)
// ---------------------------------------------------------------------------

// SendSMSVonage sends an SMS via the Vonage (Nexmo) API.
func (s *Service) SendSMSVonage(ctx context.Context, to, text string) error {
	if s.cfg.VonageAPIKey == "" || s.cfg.VonageAPISecret == "" {
		return fmt.Errorf("messaging: Vonage not configured")
	}

	payload := map[string]string{
		"api_key":    s.cfg.VonageAPIKey,
		"api_secret": s.cfg.VonageAPISecret,
		"from":       s.cfg.VonageFrom,
		"to":         to,
		"text":       text,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("messaging: marshal vonage payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://rest.nexmo.com/sms/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("messaging: create vonage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("messaging: vonage request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("messaging: vonage error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ---------------------------------------------------------------------------
// SMS (MSG91)
// ---------------------------------------------------------------------------

// SendSMSMSG91 sends an SMS via the MSG91 API.
func (s *Service) SendSMSMSG91(ctx context.Context, to, body string) error {
	if s.cfg.MSG91AuthKey == "" {
		return fmt.Errorf("messaging: MSG91 not configured")
	}

	payload := map[string]interface{}{
		"sender":  s.cfg.MSG91SenderID,
		"route":   "4", // transactional route
		"country": "91",
		"sms": []map[string]interface{}{
			{
				"message": body,
				"to":      []string{to},
			},
		},
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("messaging: marshal msg91 payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://control.msg91.com/api/v5/flow/", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("messaging: create msg91 request: %w", err)
	}
	req.Header.Set("authkey", s.cfg.MSG91AuthKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("messaging: msg91 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("messaging: msg91 error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Push notifications (APNS)
// ---------------------------------------------------------------------------

// SendPushAPNS sends an Apple Push Notification via the HTTP/2 APNS API.
func (s *Service) SendPushAPNS(ctx context.Context, deviceToken, title, body string) error {
	if s.cfg.APNSKeyID == "" || s.cfg.APNSTeamID == "" || s.cfg.APNSKeyPath == "" || s.cfg.APNSBundleID == "" {
		return fmt.Errorf("messaging: APNS not configured (set APNS_KEY_ID, APNS_TEAM_ID, APNS_KEY_PATH, APNS_BUNDLE_ID)")
	}

	keyData, err := os.ReadFile(s.cfg.APNSKeyPath)
	if err != nil {
		return fmt.Errorf("messaging: read APNS .p8 key: %w", err)
	}
	block, _ := pem.Decode(keyData)
	if block == nil {
		return fmt.Errorf("messaging: APNS key is not valid PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("messaging: parse APNS key: %w", err)
	}
	ecKey, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return fmt.Errorf("messaging: APNS key is not ECDSA (ES256)")
	}

	// Build JWT (ES256)
	now := time.Now()
	hdr := apnsB64([]byte(`{"alg":"ES256","kid":"` + s.cfg.APNSKeyID + `"}`))
	clm := apnsB64([]byte(fmt.Sprintf(`{"iss":"%s","iat":%d}`, s.cfg.APNSTeamID, now.Unix())))
	unsigned := hdr + "." + clm

	hash := sha256.Sum256([]byte(unsigned))
	r, ss, err := ecdsa.Sign(rand.Reader, ecKey, hash[:])
	if err != nil {
		return fmt.Errorf("messaging: sign APNS JWT: %w", err)
	}
	sig := apnsB64(append(apnsPad32(r), apnsPad32(ss)...))
	jwt := unsigned + "." + sig

	payload, _ := json.Marshal(map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]string{"title": title, "body": body},
			"sound": "default",
		},
	})

	apnsURL := fmt.Sprintf("https://api.push.apple.com/3/device/%s", deviceToken)
	req, err := http.NewRequestWithContext(ctx, "POST", apnsURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "bearer "+jwt)
	req.Header.Set("apns-topic", s.cfg.APNSBundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("messaging: APNS request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("messaging: APNS error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func apnsB64(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func apnsPad32(n *big.Int) []byte {
	b := n.Bytes()
	if len(b) >= 32 {
		return b[:32]
	}
	p := make([]byte, 32)
	copy(p[32-len(b):], b)
	return p
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
