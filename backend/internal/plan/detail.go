package plan

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/uid"
)

/*
 * What an item carries, and what contains it.
 *
 * Acceptance criteria are the constraints the work was agreed against. They
 * are written before anybody has decided how the behaviour will be expressed,
 * which is why they live here rather than in a specification: a Rule is a
 * criterion made executable, and it cannot be written until the criterion
 * exists to derive it from. A criterion carrying no rule is visibly still an
 * intention.
 *
 * Milestones give the roadmap its time axis. Their progress is counted from
 * the items in them rather than typed by anyone — a percentage nobody
 * measured is exactly the kind of number this codebase keeps removing.
 */

// Kinds an item can be. Two values, one real distinction: whether the system
// already claimed to do this.
var Kinds = []string{"change", "defect"}

// Criterion is one thing that has to be true for an item to be done.
type Criterion struct {
	ID        string    `json:"$id"`
	ItemID    string    `json:"itemId"`
	Text      string    `json:"text"`
	SpecRef   string    `json:"specRef"`
	Met       bool      `json:"met"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"$createdAt"`
}

// Specified reports whether this criterion has become executable behaviour.
func (c Criterion) Specified() bool { return strings.TrimSpace(c.SpecRef) != "" }

// Comment is one thing somebody said about an item.
type Comment struct {
	ID        string    `json:"$id"`
	ItemID    string    `json:"itemId"`
	AuthorID  string    `json:"authorId,omitempty"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"$createdAt"`
}

// Milestone is a dated container of work.
type Milestone struct {
	ID          string     `json:"$id"`
	ProjectID   string     `json:"projectId"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	TargetDate  *time.Time `json:"targetDate,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	CreatedAt   time.Time  `json:"$createdAt"`
	UpdatedAt   time.Time  `json:"$updatedAt"`
	// Progress, counted from the items in it. Never stored: a stored
	// percentage is out of date the moment anything moves.
	Progress Progress `json:"progress"`
}

// Progress is what a milestone's items add up to.
type Progress struct {
	Total      int `json:"total"`
	Done       int `json:"done"`
	InProgress int `json:"inProgress"`
	Blocked    int `json:"blocked"`
	// Criteria across those items, and how many have become executable
	// behaviour. "Six of ten items done" and "half the agreed constraints are
	// still unspecified" are different facts, and a roadmap that shows only
	// the first is the fiction every roadmap tool ships.
	Criteria  int `json:"criteria"`
	Specified int `json:"specified"`
}

// ── Criteria ──

// ListCriteria returns an item's acceptance criteria, in order.
func (s *Service) ListCriteria(ctx context.Context, itemID string) ([]Criterion, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, item_id, text, spec_ref, met, position, created_at
		   FROM plan_criteria WHERE item_id = $1 ORDER BY position ASC, created_at ASC`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Criterion{}
	for rows.Next() {
		var c Criterion
		if err := rows.Scan(&c.ID, &c.ItemID, &c.Text, &c.SpecRef, &c.Met, &c.Position, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AddCriterion records a constraint the work is agreed against.
func (s *Service) AddCriterion(ctx context.Context, itemID, projectID, text string) (*Criterion, error) {
	if _, err := s.Get(ctx, itemID, projectID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("a criterion needs to say something")
	}

	var maxPos sql.NullInt64
	s.db.QueryRowContext(ctx, //nolint:errcheck
		`SELECT MAX(position) FROM plan_criteria WHERE item_id = $1`, itemID).Scan(&maxPos)

	c := &Criterion{ID: uid.New("unique()"), ItemID: itemID, Text: strings.TrimSpace(text),
		Position: int(maxPos.Int64) + 1, CreatedAt: time.Now().UTC()}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO plan_criteria (id, item_id, text, position, created_at) VALUES ($1,$2,$3,$4,$5)`,
		c.ID, itemID, c.Text, c.Position, c.CreatedAt); err != nil {
		return nil, fmt.Errorf("plan: add criterion: %w", err)
	}
	return c, nil
}

// UpdateCriterion changes a criterion's text, its rule, or whether it holds.
func (s *Service) UpdateCriterion(ctx context.Context, id, projectID string, text, specRef *string, met *bool) error {
	sets := []string{}
	args := []interface{}{}
	add := func(clause string, v interface{}) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", clause, len(args)))
	}
	if text != nil {
		add("text", strings.TrimSpace(*text))
	}
	if specRef != nil {
		add("spec_ref", *specRef)
	}
	if met != nil {
		add("met", *met)
	}
	if len(sets) == 0 {
		return nil
	}

	args = append(args, id, projectID)
	query := fmt.Sprintf(
		`UPDATE plan_criteria c SET %s FROM plan_items i
		  WHERE c.id = $%d AND c.item_id = i.id AND i.project_id = $%d`,
		strings.Join(sets, ", "), len(args)-1, len(args))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// DeleteCriterion removes a criterion.
