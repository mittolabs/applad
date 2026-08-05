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
	"errors"
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

// CreateMessage persists a new message record in the given status. A non-nil
// scheduledAt records when a message should be delivered; the scheduled sweep
// (see SweepScheduledMessages) picks it up once that time has passed.
func (s *Service) CreateMessage(ctx context.Context, projectID, msgType, subject, body string, recipients []string, status string, scheduledAt *time.Time) (*Message, error) {
	id := uid.New("unique()")
	recipJSON, err := json.Marshal(recipients)
	if err != nil {
		return nil, fmt.Errorf("messaging: marshal recipients: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO messages (id, project_id, type, subject, body, recipients, status, scheduled_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, projectID, msgType, subject, body, string(recipJSON), status, scheduledAt,
	)
	if err != nil {
		return nil, fmt.Errorf("messaging: insert message: %w", err)
	}
	m := &Message{
		ID:         id,
		ProjectID:  projectID,
		Type:       msgType,
		Subject:    subject,
		Body:       body,
		Recipients: recipients,
		Status:     status,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if scheduledAt != nil {
		sa := scheduledAt.UTC().Format(time.RFC3339)
		m.ScheduledAt = &sa
	}
	return m, nil
}

// SendMessage delivers a persisted message through the channel named by its
// type, resolving the project's own provider first. This is the shared delivery
// path used by both the inline send on create and the scheduled sweep.
func (s *Service) SendMessage(ctx context.Context, m *Message) error {
	switch m.Type {
	case "email":
		return s.SendEmailForProject(ctx, m.ProjectID, m.Recipients, m.Subject, m.Body)
	case "sms":
		return s.SendSMSMulti(ctx, m.ProjectID, m.Recipients, m.Body)
	case "push":
		return s.SendPushMulti(ctx, m.ProjectID, m.Recipients, m.Subject, m.Body, nil)
	default:
		return fmt.Errorf("messaging: unknown message type %q", m.Type)
	}
}

// SweepScheduledMessages delivers messages whose scheduled_at has arrived. It
// runs on the per-minute cron tick. Each due message is claimed with a
// conditional status transition (scheduled -> processing) so that a second
// sweeper — or a replaying worker — updates zero rows and skips it, which is
// what makes delivery exactly-once even without the cron lock. After delivery
// the status becomes sent or failed. deliver may be nil, in which case the
// service's own SendMessage is used; tests pass a stub. Returns the number of
// messages claimed for delivery.
func (s *Service) SweepScheduledMessages(ctx context.Context, deliver func(context.Context, *Message) error) (int, error) {
	if deliver == nil {
		deliver = s.SendMessage
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, type, subject, body, recipients
		   FROM messages
		  WHERE status = 'scheduled' AND scheduled_at IS NOT NULL AND scheduled_at <= NOW()
		  ORDER BY scheduled_at ASC
		  LIMIT 200`)
	if err != nil {
		return 0, fmt.Errorf("messaging: query due messages: %w", err)
	}
	var due []*Message
	for rows.Next() {
		m := &Message{}
		var recipJSON string
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Type, &m.Subject, &m.Body, &recipJSON); err != nil {
			rows.Close()
			return 0, err
		}
		_ = json.Unmarshal([]byte(recipJSON), &m.Recipients)
		if m.Recipients == nil {
			m.Recipients = []string{}
		}
		due = append(due, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	claimed := 0
	for _, m := range due {
		// Claim before sending. Whoever flips scheduled -> processing owns the
		// send; a racing sweeper's UPDATE matches no row and moves on.
		res, err := s.db.ExecContext(ctx,
			`UPDATE messages SET status = 'processing' WHERE id = $1 AND status = 'scheduled'`, m.ID)
		if err != nil {
			return claimed, fmt.Errorf("messaging: claim scheduled message: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			continue // already claimed elsewhere
		}
		claimed++
		status := "sent"
		if sendErr := deliver(ctx, m); sendErr != nil {
			status = "failed"
		}
		if err := s.UpdateMessageStatus(ctx, m.ID, status); err != nil {
			return claimed, fmt.Errorf("messaging: update scheduled message status: %w", err)
		}
	}
	return claimed, nil
}

// UpdateMessageStatus sets the status (and optionally deliveredAt) for a message.
func (s *Service) UpdateMessageStatus(ctx context.Context, id, status string) error {
	var err error
	if status == "sent" {
		_, err = s.db.ExecContext(ctx,
			`UPDATE messages SET status=$1, delivered_at=NOW() WHERE id=$2`, status, id)
	} else {
		_, err = s.db.ExecContext(ctx,
			`UPDATE messages SET status=$1 WHERE id=$2`, status, id)
	}
	return err
}

// ListMessages returns paginated messages for a project.
func (s *Service) ListMessages(ctx context.Context, projectID string, limit, offset int, search string) ([]*Message, int, error) {
	n := 1
	args := []interface{}{projectID}
	where := "project_id = $1"
	if search != "" {
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
		where += fmt.Sprintf(" AND (id LIKE $%d OR type LIKE $%d OR status LIKE $%d OR subject LIKE $%d)", n+1, n+2, n+3, n+4)
		n += 4
	}

	var total int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM messages WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("messaging: count messages: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, project_id, type, subject, body, recipients, status,
		        to_char(scheduled_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		        to_char(delivered_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		        to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM messages WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, n+1, n+2),
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
		 FROM messages WHERE id=$1 AND project_id=$2`, id, projectID).
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
		"DELETE FROM messages WHERE id=$1 AND project_id=$2", id, projectID)
	return err
}

// ---------------------------------------------------------------------------
// Email
// ---------------------------------------------------------------------------

// stripHeaderNewlines removes CR and LF from a value bound for a single email
// header line. A value that carries "\r\n" would otherwise start a new header
// (or the body), which is SMTP header injection. The subject and the From
// display are folded to a single line rather than rejected, so a stray newline
// degrades gracefully instead of failing a legitimate send.
func stripHeaderNewlines(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(v)
}

// validateHeaderAddresses rejects the send if any recipient address contains a
// CR or LF. Unlike the subject, an address cannot be silently repaired — a
// newline in it is either a malformed address or an injection attempt — so the
// whole send is refused with a clear error.
func validateHeaderAddresses(to []string) error {
	for _, addr := range to {
		if strings.ContainsAny(addr, "\r\n") {
			return fmt.Errorf("messaging: recipient address contains a newline")
		}
	}
	return nil
}

// SendEmail sends an email via SMTP (primary) or returns an error if unconfigured.
// Recipient caps per single send. The project-work rate limit counts requests,
// but one request accepts a recipient array; without a cap a single call could
// fan out to an unbounded (and, for SMS, billable) list. Higher-volume delivery
// belongs to topics, which fan out to actual subscribers.
const (
	maxSMSRecipients   = 100
	maxEmailRecipients = 1000
	maxPushRecipients  = 1000
)

// ErrTooManyRecipients is returned when a single send exceeds its recipient cap.
var ErrTooManyRecipients = errors.New("messaging: too many recipients in one request")

func (s *Service) SendEmail(ctx context.Context, to []string, subject, htmlBody string) error {
	if len(to) > maxEmailRecipients {
		return ErrTooManyRecipients
	}
	if s.cfg.Host == "" {
		return fmt.Errorf("messaging: SMTP not configured")
	}
	if err := validateHeaderAddresses(to); err != nil {
		return err
	}
	addr := s.cfg.Host + ":" + s.cfg.Port
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	headers := []string{
		fmt.Sprintf("From: %s", stripHeaderNewlines(s.cfg.From)),
		fmt.Sprintf("To: %s", stripHeaderNewlines(strings.Join(to, ", "))),
		fmt.Sprintf("Subject: %s", stripHeaderNewlines(subject)),
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
// data, when non-empty, is forwarded as FCM's "data" payload so clients can
// carry custom key/value pairs alongside the notification.
func (s *Service) SendPush(ctx context.Context, token, title, body string, data map[string]string) error {
	if s.cfg.FCMServerKey == "" {
		return fmt.Errorf("messaging: FCM not configured")
	}
	fcmBody := map[string]interface{}{
		"to": token,
		"notification": map[string]string{
			"title": title,
			"body":  body,
		},
	}
	if len(data) > 0 {
		fcmBody["data"] = data
	}
	payload, _ := json.Marshal(fcmBody)
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
// Providers (per-project configured senders)
// ---------------------------------------------------------------------------

// Provider is a project-scoped messaging provider configuration.
type Provider struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"projectId"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`     // email, sms, push
	Provider  string          `json:"provider"` // smtp, mailgun, sendgrid, resend, twilio, vonage, msg91, fcm, apns
	Config    json.RawMessage `json:"config"`
	Enabled   bool            `json:"enabled"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
}

// CreateProvider persists a new messaging provider for a project.
func (s *Service) CreateProvider(ctx context.Context, projectID, name, typ, provider string, config json.RawMessage) (*Provider, error) {
	id := uid.New("unique()")
	if config == nil {
		config = json.RawMessage("{}")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO msg_providers (id, project_id, name, type, provider, config, enabled) VALUES ($1,$2,$3,$4,$5,$6,true)`,
		id, projectID, name, typ, provider, string(config))
	if err != nil {
		return nil, fmt.Errorf("messaging: create provider: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return &Provider{
		ID: id, ProjectID: projectID, Name: name, Type: typ,
		Provider: provider, Config: config, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// ListProviders returns all providers for a project.
func (s *Service) ListProviders(ctx context.Context, projectID string) ([]*Provider, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, type, provider, config, enabled,
		        to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		        to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM msg_providers WHERE project_id=$1 ORDER BY created_at ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("messaging: list providers: %w", err)
	}
	defer rows.Close()
	var providers []*Provider
	for rows.Next() {
		p := &Provider{}
		var cfgStr string
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Name, &p.Type, &p.Provider,
			&cfgStr, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Config = json.RawMessage(cfgStr)
		providers = append(providers, p)
	}
	if providers == nil {
		providers = []*Provider{}
	}
	return providers, rows.Err()
}

// GetProvider returns a single provider by ID.
func (s *Service) GetProvider(ctx context.Context, projectID, id string) (*Provider, error) {
	p := &Provider{}
	var cfgStr string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, type, provider, config, enabled,
		        to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		        to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM msg_providers WHERE id=$1 AND project_id=$2`, id, projectID).
		Scan(&p.ID, &p.ProjectID, &p.Name, &p.Type, &p.Provider,
			&cfgStr, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("messaging: provider not found")
	}
	p.Config = json.RawMessage(cfgStr)
	return p, nil
}

// UpdateProvider updates a provider's name, config, and enabled state.
func (s *Service) UpdateProvider(ctx context.Context, projectID, id, name string, config json.RawMessage, enabled bool) (*Provider, error) {
	if config == nil {
		config = json.RawMessage("{}")
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE msg_providers SET name=$1, config=$2, enabled=$3, updated_at=NOW() WHERE id=$4 AND project_id=$5`,
		name, string(config), enabled, id, projectID)
	if err != nil {
		return nil, fmt.Errorf("messaging: update provider: %w", err)
	}
	return s.GetProvider(ctx, projectID, id)
}

// DeleteProvider removes a provider.
func (s *Service) DeleteProvider(ctx context.Context, projectID, id string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM msg_providers WHERE id=$1 AND project_id=$2", id, projectID)
	return err
}

// getEnabledProvider returns the first enabled provider of the given type for a project.
func (s *Service) getEnabledProvider(ctx context.Context, projectID, typ string) (*Provider, error) {
	p := &Provider{}
	var cfgStr string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, type, provider, config, enabled,
		        to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		        to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM msg_providers WHERE project_id=$1 AND type=$2 AND enabled=true
		 ORDER BY created_at ASC LIMIT 1`, projectID, typ).
		Scan(&p.ID, &p.ProjectID, &p.Name, &p.Type, &p.Provider,
			&cfgStr, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, nil // no provider configured — caller falls back to global
	}
	p.Config = json.RawMessage(cfgStr)
	return p, nil
}

// SendEmailForProject sends an email using the project's configured email provider.
// Falls back to the global SMTP if no project provider is configured.
func (s *Service) SendEmailForProject(ctx context.Context, projectID string, to []string, subject, htmlBody string) error {
	if len(to) > maxEmailRecipients {
		return ErrTooManyRecipients
	}
	p, _ := s.getEnabledProvider(ctx, projectID, "email")
	if p == nil {
		return s.SendEmail(ctx, to, subject, htmlBody)
	}
	switch p.Provider {
	case "smtp":
		return s.sendEmailViaSMTPConfig(ctx, to, subject, htmlBody, p.Config)
	case "mailgun":
		return s.sendEmailViaMailgunConfig(ctx, to, subject, htmlBody, p.Config)
	case "sendgrid":
		return s.sendEmailViaSendgridConfig(ctx, to, subject, htmlBody, p.Config)
	case "resend":
		return s.sendEmailViaResendConfig(ctx, to, subject, htmlBody, p.Config)
	default:
		return s.SendEmail(ctx, to, subject, htmlBody)
	}
}

// sendEmailViaSMTPConfig sends via a provider-specific SMTP config.
func (s *Service) sendEmailViaSMTPConfig(ctx context.Context, to []string, subject, htmlBody string, raw json.RawMessage) error {
	var cfg struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.Host == "" {
		return fmt.Errorf("messaging: invalid SMTP provider config")
	}
	if err := validateHeaderAddresses(to); err != nil {
		return err
	}
	port := cfg.Port
	if port == "" {
		port = "587"
	}
	addr := cfg.Host + ":" + port
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	headers := []string{
		fmt.Sprintf("From: %s", stripHeaderNewlines(cfg.From)),
		fmt.Sprintf("To: %s", stripHeaderNewlines(strings.Join(to, ", "))),
		fmt.Sprintf("Subject: %s", stripHeaderNewlines(subject)),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}
	msg := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + htmlBody)
	return smtp.SendMail(addr, auth, cfg.From, to, msg)
}

// sendEmailViaMailgunConfig sends via a provider-specific Mailgun config.
func (s *Service) sendEmailViaMailgunConfig(ctx context.Context, to []string, subject, htmlBody string, raw json.RawMessage) error {
	var cfg struct {
		APIKey      string `json:"apiKey"`
		Domain      string `json:"domain"`
		EURegion    bool   `json:"euRegion"`
		SenderEmail string `json:"senderEmail"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.APIKey == "" || cfg.Domain == "" {
		return fmt.Errorf("messaging: invalid Mailgun provider config")
	}
	base := "https://api.mailgun.net"
	if cfg.EURegion {
		base = "https://api.eu.mailgun.net"
	}
	apiURL := fmt.Sprintf("%s/v3/%s/messages", base, cfg.Domain)
	form := url.Values{}
	form.Set("from", cfg.SenderEmail)
	form.Set("to", strings.Join(to, ", "))
	form.Set("subject", subject)
	form.Set("html", htmlBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("messaging: create mailgun request: %w", err)
	}
	req.SetBasicAuth("api", cfg.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.doRequest(req, "mailgun")
}

// sendEmailViaSendgridConfig sends via a provider-specific SendGrid config.
func (s *Service) sendEmailViaSendgridConfig(ctx context.Context, to []string, subject, htmlBody string, raw json.RawMessage) error {
	var cfg struct {
		APIKey      string `json:"apiKey"`
		SenderEmail string `json:"senderEmail"`
		SenderName  string `json:"senderName"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.APIKey == "" {
		return fmt.Errorf("messaging: invalid SendGrid provider config")
	}
	toList := make([]map[string]string, 0, len(to))
	for _, addr := range to {
		toList = append(toList, map[string]string{"email": addr})
	}
	from := map[string]string{"email": cfg.SenderEmail}
	if cfg.SenderName != "" {
		from["name"] = cfg.SenderName
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"personalizations": []map[string]interface{}{{"to": toList}},
		"from":             from,
		"subject":          subject,
		"content":          []map[string]string{{"type": "text/html", "value": htmlBody}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("messaging: create sendgrid request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	return s.doRequest(req, "sendgrid")
}

// sendEmailViaResendConfig sends via a provider-specific Resend config.
func (s *Service) sendEmailViaResendConfig(ctx context.Context, to []string, subject, htmlBody string, raw json.RawMessage) error {
	var cfg struct {
		APIKey      string `json:"apiKey"`
		SenderEmail string `json:"senderEmail"`
		SenderName  string `json:"senderName"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.APIKey == "" {
		return fmt.Errorf("messaging: invalid Resend provider config")
	}
	from := cfg.SenderEmail
	if cfg.SenderName != "" {
		from = cfg.SenderName + " <" + cfg.SenderEmail + ">"
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"from": from, "to": to, "subject": subject, "html": htmlBody,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("messaging: create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	return s.doRequest(req, "resend")
}

// SendSMSForProject sends an SMS using the project's configured SMS provider.
// Falls back to the global Twilio config if no project provider is configured.
func (s *Service) SendSMSForProject(ctx context.Context, projectID, to, body string) error {
	p, _ := s.getEnabledProvider(ctx, projectID, "sms")
	if p == nil {
		return s.SendSMS(ctx, to, body)
	}
	switch p.Provider {
	case "twilio":
		return s.sendSMSViaTwilioConfig(ctx, to, body, p.Config)
	case "vonage":
		return s.sendSMSViaVonageConfig(ctx, to, body, p.Config)
	case "msg91":
		return s.sendSMSViaMSG91Config(ctx, to, body, p.Config)
	default:
		return s.SendSMS(ctx, to, body)
	}
}

func (s *Service) sendSMSViaTwilioConfig(ctx context.Context, to, body string, raw json.RawMessage) error {
	var cfg struct {
		SID   string `json:"sid"`
		Token string `json:"token"`
		From  string `json:"from"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.SID == "" {
		return fmt.Errorf("messaging: invalid Twilio provider config")
	}
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", cfg.SID)
	form := url.Values{}
	form.Set("From", cfg.From)
	form.Set("To", to)
	form.Set("Body", body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("messaging: create twilio request: %w", err)
	}
	req.SetBasicAuth(cfg.SID, cfg.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.doRequest(req, "twilio")
}

func (s *Service) sendSMSViaVonageConfig(ctx context.Context, to, body string, raw json.RawMessage) error {
	var cfg struct {
		APIKey    string `json:"apiKey"`
		APISecret string `json:"apiSecret"`
		From      string `json:"from"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.APIKey == "" {
		return fmt.Errorf("messaging: invalid Vonage provider config")
	}
	payload, _ := json.Marshal(map[string]string{
		"api_key": cfg.APIKey, "api_secret": cfg.APISecret,
		"from": cfg.From, "to": to, "text": body,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://rest.nexmo.com/sms/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("messaging: create vonage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return s.doRequest(req, "vonage")
}

func (s *Service) sendSMSViaMSG91Config(ctx context.Context, to, body string, raw json.RawMessage) error {
	var cfg struct {
		AuthKey  string `json:"authKey"`
		SenderID string `json:"senderId"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.AuthKey == "" {
		return fmt.Errorf("messaging: invalid MSG91 provider config")
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"sender": cfg.SenderID, "route": "4", "country": "91",
		"sms": []map[string]interface{}{{"message": body, "to": []string{to}}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://control.msg91.com/api/v5/flow/", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("messaging: create msg91 request: %w", err)
	}
	req.Header.Set("authkey", cfg.AuthKey)
	req.Header.Set("Content-Type", "application/json")
	return s.doRequest(req, "msg91")
}

// SendPushForProject sends a push notification using the project's configured push provider.
// Falls back to global FCM if no project provider is configured.
func (s *Service) SendPushForProject(ctx context.Context, projectID, token, title, body string, data map[string]string) error {
	p, _ := s.getEnabledProvider(ctx, projectID, "push")
	if p == nil {
		return s.SendPush(ctx, token, title, body, data)
	}
	switch p.Provider {
	case "fcm":
		return s.sendPushViaFCMConfig(ctx, token, title, body, data, p.Config)
	default:
		return s.SendPush(ctx, token, title, body, data)
	}
}

// SendSMSMulti sends the same SMS body to each recipient, resolving the
// project's provider per send. It attempts every recipient and returns the
// first error, mirroring how a multi-recipient email reports failure.
func (s *Service) SendSMSMulti(ctx context.Context, projectID string, to []string, body string) error {
	if len(to) > maxSMSRecipients {
		return ErrTooManyRecipients
	}
	var firstErr error
	for _, num := range to {
		if err := s.SendSMSForProject(ctx, projectID, num, body); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// SendPushMulti sends the same push notification to each recipient token,
// forwarding the optional data payload, and returns the first error.
func (s *Service) SendPushMulti(ctx context.Context, projectID string, tokens []string, title, body string, data map[string]string) error {
	if len(tokens) > maxPushRecipients {
		return ErrTooManyRecipients
	}
	var firstErr error
	for _, token := range tokens {
		if err := s.SendPushForProject(ctx, projectID, token, title, body, data); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *Service) sendPushViaFCMConfig(ctx context.Context, token, title, body string, data map[string]string, raw json.RawMessage) error {
	var cfg struct {
		ServerKey string `json:"serverKey"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.ServerKey == "" {
		return fmt.Errorf("messaging: invalid FCM provider config")
	}
	fcmBody := map[string]interface{}{
		"to":           token,
		"notification": map[string]string{"title": title, "body": body},
	}
	if len(data) > 0 {
		fcmBody["data"] = data
	}
	payload, _ := json.Marshal(fcmBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fcm.googleapis.com/fcm/send", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("messaging: create fcm request: %w", err)
	}
	req.Header.Set("Authorization", "key="+cfg.ServerKey)
	req.Header.Set("Content-Type", "application/json")
	return s.doRequest(req, "fcm")
}

// ---------------------------------------------------------------------------
// Topics (DB-backed)
// ---------------------------------------------------------------------------

// CreateTopic persists a new topic.
func (s *Service) CreateTopic(ctx context.Context, projectID, name string) (*Topic, error) {
	id := uid.New("unique()")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO msg_topics (id, project_id, name) VALUES ($1,$2,$3)`,
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
		 WHERE t.project_id = $1
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
		 WHERE t.id=$1 AND t.project_id=$2
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
		"SELECT COUNT(*) FROM msg_topics WHERE id=$1 AND project_id=$2",
		topicID, projectID).Scan(&count); err != nil || count == 0 {
		return nil, fmt.Errorf("messaging: topic not found")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO msg_topic_subscribers (topic_id, target) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
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
			sendErr = s.SendSMSForProject(ctx, projectID, sub, body)
		case strings.Contains(sub, "@"):
			sendErr = s.SendEmailForProject(ctx, projectID, []string{sub}, subject, body)
		default:
			sendErr = s.SendPushForProject(ctx, projectID, sub, subject, body, nil)
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

// ---------------------------------------------------------------------------
// Message Templates
// ---------------------------------------------------------------------------

// Template is a reusable message template with {{variable}} placeholders.
type Template struct {
	ID        string    `json:"$id"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // email, sms, push
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Variables []string  `json:"variables"`
	CreatedAt time.Time `json:"$createdAt"`
	UpdatedAt time.Time `json:"$updatedAt"`
}

// CreateTemplate persists a new message template.
func (s *Service) CreateTemplate(ctx context.Context, projectID, templateID, name, typ, subject, body string, variables []string) (*Template, error) {
	id := uid.New(templateID)
	now := time.Now().UTC()
	if variables == nil {
		variables = []string{}
	}
	varJSON, _ := json.Marshal(variables)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO message_templates (id, project_id, name, type, subject, body, variables, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, projectID, name, typ, subject, body, string(varJSON), now, now)
	if err != nil {
		return nil, fmt.Errorf("messaging: create template: %w", err)
	}
	return &Template{
		ID: id, ProjectID: projectID, Name: name, Type: typ,
		Subject: subject, Body: body, Variables: variables,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// GetTemplate fetches a single template by ID.
func (s *Service) GetTemplate(ctx context.Context, templateID, projectID string) (*Template, error) {
	var t Template
	var varJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, type, subject, body, variables, created_at, updated_at
		 FROM message_templates WHERE id = $1 AND project_id = $2`,
		templateID, projectID).
		Scan(&t.ID, &t.ProjectID, &t.Name, &t.Type, &t.Subject, &t.Body, &varJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(varJSON), &t.Variables) //nolint:errcheck
	if t.Variables == nil {
		t.Variables = []string{}
	}
	return &t, nil
}

// ListTemplates returns all templates for a project.
func (s *Service) ListTemplates(ctx context.Context, projectID string) ([]*Template, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, type, subject, body, variables, created_at, updated_at
		 FROM message_templates WHERE project_id = $1 ORDER BY created_at DESC`,
		projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var templates []*Template
	for rows.Next() {
		var t Template
		var varJSON string
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Name, &t.Type, &t.Subject, &t.Body, &varJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal([]byte(varJSON), &t.Variables) //nolint:errcheck
		if t.Variables == nil {
			t.Variables = []string{}
		}
		templates = append(templates, &t)
	}
	if templates == nil {
		templates = []*Template{}
	}
	return templates, len(templates), nil
}

// UpdateTemplate updates an existing template.
func (s *Service) UpdateTemplate(ctx context.Context, templateID, projectID, name, typ, subject, body string, variables []string) (*Template, error) {
	now := time.Now().UTC()
	if variables == nil {
		variables = []string{}
	}
	varJSON, _ := json.Marshal(variables)
	_, err := s.db.ExecContext(ctx,
		`UPDATE message_templates SET name=$1, type=$2, subject=$3, body=$4, variables=$5, updated_at=$6
		 WHERE id=$7 AND project_id=$8`,
		name, typ, subject, body, string(varJSON), now, templateID, projectID)
	if err != nil {
		return nil, fmt.Errorf("messaging: update template: %w", err)
	}
	return s.GetTemplate(ctx, templateID, projectID)
}

// DeleteTemplate removes a template.
func (s *Service) DeleteTemplate(ctx context.Context, templateID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM message_templates WHERE id=$1 AND project_id=$2",
		templateID, projectID)
	return err
}

// renderTemplate replaces {{key}} placeholders with values from the provided map.
func renderTemplate(body string, vars map[string]string) string {
	for k, v := range vars {
		body = strings.ReplaceAll(body, "{{"+k+"}}", v)
	}
	return body
}

// SendTemplate renders a template with variables and sends the message.
func (s *Service) SendTemplate(ctx context.Context, templateID, projectID string, to []string, variables map[string]string) error {
	t, err := s.GetTemplate(ctx, templateID, projectID)
	if err != nil {
		return fmt.Errorf("messaging: template not found: %w", err)
	}
	subject := renderTemplate(t.Subject, variables)
	body := renderTemplate(t.Body, variables)
	switch t.Type {
	case "email":
		if err := s.SendEmail(ctx, to, subject, body); err != nil {
			return err
		}
	case "sms":
		for _, num := range to {
			if err := s.SendSMS(ctx, num, body); err != nil {
				return err
			}
		}
	case "push":
		for _, token := range to {
			if err := s.SendPush(ctx, token, subject, body, nil); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("messaging: unknown template type %q", t.Type)
	}
	return nil
}
