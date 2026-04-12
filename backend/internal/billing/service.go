// Package billing provides usage metering, billing events, plan/subscription
// management, and invoice generation (Stripe-ready).
package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"encoding/json"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Plan defines a billing plan with limits and features.
type Plan struct {
	ID            string                 `json:"$id"`
	Name          string                 `json:"name"`
	Slug          string                 `json:"slug"`
	PriceMonthly  int                    `json:"priceMonthly"` // cents
	PriceYearly   int                    `json:"priceYearly"`  // cents
	Limits        map[string]interface{} `json:"limits"`
	Features      []string               `json:"features"`
	Active        bool                   `json:"active"`
	CreatedAt     time.Time              `json:"$createdAt"`
	UpdatedAt     time.Time              `json:"$updatedAt"`
}

// Subscription links a project to a plan.
type Subscription struct {
	ID                    string     `json:"$id"`
	ProjectID             string     `json:"projectId"`
	PlanID                string     `json:"planId"`
	Status                string     `json:"status"` // active | past_due | cancelled | trialing
	StripeCustomerID      string     `json:"stripeCustomerId,omitempty"`
	StripeSubscriptionID  string     `json:"stripeSubscriptionId,omitempty"`
	CurrentPeriodStart    *time.Time `json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd      *time.Time `json:"currentPeriodEnd,omitempty"`
	CancelAtPeriodEnd     bool       `json:"cancelAtPeriodEnd"`
	CreatedAt             time.Time  `json:"$createdAt"`
	UpdatedAt             time.Time  `json:"$updatedAt"`
}

// BillingEvent is a metered usage event.
type BillingEvent struct {
	ID        string                 `json:"$id"`
	ProjectID string                 `json:"projectId"`
	EventType string                 `json:"eventType"`
	Quantity  int64                  `json:"quantity"`
	Unit      string                 `json:"unit"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"$createdAt"`
}

// Invoice is a billing invoice.
type Invoice struct {
	ID               string     `json:"$id"`
	ProjectID        string     `json:"projectId"`
	SubscriptionID   string     `json:"subscriptionId,omitempty"`
	AmountCents      int        `json:"amountCents"`
	Currency         string     `json:"currency"`
	Status           string     `json:"status"` // draft | open | paid | void
	StripeInvoiceID  string     `json:"stripeInvoiceId,omitempty"`
	PeriodStart      *time.Time `json:"periodStart,omitempty"`
	PeriodEnd        *time.Time `json:"periodEnd,omitempty"`
	PaidAt           *time.Time `json:"paidAt,omitempty"`
	CreatedAt        time.Time  `json:"$createdAt"`
}

// UsageSummary is aggregated usage for a project over a period.
type UsageSummary struct {
	ProjectID string                    `json:"projectId"`
	Period    string                    `json:"period"`
	Events    map[string]int64          `json:"events"`
	Total     int64                     `json:"total"`
}

// Service manages billing data.
type Service struct {
	db *db.DB
}

// NewService creates a new billing Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// ── Plans ─────────────────────────────────────────────────────────────────────

// ListPlans returns all active billing plans.
func (s *Service) ListPlans(ctx context.Context) ([]*Plan, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, slug, price_monthly, price_yearly, limits, COALESCE(features,'[]'), active, created_at, updated_at
		 FROM billing_plans WHERE active = TRUE ORDER BY price_monthly ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// CreatePlan creates a new billing plan (admin only).
func (s *Service) CreatePlan(ctx context.Context, name, slug string, priceMonthly, priceYearly int, limits map[string]interface{}, features []string) (*Plan, error) {
	p := &Plan{
		ID: uid.New(""), Name: name, Slug: slug,
		PriceMonthly: priceMonthly, PriceYearly: priceYearly,
		Limits: limits, Features: features, Active: true,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	limitsJSON, _ := json.Marshal(limits)
	featuresJSON, _ := json.Marshal(features)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO billing_plans (id, name, slug, price_monthly, price_yearly, limits, features, active, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,TRUE,$8,$9)`,
		p.ID, p.Name, p.Slug, p.PriceMonthly, p.PriceYearly,
		nullBytes(limitsJSON), featuresJSON, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, fmt.Errorf("billing: plan %q already exists", slug)
		}
		return nil, fmt.Errorf("billing: create plan: %w", err)
	}
	return p, nil
}

// ── Subscriptions ─────────────────────────────────────────────────────────────

// GetSubscription returns the current subscription for a project.
func (s *Service) GetSubscription(ctx context.Context, projectID string) (*Subscription, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, plan_id, status,
		        COALESCE(stripe_customer_id,''), COALESCE(stripe_subscription_id,''),
		        current_period_start, current_period_end, cancel_at_period_end, created_at, updated_at
		 FROM billing_subscriptions WHERE project_id = $1`, projectID)
	return scanSubscription(row)
}

// Subscribe subscribes a project to a plan.
func (s *Service) Subscribe(ctx context.Context, projectID, planID string) (*Subscription, error) {
	sub := &Subscription{
		ID: uid.New(""), ProjectID: projectID, PlanID: planID,
		Status: "active",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	now := time.Now().UTC()
	periodEnd := now.AddDate(0, 1, 0) // 1 month
	sub.CurrentPeriodStart = &now
	sub.CurrentPeriodEnd = &periodEnd
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO billing_subscriptions (id, project_id, plan_id, status, current_period_start, current_period_end, cancel_at_period_end, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,FALSE,$7,$8)
		 ON CONFLICT (project_id) DO UPDATE SET plan_id=EXCLUDED.plan_id, status='active', current_period_start=EXCLUDED.current_period_start, current_period_end=EXCLUDED.current_period_end, updated_at=EXCLUDED.updated_at`,
		sub.ID, sub.ProjectID, sub.PlanID, sub.Status,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CreatedAt, sub.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("billing: subscribe: %w", err)
	}
	return sub, nil
}