func (s *Service) DeleteCriterion(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM plan_criteria c USING plan_items i
		  WHERE c.id = $1 AND c.item_id = i.id AND i.project_id = $2`, id, projectID)
	return err
}

// ── Comments ──

// ListComments returns an item's discussion, oldest first.
func (s *Service) ListComments(ctx context.Context, itemID string) ([]Comment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, item_id, COALESCE(author_id,''), body, created_at
		   FROM plan_comments WHERE item_id = $1 ORDER BY created_at ASC`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ItemID, &c.AuthorID, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AddComment records something somebody said.
func (s *Service) AddComment(ctx context.Context, itemID, projectID, authorID, body string) (*Comment, error) {
	if _, err := s.Get(ctx, itemID, projectID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("a comment needs to say something")
	}

	c := &Comment{ID: uid.New("unique()"), ItemID: itemID, AuthorID: authorID,
		Body: strings.TrimSpace(body), CreatedAt: time.Now().UTC()}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO plan_comments (id, item_id, author_id, body, created_at)
		 VALUES ($1,$2,NULLIF($3,''),$4,$5)`,
		c.ID, itemID, authorID, c.Body, c.CreatedAt); err != nil {
		return nil, fmt.Errorf("plan: add comment: %w", err)
	}
	return c, nil
}

// DeleteComment removes a comment.
func (s *Service) DeleteComment(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM plan_comments c USING plan_items i
		  WHERE c.id = $1 AND c.item_id = i.id AND i.project_id = $2`, id, projectID)
	return err
}

// ── Milestones ──

// ListMilestones returns a project's milestones by target date, each with its
// progress counted from the items in it.
func (s *Service) ListMilestones(ctx context.Context, projectID string) ([]*Milestone, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, description, target_date, completed_at, created_at, updated_at
		   FROM plan_milestones WHERE project_id = $1
		  ORDER BY target_date ASC NULLS LAST, created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*Milestone{}
	for rows.Next() {
		m, err := scanMilestone(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, s.countProgress(ctx, projectID, out)
}

// CreateMilestone adds a dated container of work.
func (s *Service) CreateMilestone(ctx context.Context, projectID, name, description string, target *time.Time) (*Milestone, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	m := &Milestone{ID: uid.New("unique()"), ProjectID: projectID,
		Name: strings.TrimSpace(name), Description: description, TargetDate: target,
		CreatedAt: time.Now().UTC()}
	m.UpdatedAt = m.CreatedAt

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO plan_milestones (id, project_id, name, description, target_date, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$6)`,
		m.ID, projectID, m.Name, m.Description, m.TargetDate, m.CreatedAt); err != nil {
		return nil, fmt.Errorf("plan: create milestone: %w", err)
	}
	return m, nil
}

// UpdateMilestone changes a milestone.
func (s *Service) UpdateMilestone(ctx context.Context, id, projectID, name, description string, target *time.Time, completed *bool) error {
	var completedAt interface{}
	if completed != nil && *completed {
		completedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE plan_milestones
		    SET name = COALESCE(NULLIF($1,''), name),
		        description = $2, target_date = $3,
		        completed_at = CASE WHEN $4::boolean IS NULL THEN completed_at
		                            WHEN $4 THEN $5 ELSE NULL END,
		        updated_at = NOW()
		  WHERE id = $6 AND project_id = $7`,
		name, description, target, completed, completedAt, id, projectID)
	return err
}

// DeleteMilestone removes a milestone. Its items survive, unassigned: work
// outlives the date somebody hoped to finish it by.
func (s *Service) DeleteMilestone(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM plan_milestones WHERE id = $1 AND project_id = $2`, id, projectID)
	return err
}

// countProgress fills in what each milestone's items add up to, in two
// queries rather than two per milestone.
func (s *Service) countProgress(ctx context.Context, projectID string, milestones []*Milestone) error {
	if len(milestones) == 0 {
		return nil
	}
	byID := map[string]*Milestone{}
	for _, m := range milestones {
		byID[m.ID] = m
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT milestone_id, status, COUNT(*) FROM plan_items
		  WHERE project_id = $1 AND milestone_id IS NOT NULL
		  GROUP BY milestone_id, status`, projectID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, status string
		var n int
		if err := rows.Scan(&id, &status, &n); err != nil {
			rows.Close()
			return err
		}
		m, ok := byID[id]
		if !ok {
			continue
		}
		m.Progress.Total += n
		switch status {
		case "done":
			m.Progress.Done += n
		case "in_progress":
			m.Progress.InProgress += n
		case "blocked":
			m.Progress.Blocked += n
		}
	}
	rows.Close()

	// How much of what was agreed has become behaviour anything can check.
	crit, err := s.db.QueryContext(ctx,
		`SELECT i.milestone_id, COUNT(*), COUNT(*) FILTER (WHERE c.spec_ref <> '')
		   FROM plan_criteria c JOIN plan_items i ON i.id = c.item_id
		  WHERE i.project_id = $1 AND i.milestone_id IS NOT NULL
		  GROUP BY i.milestone_id`, projectID)
	if err != nil {
		return err
	}
	defer crit.Close()

	for crit.Next() {
		var id string
		var total, specified int
		if err := crit.Scan(&id, &total, &specified); err != nil {
			return err
		}
		if m, ok := byID[id]; ok {
			m.Progress.Criteria, m.Progress.Specified = total, specified
		}
	}
	return crit.Err()
}

func scanMilestone(row scanner) (*Milestone, error) {
	var m Milestone
	var target, completed sql.NullTime
	if err := row.Scan(&m.ID, &m.ProjectID, &m.Name, &m.Description,
		&target, &completed, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	if target.Valid {
		m.TargetDate = &target.Time
	}
	if completed.Valid {
		m.CompletedAt = &completed.Time
	}
	return &m, nil
}
