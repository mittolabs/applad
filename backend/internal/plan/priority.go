package plan

import (
	"context"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/uid"
)

/*
 * Priority, derived from two answers rather than picked from a list.
 *
 * A single field inflates: nothing costs anything to call urgent, so
 * everything becomes urgent and the word stops sorting anything. It also
 * averages two facts that move independently — how much this matters, and how
 * soon it is needed.
 *
 * Impact is a property of the problem; urgency is a property of the calendar.
 * High impact never falls below high because a deadline is soft: you may defer
 * it, you may not demote it. The default grid encodes that asymmetry, and a
 * project can edit it, because a shop where every item comes from a paying
 * client calibrates differently from one building its own product.
 */

// Levels an axis can take. Three, because a person can hold three apart and
// argue about them; five invites a middle nobody means.
const (
	LevelLow    = 1
	LevelMedium = 2
	LevelHigh   = 3
)

// Cell is one square of the grid.
type Cell struct {
	Kind     string `json:"kind"`
	Impact   int    `json:"impact"`
	Urgency  int    `json:"urgency"`
	Priority string `json:"priority"`
}

// DefaultGrid is the seed, not the law.
//
//	             urgency:  low       medium     high
//	impact high         |  high      high       urgent
//	impact medium       |  medium    medium     high
//	impact low          |  low       medium     medium
//
// Low survives only when both answers are low — it matters little and nobody
// is waiting. Everything else is at least medium, and impact dominates: a
// high-impact item is never below high, whatever the calendar says.
var DefaultGrid = map[[2]int]string{
	{3, 3}: "urgent", {3, 2}: "high", {3, 1}: "high",
	{2, 3}: "high", {2, 2}: "medium", {2, 1}: "medium",
	{1, 3}: "medium", {1, 2}: "medium", {1, 1}: "low",
}

// LevelName is how a level reads to a person.
var LevelName = map[int]string{LevelLow: "low", LevelMedium: "medium", LevelHigh: "high"}

// Grid returns a project's matrix for one kind, seeding the default the first
// time it is asked for.
//
// Seeded on read rather than on project creation: a project that never opens
// the matrix never needs rows, and one created before this existed would
// otherwise have none.
func (s *Service) Grid(ctx context.Context, projectID, kind string) ([]Cell, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT kind, impact, urgency, priority FROM plan_priority_matrix
		  WHERE project_id = $1 AND kind = $2
		  ORDER BY impact DESC, urgency ASC`, projectID, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cells := []Cell{}
	for rows.Next() {
		var c Cell
		if err := rows.Scan(&c.Kind, &c.Impact, &c.Urgency, &c.Priority); err != nil {
			return nil, err
		}
		cells = append(cells, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cells) == 9 {
		return cells, nil
	}
	return s.seedGrid(ctx, projectID, kind)
}

func (s *Service) seedGrid(ctx context.Context, projectID, kind string) ([]Cell, error) {
	now := time.Now().UTC()
	for impact := LevelHigh; impact >= LevelLow; impact-- {
		for urgency := LevelLow; urgency <= LevelHigh; urgency++ {
			if _, err := s.db.ExecContext(ctx,
				`INSERT INTO plan_priority_matrix (id, project_id, kind, impact, urgency, priority, created_at, updated_at)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$7)
				 ON CONFLICT (project_id, kind, impact, urgency) DO NOTHING`,
				uid.New("unique()"), projectID, kind, impact, urgency,
				DefaultGrid[[2]int{impact, urgency}], now); err != nil {
				return nil, fmt.Errorf("plan: seed grid: %w", err)
			}
		}
	}

	cells := make([]Cell, 0, 9)
	for impact := LevelHigh; impact >= LevelLow; impact-- {
		for urgency := LevelLow; urgency <= LevelHigh; urgency++ {
			cells = append(cells, Cell{Kind: kind, Impact: impact, Urgency: urgency,
				Priority: DefaultGrid[[2]int{impact, urgency}]})
		}
	}
	return cells, nil
}

// SetCell changes what one square resolves to, and re-derives every item that
// square decides.
//
// Editing the grid without re-deriving would leave items showing a priority
// their own answers no longer produce — which is the same class of lie as a
// stored percentage.
func (s *Service) SetCell(ctx context.Context, projectID, kind string, impact, urgency int, priority string) error {
	if !contains(Priorities, priority) {
		return fmt.Errorf("priority must be one of: %v", Priorities)
	}
	if impact < LevelLow || impact > LevelHigh || urgency < LevelLow || urgency > LevelHigh {
		return fmt.Errorf("impact and urgency are 1, 2 or 3")
	}
	if _, err := s.Grid(ctx, projectID, kind); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx,
		`UPDATE plan_priority_matrix SET priority = $1, updated_at = NOW()
		  WHERE project_id = $2 AND kind = $3 AND impact = $4 AND urgency = $5`,
		priority, projectID, kind, impact, urgency); err != nil {
		return err
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE plan_items SET priority = $1, updated_at = NOW()
		  WHERE project_id = $2 AND kind = $3 AND priority_is_manual = FALSE
		    AND priority_impact = $4 AND priority_urgency = $5`,
		priority, projectID, kind, impact, urgency)
	return err
}

// derivePriority answers what the grid says for one pair of answers.
func (s *Service) derivePriority(ctx context.Context, projectID, kind string, impact, urgency int) (string, error) {
	cells, err := s.Grid(ctx, projectID, kind)
	if err != nil {
		return "", err
	}
	for _, c := range cells {
		if c.Impact == impact && c.Urgency == urgency {
			return c.Priority, nil
		}
	}
	return DefaultGrid[[2]int{impact, urgency}], nil
}

// Rate records how much an item matters and how soon it is needed, and takes
// its priority from the grid.
//
// Rating is what makes a priority derived; setting one by hand is still
// allowed and is recorded as such, because an override made deliberately is
// worth keeping and worth being able to see.
func (s *Service) Rate(ctx context.Context, id, projectID, actorID string, impact, urgency int) (*Item, error) {
	item, err := s.Get(ctx, id, projectID)
	if err != nil {
		return nil, err
	}
	if impact < LevelLow || impact > LevelHigh || urgency < LevelLow || urgency > LevelHigh {
		return nil, fmt.Errorf("impact and urgency are 1 (low), 2 (medium) or 3 (high)")
	}

	priority, err := s.derivePriority(ctx, projectID, item.Kind, impact, urgency)
	if err != nil {
		return nil, err
	}

	before := *item
	if _, err := s.db.ExecContext(ctx,
		`UPDATE plan_items SET priority_impact = $1, priority_urgency = $2,
		     priority_is_manual = FALSE, priority = $3, updated_at = NOW()
		   WHERE id = $4 AND project_id = $5`,
		impact, urgency, priority, id, projectID); err != nil {
		return nil, fmt.Errorf("plan: rate item: %w", err)
	}

	after, err := s.Get(ctx, id, projectID)
	if err != nil {
		return nil, err
	}
	s.recordChanges(ctx, id, actorID, &before, after)
	return after, nil
}