// CancelSubscription schedules a subscription to cancel at period end.
func (s *Service) CancelSubscription(ctx context.Context, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE billing_subscriptions SET cancel_at_period_end=TRUE, updated_at=$1 WHERE project_id=$2",
		time.Now().UTC(), projectID)
	return err
}

// ── Metering ──────────────────────────────────────────────────────────────────

// RecordEvent records a billing/metering event.
func (s *Service) RecordEvent(ctx context.Context, projectID, eventType string, quantity int64, unit string, metadata map[string]interface{}) (*BillingEvent, error) {
	e := &BillingEvent{
		ID: uid.New(""), ProjectID: projectID, EventType: eventType,
		Quantity: quantity, Unit: unit, Metadata: metadata,
		CreatedAt: time.Now().UTC(),
	}
	metaJSON, _ := json.Marshal(metadata)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO billing_events (id, project_id, event_type, quantity, unit, metadata, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.ID, e.ProjectID, e.EventType, e.Quantity, e.Unit, nullBytes(metaJSON), e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("billing: record event: %w", err)
	}
	return e, nil
}

// GetUsageSummary returns aggregated usage for a project for the current billing period.
func (s *Service) GetUsageSummary(ctx context.Context, projectID string, from, to time.Time) (*UsageSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT event_type, SUM(quantity) FROM billing_events
		 WHERE project_id = $1 AND created_at BETWEEN $2 AND $3
		 GROUP BY event_type`, projectID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	summary := &UsageSummary{
		ProjectID: projectID,
		Period:    from.Format("2006-01-02") + "/" + to.Format("2006-01-02"),
		Events:    make(map[string]int64),
	}
	var total int64
	for rows.Next() {
		var et string
		var qty int64
		if err := rows.Scan(&et, &qty); err != nil {
			return nil, err
		}
		summary.Events[et] = qty
		total += qty
	}
	summary.Total = total
	return summary, nil
}

// ── Invoices ──────────────────────────────────────────────────────────────────

// ListInvoices returns invoices for a project.
func (s *Service) ListInvoices(ctx context.Context, projectID string, limit, offset int) ([]*Invoice, int, error) {
	if limit <= 0 {
		limit = 20
	}
	var total int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invoices WHERE project_id=$1", projectID).Scan(&total) //nolint:errcheck
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, COALESCE(subscription_id,''), amount_cents, currency, status,
		        COALESCE(stripe_invoice_id,''), period_start, period_end, paid_at, created_at
		 FROM invoices WHERE project_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		projectID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, inv)
	}
	return out, total, nil
}

// GetInvoice returns a single invoice by ID for a project.
func (s *Service) GetInvoice(ctx context.Context, projectID, invoiceID string) (*Invoice, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, COALESCE(subscription_id,''), amount_cents, currency, status,
		        COALESCE(stripe_invoice_id,''), period_start, period_end, paid_at, created_at
		 FROM invoices WHERE project_id=$1 AND id=$2`,
		projectID, invoiceID)
	return scanInvoice(row)
}

// ── scanners ──────────────────────────────────────────────────────────────────

func scanPlan(row interface{ Scan(...interface{}) error }) (*Plan, error) {
	p := &Plan{}
	var limitsRaw, featuresRaw []byte
	if err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.PriceMonthly, &p.PriceYearly,
		&limitsRaw, &featuresRaw, &p.Active, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal(limitsRaw, &p.Limits)     //nolint:errcheck
	json.Unmarshal(featuresRaw, &p.Features) //nolint:errcheck
	return p, nil
}

func scanSubscription(row interface{ Scan(...interface{}) error }) (*Subscription, error) {
	s := &Subscription{}
	if err := row.Scan(&s.ID, &s.ProjectID, &s.PlanID, &s.Status,
		&s.StripeCustomerID, &s.StripeSubscriptionID,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelAtPeriodEnd,
		&s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	return s, nil
}

func scanInvoice(row interface{ Scan(...interface{}) error }) (*Invoice, error) {
	inv := &Invoice{}
	if err := row.Scan(&inv.ID, &inv.ProjectID, &inv.SubscriptionID, &inv.AmountCents,
		&inv.Currency, &inv.Status, &inv.StripeInvoiceID,
		&inv.PeriodStart, &inv.PeriodEnd, &inv.PaidAt, &inv.CreatedAt); err != nil {
		return nil, err
	}
	return inv, nil
}

func nullBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
