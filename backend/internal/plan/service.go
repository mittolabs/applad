// Package plan holds the work a project has decided to do.
//
// An item is intent: a decision, in a sentence somebody would say out loud.
// What the software must then *do* is a specification, and whether it does it
// is a test — separate objects, pointed at rather than contained, so that a
// specification outlives the item that prompted it. The item is closed and
// forgotten; the behaviour it asked for is permanent.
package plan

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Statuses an item can be in. Kept closed: a status nobody can name is a
// status nobody can filter by.
// in_review is built-but-not-accepted: waiting on somebody's judgement. It is
// deliberately not "waiting for the tests to pass" — that is what a criterion
// carrying a specification reference already says, and it is answered by
// running them rather than by moving a card.
var Statuses = []string{"todo", "in_progress", "in_review", "blocked", "done", "cancelled"}

// Priorities, coarsest first.
var Priorities = []string{"low", "medium", "high", "urgent"}

// Item is one piece of work.
type Item struct {
	ID        string `json:"$id"`
	ProjectID string `json:"projectId"`
	ParentID  string `json:"parentId,omitempty"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Status    string `json:"status"`
	Priority  string `json:"priority"`
	// Kind separates a change from a defect: whether the system already
	// claimed to do this. Not a taxonomy of ticket types — two values with
	// one real difference.
	Kind        string     `json:"kind"`
	MilestoneID string     `json:"milestoneId,omitempty"`
	TargetDate  *time.Time `json:"targetDate,omitempty"`
	AssigneeID  string     `json:"assigneeId,omitempty"`
	Labels      []string   `json:"labels"`
	Rank        int64      `json:"rank"`
	CreatedBy   string     `json:"createdBy,omitempty"`
	ClosedAt    *time.Time `json:"closedAt,omitempty"`
	CreatedAt   time.Time  `json:"$createdAt"`
	UpdatedAt   time.Time  `json:"$updatedAt"`
	Links       []Link     `json:"links"`
	// Counted alongside the item so a list can say how much of what was
	// agreed has become behaviour, without asking per item.
	CriteriaCount     int `json:"criteriaCount"`
	CriteriaSpecified int `json:"criteriaSpecified"`
}

// Link is something an item points at.
type Link struct {
	ID        string    `json:"$id"`
	ItemID    string    `json:"itemId"`
	Kind      string    `json:"kind"`
	Ref       string    `json:"ref"`
	Label     string    `json:"label,omitempty"`
	CreatedAt time.Time `json:"$createdAt"`
}

// Service is the plan's data access.
type Service struct{ db *db.DB }

// NewService creates a plan service.
func NewService(database *db.DB) *Service { return &Service{db: database} }

// Input is what a caller supplies to create or change an item.
//
// Pointers where absence and emptiness differ: clearing a body and not
// mentioning it are different requests, and an update that cannot tell them
// apart silently erases whatever it was not told about.
type Input struct {
	Title       *string
	Body        *string
	Status      *string
	Priority    *string
	AssigneeID  *string
	ParentID    *string
	Kind        *string
	MilestoneID *string
	TargetDate  *time.Time
	Labels      *[]string
	Rank        *int64
}

// Validate reports what is wrong with an input, in the caller's terms.
func (in Input) Validate() error {
	if in.Title != nil && strings.TrimSpace(*in.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if in.Status != nil && !contains(Statuses, *in.Status) {
		return fmt.Errorf("status must be one of: %s", strings.Join(Statuses, ", "))
	}
	if in.Priority != nil && !contains(Priorities, *in.Priority) {
		return fmt.Errorf("priority must be one of: %s", strings.Join(Priorities, ", "))
	}
	if in.Kind != nil && !contains(Kinds, *in.Kind) {
		return fmt.Errorf("kind must be one of: %s", strings.Join(Kinds, ", "))
	}
	return nil
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// Closed reports whether a status means the work is over, either way.
func Closed(status string) bool { return status == "done" || status == "cancelled" }

// Create records a new item at the end of the list.
func (s *Service) Create(ctx context.Context, projectID, createdBy string, in Input) (*Item, error) {
	if in.Title == nil {
		return nil, fmt.Errorf("title is required")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}

	item := &Item{
		ID: uid.New("unique()"), ProjectID: projectID, Title: strings.TrimSpace(*in.Title),
		Status: "todo", Priority: "medium", Kind: "change", Labels: []string{}, CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
	}
	item.UpdatedAt = item.CreatedAt
	applyInput(item, in)

	// Sparse ranks, so an item can later be dropped between two others
	// without renumbering everything below it.
	var maxRank sql.NullInt64
	s.db.QueryRowContext(ctx, //nolint:errcheck
		`SELECT MAX(rank) FROM plan_items WHERE project_id = $1`, projectID).Scan(&maxRank)
	item.Rank = maxRank.Int64 + 1000

	labels, _ := json.Marshal(item.Labels)
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO plan_items (id, project_id, parent_id, title, body, status, priority, kind,
		    milestone_id, target_date, assignee_id, labels, rank, created_by, created_at, updated_at)
		 VALUES ($1, $2, NULLIF($3,''), $4, $5, $6, $7, $8, NULLIF($9,''), $10, NULLIF($11,''), $12, $13, NULLIF($14,''), $15, $15)`,
		item.ID, projectID, item.ParentID, item.Title, item.Body, item.Status, item.Priority, item.Kind,
		item.MilestoneID, item.TargetDate, item.AssigneeID, labels, item.Rank, createdBy, item.CreatedAt); err != nil {
		return nil, fmt.Errorf("plan: create item: %w", err)
	}
	return item, nil
}

