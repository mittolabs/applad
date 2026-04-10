// Package webhooks implements outbound webhook delivery with HMAC-SHA256 signing.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Webhook represents an outbound webhook subscription.
type Webhook struct {
	ID        string    `json:"$id"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"$createdAt"`
	UpdatedAt time.Time `json:"$updatedAt"`
}

// WebhookDelivery represents a single delivery attempt for a webhook event.
type WebhookDelivery struct {
	ID         string    `json:"$id"`
	WebhookID  string    `json:"webhookId"`
	Event      string    `json:"event"`
	Payload    string    `json:"payload"`
	StatusCode int       `json:"statusCode"`
	Response   string    `json:"response"`
	Attempts   int       `json:"attempts"`
	Success    bool      `json:"success"`
	CreatedAt  time.Time `json:"$createdAt"`
}

// Service handles webhook business logic.
type Service struct {
	db     *db.DB
	client *http.Client
}

// NewService creates a new webhooks Service.
func NewService(database *db.DB) *Service {
	return &Service{
		db: database,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Create creates a new webhook.
func (s *Service) Create(ctx context.Context, projectID, name, url string, events []string, secret string, enabled bool) (*Webhook, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()

	if secret == "" {
		secret = uid.RandomHex(32)
	}

	eventsJSON, _ := json.Marshal(events)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO webhooks (id, project_id, name, url, events, secret, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, projectID, name, url, eventsJSON, secret, enabled, now, now)
	if err != nil {
		return nil, fmt.Errorf("webhooks: create: %w", err)
	}
	return &Webhook{
		ID: id, ProjectID: projectID, Name: name, URL: url,
		Events: events, Secret: secret, Enabled: enabled,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// List returns all webhooks for a project.
func (s *Service) List(ctx context.Context, projectID string) ([]*Webhook, int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, name, url, events, secret, enabled, created_at, updated_at FROM webhooks WHERE project_id = ? ORDER BY created_at DESC",
		projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var webhooks []*Webhook
	for rows.Next() {
		w, err := scanWebhook(rows)
		if err != nil {
			return nil, 0, err
		}
		webhooks = append(webhooks, w)
	}
	if webhooks == nil {
		webhooks = []*Webhook{}
	}
	return webhooks, len(webhooks), nil
}

// Get returns a webhook by ID.
func (s *Service) Get(ctx context.Context, id, projectID string) (*Webhook, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, project_id, name, url, events, secret, enabled, created_at, updated_at FROM webhooks WHERE id = ? AND project_id = ?",
		id, projectID)
	var w Webhook
	var eventsJSON []byte
	err := row.Scan(&w.ID, &w.ProjectID, &w.Name, &w.URL, &eventsJSON, &w.Secret, &w.Enabled, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("webhook not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(eventsJSON, &w.Events) //nolint:errcheck
	if w.Events == nil {
		w.Events = []string{}
	}
	return &w, nil
}

// Update updates a webhook.
func (s *Service) Update(ctx context.Context, id, projectID, name, url string, events []string, secret string, enabled bool) (*Webhook, error) {
	eventsJSON, _ := json.Marshal(events)
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		"UPDATE webhooks SET name = ?, url = ?, events = ?, secret = ?, enabled = ?, updated_at = ? WHERE id = ? AND project_id = ?",
		name, url, eventsJSON, secret, enabled, now, id, projectID)
	if err != nil {
		return nil, fmt.Errorf("webhooks: update: %w", err)
	}
	return s.Get(ctx, id, projectID)
}

// Delete removes a webhook.
func (s *Service) Delete(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM webhooks WHERE id = ? AND project_id = ?", id, projectID)
	return err
}

// ListDeliveries returns delivery history for a webhook.
func (s *Service) ListDeliveries(ctx context.Context, webhookID string) ([]*WebhookDelivery, int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, webhook_id, event, payload, status_code, response, attempts, success, created_at FROM webhook_deliveries WHERE webhook_id = ? ORDER BY created_at DESC",
		webhookID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var deliveries []*WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.Event, &d.Payload, &d.StatusCode, &d.Response, &d.Attempts, &d.Success, &d.CreatedAt); err != nil {
			return nil, 0, err
		}
		deliveries = append(deliveries, &d)
	}
	if deliveries == nil {
		deliveries = []*WebhookDelivery{}
	}
	return deliveries, len(deliveries), nil
}

// Deliver sends an event to all matching webhooks for a project.
func (s *Service) Deliver(ctx context.Context, projectID, event string, payload map[string]interface{}) error {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, name, url, events, secret, enabled, created_at, updated_at FROM webhooks WHERE project_id = ? AND enabled = true",
		projectID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var webhooks []*Webhook
	for rows.Next() {
		w, err := scanWebhook(rows)
		if err != nil {
			return err
		}
		webhooks = append(webhooks, w)
	}

	for _, w := range webhooks {
		if !matchesEvent(w.Events, event) {
			continue
		}
		go s.deliverOne(context.Background(), w, event, payload) //nolint:errcheck
	}
	return nil
}

// RetryDelivery retries a failed delivery (max 3 attempts).
func (s *Service) RetryDelivery(ctx context.Context, deliveryID string) (*WebhookDelivery, error) {
	var d WebhookDelivery
	err := s.db.QueryRowContext(ctx,
		"SELECT id, webhook_id, event, payload, status_code, response, attempts, success, created_at FROM webhook_deliveries WHERE id = ?",
		deliveryID).Scan(&d.ID, &d.WebhookID, &d.Event, &d.Payload, &d.StatusCode, &d.Response, &d.Attempts, &d.Success, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("delivery not found")
	}
	if err != nil {
		return nil, err
	}
	if d.Success {
		return &d, nil
	}
	if d.Attempts >= 3 {
		return nil, fmt.Errorf("max retry attempts (3) reached")
	}

	// Look up the webhook
	var w Webhook
	var eventsJSON []byte
	err = s.db.QueryRowContext(ctx,
		"SELECT id, project_id, name, url, events, secret, enabled, created_at, updated_at FROM webhooks WHERE id = ?",
		d.WebhookID).Scan(&w.ID, &w.ProjectID, &w.Name, &w.URL, &eventsJSON, &w.Secret, &w.Enabled, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("webhook not found for delivery")
	}
	json.Unmarshal(eventsJSON, &w.Events) //nolint:errcheck

	// Parse the stored payload
	var payload map[string]interface{}
	json.Unmarshal([]byte(d.Payload), &payload) //nolint:errcheck

	statusCode, respBody, deliverErr := s.sendHTTP(&w, d.Event, payload)
	success := deliverErr == nil && statusCode >= 200 && statusCode < 300
	d.Attempts++
	d.StatusCode = statusCode
	d.Response = respBody
	d.Success = success

	s.db.ExecContext(ctx,
		"UPDATE webhook_deliveries SET status_code = ?, response = ?, attempts = ?, success = ? WHERE id = ?",
		statusCode, respBody, d.Attempts, success, d.ID) //nolint:errcheck

	return &d, nil
}

// deliverOne sends a single webhook delivery and records the result.
func (s *Service) deliverOne(ctx context.Context, w *Webhook, event string, payload map[string]interface{}) {
	payloadJSON, _ := json.Marshal(payload)
	statusCode, respBody, err := s.sendHTTP(w, event, payload)
	success := err == nil && statusCode >= 200 && statusCode < 300

	id := uid.New("unique()")
	now := time.Now().UTC()
	s.db.ExecContext(ctx,
		"INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status_code, response, attempts, success, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, w.ID, event, string(payloadJSON), statusCode, respBody, 1, success, now) //nolint:errcheck
}

// sendHTTP performs the HTTP POST with HMAC-SHA256 signature.
func (s *Service) sendHTTP(w *Webhook, event string, payload map[string]interface{}) (int, string, error) {
	body, _ := json.Marshal(payload)

	mac := hmac.New(sha256.New, []byte(w.Secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest(http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return 0, "", fmt.Errorf("webhooks: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", event)
	req.Header.Set("User-Agent", "Applad-Webhooks/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err.Error(), err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(respBytes), nil
}

// matchesEvent checks if any subscribed event pattern matches the event.
// Supports wildcard "*" to match all events, and prefix matching with ".*" suffix.
func matchesEvent(subscribed []string, event string) bool {
	for _, s := range subscribed {
		if s == "*" || s == event {
			return true
		}
		// prefix match: "databases.*" matches "databases.rows.create"
		if len(s) > 2 && s[len(s)-2:] == ".*" {
			prefix := s[:len(s)-1] // "databases."
			if len(event) >= len(prefix) && event[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// scanWebhook scans a webhook row from a sql.Rows result.
func scanWebhook(rows *sql.Rows) (*Webhook, error) {
	var w Webhook
	var eventsJSON []byte
	if err := rows.Scan(&w.ID, &w.ProjectID, &w.Name, &w.URL, &eventsJSON, &w.Secret, &w.Enabled, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal(eventsJSON, &w.Events) //nolint:errcheck
	if w.Events == nil {
		w.Events = []string{}
	}
	return &w, nil
}
