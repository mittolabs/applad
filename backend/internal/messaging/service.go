// Package messaging implements Applad's messaging service:
// email via SMTP/Mailgun/Resend, SMS via Twilio/Vonage/MSG91,
// and push notifications via FCM/APNS.
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
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

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
	// FCM push notifications (legacy server key)
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

// Message is a persistent record of a sent/draft message.
type Message struct {
	ID          string   `json:"id"`
	ProjectID   string   `json:"projectId"`
	Type        string   `json:"type"`
	Subject     string   `json:"subject"`
	Body        string   `json:"body"`
	Recipients  []string `json:"recipients"`
	Status      string   `json:"status"`
	ScheduledAt *string  `json:"scheduledAt"`
	DeliveredAt *string  `json:"deliveredAt"`
	CreatedAt   string   `json:"createdAt"`
}

// Topic represents a messaging topic with subscribers.
type Topic struct {
	ID          string   `json:"id"`
	ProjectID   string   `json:"projectId"`
	Name        string   `json:"name"`
	Subscribers []string `json:"subscribers"`
}

// Service handles email, SMS, and push notification sending.
type Service struct {
	cfg        Config
	db         *db.DB
	httpClient *http.Client
}

// NewService creates a new messaging Service.
func NewService(database *db.DB, cfg Config) *Service {
	return &Service{
		cfg: cfg,
		db:  database,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// CreateMessage persists a new message record in the given status.
func (s *Service) CreateMessage(ctx context.Context, projectID, msgType, subject, body string, recipients []string, status string) (*Message, error) {
	id := uid.New("msg")
	recipJSON, err := json.Marshal(recipients)
	if err != nil {
		return nil, fmt.Errorf("messaging: marshal recipients: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO messages (id, project_id, type, subject, body, recipients, status) VALUES (?,?,?,?,?,?,?)`,
		id, projectID, msgType, subject, body, string(recipJSON), status,
	)
	if err != nil {
		return nil, fmt.Errorf("messaging: insert message: %w", err)
	}
	return &Message{
		ID:         id,
		ProjectID:  projectID,
		Type:       msgType,
		Subject:    subject,
		Body:       body,
		Recipients: recipients,
		Status:     status,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// UpdateMessageStatus sets the status (and optionally deliveredAt) for a message.
func (s *Service) UpdateMessageStatus(ctx context.Context, id, status string) error {
	var err error
	if status == "sent" {
		_, err = s.db.ExecContext(ctx,
			`UPDATE messages SET status=?, delivered_at=NOW() WHERE id=?`, status, id)
	} else {
		_, err = s.db.ExecContext(ctx,
			`UPDATE messages SET status=? WHERE id=?`, status, id)
	}
	return err
}

// ListMessages returns paginated messages for a project.
func (s *Service) ListMessages(ctx context.Context, projectID string, limit, offset int, search string) ([]*Message, int, error) {
	args := []interface{}{projectID}
	where := "project_id = ?"
	if search != "" {
		where += " AND (id LIKE ? OR type LIKE ? OR status LIKE ? OR subject LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}

	var total int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM messages WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("messaging: count messages: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, type, subject, body, recipients, status,
		        to_char(scheduled_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		        to_char(delivered_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		        to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM messages WHERE `+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("messaging: list messages: %w", err)
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		m := &Message{}
		var recipJSON string
		err := rows.Scan(&m.ID, &m.ProjectID, &m.Type, &m.Subject, &m.Body,
			&recipJSON, &m.Status, &m.ScheduledAt, &m.DeliveredAt, &m.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal([]byte(recipJSON), &m.Recipients)
		if m.Recipients == nil {
			m.Recipients = []string{}
		}
		msgs = append(msgs, m)
	}
	if msgs == nil {
		msgs = []*Message{}
	}
	return msgs, total, rows.Err()
}

// GetMessage returns a single message by ID.
func (s *Service) GetMessage(ctx context.Context, projectID, id string) (*Message, error) {
	m := &Message{}
	var recipJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, type, subject, body, recipients, status,
		        to_char(scheduled_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		        to_char(delivered_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		        to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM messages WHERE id=? AND project_id=?`, id, projectID).
		Scan(&m.ID, &m.ProjectID, &m.Type, &m.Subject, &m.Body,
			&recipJSON, &m.Status, &m.ScheduledAt, &m.DeliveredAt, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("messaging: get message: %w", err)
	}
	_ = json.Unmarshal([]byte(recipJSON), &m.Recipients)
	if m.Recipients == nil {
		m.Recipients = []string{}
	}
	return m, nil
}

// DeleteMessage removes a message record.
func (s *Service) DeleteMessage(ctx context.Context, projectID, id string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM messages WHERE id=? AND project_id=?", id, projectID)
	return err
}

// ---------------------------------------------------------------------------
// Email
// ---------------------------------------------------------------------------

// SendEmail sends an email via SMTP (primary) or returns an error if unconfigured.
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
	return s.doRequest(req, "mailgun")
}

// SendEmailResend sends an email via the Resend API.
func (s *Service) SendEmailResend(ctx context.Context, to []string, subject, htmlBody string) error {
	if s.cfg.ResendAPIKey == "" {
		return fmt.Errorf("messaging: Resend not configured")
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"from": s.cfg.From, "to": to, "subject": subject, "html": htmlBody,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("messaging: create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")
	return s.doRequest(req, "resend")
}

// ---------------------------------------------------------------------------
// SMS
// ---------------------------------------------------------------------------

// SendSMS sends an SMS via Twilio.
func (s *Service) SendSMS(ctx context.Context, to, body string) error {
	if s.cfg.TwilioSID == "" || s.cfg.TwilioToken == "" {
		return fmt.Errorf("messaging: Twilio not configured")
	}
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.cfg.TwilioSID)
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
	return s.doRequest(req, "twilio")
}

// SendSMSVonage sends an SMS via the Vonage (Nexmo) API.
func (s *Service) SendSMSVonage(ctx context.Context, to, text string) error {
	if s.cfg.VonageAPIKey == "" || s.cfg.VonageAPISecret == "" {
		return fmt.Errorf("messaging: Vonage not configured")
	}
	payload, _ := json.Marshal(map[string]string{
		"api_key": s.cfg.VonageAPIKey, "api_secret": s.cfg.VonageAPISecret,
		"from": s.cfg.VonageFrom, "to": to, "text": text,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://rest.nexmo.com/sms/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("messaging: create vonage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return s.doRequest(req, "vonage")
}

// SendSMSMSG91 sends an SMS via the MSG91 API.
func (s *Service) SendSMSMSG91(ctx context.Context, to, body string) error {
	if s.cfg.MSG91AuthKey == "" {
		return fmt.Errorf("messaging: MSG91 not configured")
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"sender": s.cfg.MSG91SenderID, "route": "4", "country": "91",
		"sms": []map[string]interface{}{{"message": body, "to": []string{to}}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://control.msg91.com/api/v5/flow/", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("messaging: create msg91 request: %w", err)
	}
	req.Header.Set("authkey", s.cfg.MSG91AuthKey)
	req.Header.Set("Content-Type", "application/json")
	return s.doRequest(req, "msg91")
}

// ---------------------------------------------------------------------------
// Push notifications
// ---------------------------------------------------------------------------

// SendPush sends a push notification via the FCM legacy HTTP API.
// Uses the server key as Authorization header — correct for the legacy API.
func (s *Service) SendPush(ctx context.Context, token, title, body string) error {
	if s.cfg.FCMServerKey == "" {
		return fmt.Errorf("messaging: FCM not configured")
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"to": token,
		"notification": map[string]string{
			"title": title,
			"body":  body,
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fcm.googleapis.com/fcm/send", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("messaging: create fcm request: %w", err)
	}
	req.Header.Set("Authorization", "key="+s.cfg.FCMServerKey)
	req.Header.Set("Content-Type", "application/json")
	return s.doRequest(req, "fcm")
}

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

	apnsPayload, _ := json.Marshal(map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]string{"title": title, "body": body},
			"sound": "default",
		},
	})
	apnsURL := fmt.Sprintf("https://api.push.apple.com/3/device/%s", deviceToken)
	req, err := http.NewRequestWithContext(ctx, "POST", apnsURL, bytes.NewReader(apnsPayload))
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
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("messaging: APNS error %d: %s", resp.StatusCode, b)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Topics (DB-backed)
// ---------------------------------------------------------------------------

// CreateTopic persists a new topic.
func (s *Service) CreateTopic(ctx context.Context, projectID, name string) (*Topic, error) {
	id := uid.New("msg")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO msg_topics (id, project_id, name) VALUES (?,?,?)`,
		id, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("messaging: create topic: %w", err)
	}
	return &Topic{ID: id, ProjectID: projectID, Name: name, Subscribers: []string{}}, nil
}

// ListTopics returns all topics for a project.
func (s *Service) ListTopics(ctx context.Context, projectID string) ([]*Topic, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT t.id, t.name,
		        string_agg(ts.target, ',' ORDER BY ts.id) AS subs
		 FROM msg_topics t
		 LEFT JOIN msg_topic_subscribers ts ON ts.topic_id = t.id
		 WHERE t.project_id = ?
		 GROUP BY t.id, t.name
		 ORDER BY t.created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("messaging: list topics: %w", err)
	}
	defer rows.Close()
	var topics []*Topic
	for rows.Next() {
		t := &Topic{ProjectID: projectID}
		var subsStr *string
		if err := rows.Scan(&t.ID, &t.Name, &subsStr); err != nil {
			return nil, err
		}
		if subsStr != nil && *subsStr != "" {
			t.Subscribers = strings.Split(*subsStr, ",")
		} else {
			t.Subscribers = []string{}
		}
		topics = append(topics, t)
	}
	if topics == nil {
		topics = []*Topic{}
	}
	return topics, rows.Err()
}

// GetTopic returns a topic by ID.
func (s *Service) GetTopic(ctx context.Context, projectID, topicID string) (*Topic, error) {
	t := &Topic{ProjectID: projectID}
	var subsStr *string
	err := s.db.QueryRowContext(ctx,
		`SELECT t.id, t.name,
		        string_agg(ts.target, ',' ORDER BY ts.id) AS subs
		 FROM msg_topics t
		 LEFT JOIN msg_topic_subscribers ts ON ts.topic_id = t.id
		 WHERE t.id=? AND t.project_id=?
		 GROUP BY t.id, t.name`, topicID, projectID).
		Scan(&t.ID, &t.Name, &subsStr)
	if err != nil {
		return nil, fmt.Errorf("messaging: topic not found")
	}
	if subsStr != nil && *subsStr != "" {
		t.Subscribers = strings.Split(*subsStr, ",")
	} else {
		t.Subscribers = []string{}
	}
	return t, nil
}

// AddSubscriber adds a target to a topic.
func (s *Service) AddSubscriber(ctx context.Context, projectID, topicID, target string) (*Topic, error) {
	// Verify topic belongs to project
	var count int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM msg_topics WHERE id=? AND project_id=?",
		topicID, projectID).Scan(&count); err != nil || count == 0 {
		return nil, fmt.Errorf("messaging: topic not found")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO msg_topic_subscribers (topic_id, target) VALUES (?,?) ON CONFLICT DO NOTHING`,
		topicID, target)
	if err != nil {
		return nil, fmt.Errorf("messaging: add subscriber: %w", err)
	}
	return s.GetTopic(ctx, projectID, topicID)
}

// SendToTopic sends a message to all subscribers of a topic.
func (s *Service) SendToTopic(ctx context.Context, projectID, topicID, subject, body string) error {
	t, err := s.GetTopic(ctx, projectID, topicID)
	if err != nil {
		return err
	}
	var firstErr error
	for _, sub := range t.Subscribers {
		var sendErr error
		switch {
		case strings.HasPrefix(sub, "+"):
			sendErr = s.SendSMS(ctx, sub, body)
		case strings.Contains(sub, "@"):
			sendErr = s.SendEmail(ctx, []string{sub}, subject, body)
		default:
			sendErr = s.SendPush(ctx, sub, subject, body)
		}
		if sendErr != nil && firstErr == nil {
			firstErr = sendErr
		}
	}
	return firstErr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *Service) doRequest(req *http.Request, provider string) error {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("messaging: %s request failed: %w", provider, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("messaging: %s error %d: %s", provider, resp.StatusCode, b)
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