// Update changes only what was named.
func (s *Service) Update(ctx context.Context, id, projectID string, in Input) (*Item, error) {
	return s.UpdateAs(ctx, id, projectID, "", in)
}

// UpdateAs is Update, attributing the change to somebody.
func (s *Service) UpdateAs(ctx context.Context, id, projectID, actorID string, in Input) (*Item, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	item, err := s.Get(ctx, id, projectID)
	if err != nil {
		return nil, err
	}
	before := *item

	wasClosed := Closed(item.Status)
	applyInput(item, in)

	// Closing stamps the time; reopening clears it, so a reopened item does
	// not carry a date saying it finished.
	nowClosed := Closed(item.Status)
	switch {
	case nowClosed && !wasClosed:
		now := time.Now().UTC()
		item.ClosedAt = &now
	case !nowClosed && wasClosed:
		item.ClosedAt = nil
	}

	labels, _ := json.Marshal(item.Labels)
	if _, err := s.db.ExecContext(ctx,
		`UPDATE plan_items SET parent_id = NULLIF($1,''), title = $2, body = $3, status = $4,
		     priority = $5, kind = $6, milestone_id = NULLIF($7,''), target_date = $8,
		     assignee_id = NULLIF($9,''), labels = $10, rank = $11, closed_at = $12,
		     updated_at = NOW()
		   WHERE id = $13 AND project_id = $14`,
		item.ParentID, item.Title, item.Body, item.Status, item.Priority, item.Kind,
		item.MilestoneID, item.TargetDate, item.AssigneeID, labels, item.Rank,
		item.ClosedAt, id, projectID); err != nil {
		return nil, fmt.Errorf("plan: update item: %w", err)
	}
	s.recordChanges(ctx, id, actorID, &before, item)
	return s.Get(ctx, id, projectID)
}

func applyInput(item *Item, in Input) {
	if in.Title != nil {
		item.Title = strings.TrimSpace(*in.Title)
	}
	if in.Body != nil {
		item.Body = *in.Body
	}
	if in.Status != nil {
		item.Status = *in.Status
	}
	if in.Priority != nil {
		item.Priority = *in.Priority
	}
	if in.AssigneeID != nil {
		item.AssigneeID = *in.AssigneeID
	}
	if in.ParentID != nil {
		item.ParentID = *in.ParentID
	}
	if in.Labels != nil {
		item.Labels = *in.Labels
	}
	if in.Rank != nil {
		item.Rank = *in.Rank
	}
	if in.Kind != nil {
		item.Kind = *in.Kind
	}
	if in.MilestoneID != nil {
		item.MilestoneID = *in.MilestoneID
	}
	if in.TargetDate != nil {
		item.TargetDate = in.TargetDate
	}
	if item.Labels == nil {
		item.Labels = []string{}
	}
}

// Filter narrows a listing.
type Filter struct {
	Status      string
	Assignee    string
	Label       string
	Search      string
	ParentID    string
	MilestoneID string
	// IncludeClosed keeps done and cancelled work in the list. Off by
	// default: a backlog that shows everything ever finished is not a backlog.
	IncludeClosed bool
}

// List returns a project's items, hand-ordered.
func (s *Service) List(ctx context.Context, projectID string, f Filter) ([]*Item, error) {
	query := `SELECT i.id, i.project_id, COALESCE(i.parent_id,''), i.title, i.body, i.status, i.priority, i.kind,
	                 COALESCE(i.milestone_id,''), i.target_date,
	                 COALESCE(i.assignee_id,''), i.labels, i.rank, COALESCE(i.created_by,''),
	                 i.closed_at, i.created_at, i.updated_at,
	                 (SELECT COUNT(*) FROM plan_criteria c WHERE c.item_id = i.id),
	                 (SELECT COUNT(*) FROM plan_criteria c WHERE c.item_id = i.id AND c.spec_ref <> '')
	            FROM plan_items i WHERE i.project_id = $1`
	args := []interface{}{projectID}

	add := func(clause string, value interface{}) {
		args = append(args, value)
		query += fmt.Sprintf(" AND %s $%d", clause, len(args))
	}
	if f.Status != "" {
		add("i.status =", f.Status)
	} else if !f.IncludeClosed {
		query += " AND i.status NOT IN ('done','cancelled')"
	}
	if f.Assignee != "" {
		add("i.assignee_id =", f.Assignee)
	}
	if f.ParentID != "" {
		add("i.parent_id =", f.ParentID)
	}
	if f.MilestoneID != "" {
		add("i.milestone_id =", f.MilestoneID)
	}
	if f.Label != "" {
		add("i.labels @>", fmt.Sprintf("[%q]", f.Label))
	}
	if f.Search != "" {
		args = append(args, "%"+strings.ToLower(f.Search)+"%")
		query += fmt.Sprintf(" AND (LOWER(i.title) LIKE $%d OR LOWER(i.body) LIKE $%d)", len(args), len(args))
	}
	query += " ORDER BY i.rank ASC, i.created_at ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []*Item{}
	byID := map[string]*Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		byID[item.ID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, s.attachLinks(ctx, projectID, byID)
}

// Get returns one item with everything it points at.
func (s *Service) Get(ctx context.Context, id, projectID string) (*Item, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT i.id, i.project_id, COALESCE(i.parent_id,''), i.title, i.body, i.status, i.priority, i.kind,
		        COALESCE(i.milestone_id,''), i.target_date,
		        COALESCE(i.assignee_id,''), i.labels, i.rank, COALESCE(i.created_by,''),
		        i.closed_at, i.created_at, i.updated_at,
		        (SELECT COUNT(*) FROM plan_criteria c WHERE c.item_id = i.id),
		        (SELECT COUNT(*) FROM plan_criteria c WHERE c.item_id = i.id AND c.spec_ref <> '')
		   FROM plan_items i WHERE i.id = $1 AND i.project_id = $2`, id, projectID)

	item, err := scanItem(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plan item not found")
	}
	if err != nil {
		return nil, err
	}
	return item, s.attachLinks(ctx, projectID, map[string]*Item{item.ID: item})
}

// Delete removes an item. Its children survive, orphaned rather than deleted:
// the sub-tasks of an abandoned item are usually still work.
func (s *Service) Delete(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM plan_items WHERE id = $1 AND project_id = $2`, id, projectID)
	return err
}

// AddLink points an item at something.
func (s *Service) AddLink(ctx context.Context, itemID, projectID, kind, ref, label string) (*Link, error) {
	if _, err := s.Get(ctx, itemID, projectID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(kind) == "" || strings.TrimSpace(ref) == "" {
		return nil, fmt.Errorf("kind and ref are required")
	}

	link := &Link{ID: uid.New("unique()"), ItemID: itemID, Kind: kind, Ref: ref,
		Label: label, CreatedAt: time.Now().UTC()}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO plan_item_links (id, item_id, kind, ref, label, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (item_id, kind, ref) DO NOTHING`,
		link.ID, itemID, kind, ref, label, link.CreatedAt); err != nil {
		return nil, fmt.Errorf("plan: add link: %w", err)
	}
	return link, nil
}

// RemoveLink unpoints an item.
func (s *Service) RemoveLink(ctx context.Context, linkID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM plan_item_links l USING plan_items i
		  WHERE l.id = $1 AND l.item_id = i.id AND i.project_id = $2`, linkID, projectID)
	return err
}

// attachLinks fills in what each item points at, in one query rather than one
// per item.
func (s *Service) attachLinks(ctx context.Context, projectID string, byID map[string]*Item) error {
	for _, item := range byID {
		item.Links = []Link{}
	}
	if len(byID) == 0 {
		return nil
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT l.id, l.item_id, l.kind, l.ref, l.label, l.created_at
		   FROM plan_item_links l JOIN plan_items i ON i.id = l.item_id
		  WHERE i.project_id = $1 ORDER BY l.created_at ASC`, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ID, &l.ItemID, &l.Kind, &l.Ref, &l.Label, &l.CreatedAt); err != nil {
			return err
		}
		if item, ok := byID[l.ItemID]; ok {
			item.Links = append(item.Links, l)
		}
	}
	return rows.Err()
}

// scanner is what both QueryRow and Rows satisfy.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanItem(row scanner) (*Item, error) {
	var item Item
	var labels []byte
	var closedAt sql.NullTime
	var targetDate sql.NullTime
	if err := row.Scan(&item.ID, &item.ProjectID, &item.ParentID, &item.Title, &item.Body,
		&item.Status, &item.Priority, &item.Kind, &item.MilestoneID, &targetDate,
		&item.AssigneeID, &labels, &item.Rank,
		&item.CreatedBy, &closedAt, &item.CreatedAt, &item.UpdatedAt,
		&item.CriteriaCount, &item.CriteriaSpecified); err != nil {
		return nil, err
	}
	json.Unmarshal(labels, &item.Labels) //nolint:errcheck
	if item.Labels == nil {
		item.Labels = []string{}
	}
	if closedAt.Valid {
		item.ClosedAt = &closedAt.Time
	}
	if targetDate.Valid {
		item.TargetDate = &targetDate.Time
	}
	return &item, nil
}
