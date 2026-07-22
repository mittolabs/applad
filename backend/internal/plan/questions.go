package plan

import (
	"context"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/uid"
)

/*
 * Scoring a dimension by asking things people can actually answer.
 *
 * "What is the impact of this — low, medium or high?" is the guess the matrix
 * was meant to remove, asked one level up: nobody knows what medium impact
 * means and two people never mean the same thing by it. "Is there a
 * workaround?" has an answer.
 *
 * So each dimension is scored. Options carry scores, the scores sum, and a
 * band turns the sum into a level; the level feeds the grid. One option may
 * force the top of the scale on its own, because data loss should not have to
 * out-argue three other answers.
 *
 * The questions are seeded, not fixed. What counts as impact in a product
 * team is not what counts in a support desk, and neither is a fact about the
 * world.
 */

// Question is one thing asked about an item.
type Question struct {
	ID        string   `json:"$id"`
	Dimension string   `json:"dimension"`
	Text      string   `json:"text"`
	Help      string   `json:"help,omitempty"`
	Position  int      `json:"position"`
	Options   []Option `json:"options"`
	// AnsweredWith is the option chosen for the item being asked about, when
	// the questions are fetched for one.
	AnsweredWith string `json:"answeredWith,omitempty"`
}

// Option is one answer, and what it is worth.
type Option struct {
	ID        string `json:"$id"`
	Label     string `json:"label"`
	Score     int    `json:"score"`
	ForcesTop bool   `json:"forcesTop"`
	Position  int    `json:"position"`
}

// Band maps a score to a level.
type Band struct {
	Dimension string `json:"dimension"`
	Level     int    `json:"level"`
	MinScore  int    `json:"minScore"`
	MaxScore  int    `json:"maxScore"`
}

type seedQuestion struct {
	dimension, text, help string
	options               []seedOption
}

type seedOption struct {
	label     string
	score     int
	forcesTop bool
}

// defaultQuestions are calibrated for a team building its own product, where
// nobody is filing a ticket and the work competes only with other work.
var defaultQuestions = []seedQuestion{
	{"impact", "Does anything work badly for users until this exists?", "", []seedOption{
		{"No — it is an improvement", 0, false},
		{"Yes — something is worse than it should be", 2, false},
		{"Yes — something is broken", 4, false},
	}},
	{"impact", "How many people does it affect?", "", []seedOption{
		{"A few", 1, false},
		{"Many", 2, false},
		{"Everyone", 3, false},
	}},
	{"impact", "Does other planned work depend on it?", "", []seedOption{
		{"No", 0, false},
		{"Yes", 2, false},
	}},
	{"impact", "Is data at risk of being lost or wrong?", "Decides the priority on its own.", []seedOption{
		{"No", 0, false},
		{"Yes", 5, true},
	}},

	{"urgency", "Is anybody blocked right now?", "", []seedOption{
		{"No", 0, false},
		{"Yes", 3, false},
	}},
	{"urgency", "Is there a workaround?", "No workaround is what makes something urgent rather than merely important.", []seedOption{
		{"Yes, an easy one", 0, false},
		{"Yes, an awkward one", 2, false},
		{"No", 4, false},
	}},
	{"urgency", "Is it tied to a date somebody outside the team is expecting?", "", []seedOption{
		{"No", 0, false},
		{"Yes", 3, false},
	}},
}

// defaultBands turn a summed score into a level. Deliberately generous at the
// bottom: most work is not high impact, and a scale where everything lands
// high is the scale we started with.
var defaultBands = []Band{
	{"", 1, 0, 3},
	{"", 2, 4, 7},
	{"", 3, 8, 999},
}

// Questions returns a project's questions, seeding the defaults on first ask.
//
// When itemID is given, each question also carries the answer recorded for
// that item.
func (s *Service) Questions(ctx context.Context, projectID, itemID string) ([]Question, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, dimension, text, help, position FROM plan_priority_questions
		  WHERE project_id = $1 ORDER BY dimension ASC, position ASC`, projectID)
	if err != nil {
		return nil, err
	}

	var questions []Question
	byID := map[string]int{}
	for rows.Next() {
		var q Question
		if err := rows.Scan(&q.ID, &q.Dimension, &q.Text, &q.Help, &q.Position); err != nil {
			rows.Close()
			return nil, err
		}
		q.Options = []Option{}
		byID[q.ID] = len(questions)
		questions = append(questions, q)
	}
	rows.Close()

	if len(questions) == 0 {
		if err := s.seedQuestions(ctx, projectID); err != nil {
			return nil, err
		}
		return s.Questions(ctx, projectID, itemID)
	}

	opts, err := s.db.QueryContext(ctx,
		`SELECT o.id, o.question_id, o.label, o.score, o.forces_top, o.position
		   FROM plan_priority_options o JOIN plan_priority_questions q ON q.id = o.question_id
		  WHERE q.project_id = $1 ORDER BY o.position ASC`, projectID)
	if err != nil {
		return nil, err
	}
	for opts.Next() {
		var o Option
		var qid string
		if err := opts.Scan(&o.ID, &qid, &o.Label, &o.Score, &o.ForcesTop, &o.Position); err != nil {
			opts.Close()
			return nil, err
		}
		if i, ok := byID[qid]; ok {
			questions[i].Options = append(questions[i].Options, o)
		}
	}
	opts.Close()

	if itemID != "" {
		answers, err := s.db.QueryContext(ctx,
			`SELECT question_id, option_id FROM plan_priority_answers WHERE item_id = $1`, itemID)
		if err != nil {
			return nil, err
		}
		defer answers.Close()
		for answers.Next() {
			var qid, oid string
			if err := answers.Scan(&qid, &oid); err != nil {
				return nil, err
			}
			if i, ok := byID[qid]; ok {
				questions[i].AnsweredWith = oid
			}
		}
	}
	return questions, nil
}

func (s *Service) seedQuestions(ctx context.Context, projectID string) error {
	now := time.Now().UTC()
	for i, q := range defaultQuestions {
		qid := uid.New("unique()")
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO plan_priority_questions (id, project_id, dimension, text, help, position, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			qid, projectID, q.dimension, q.text, q.help, i, now); err != nil {
			return fmt.Errorf("plan: seed questions: %w", err)
		}
		for j, o := range q.options {
			if _, err := s.db.ExecContext(ctx,
				`INSERT INTO plan_priority_options (id, question_id, label, score, forces_top, position)
				 VALUES ($1,$2,$3,$4,$5,$6)`,
				uid.New("unique()"), qid, o.label, o.score, o.forcesTop, j); err != nil {
				return fmt.Errorf("plan: seed options: %w", err)
			}
		}
	}

	for _, dimension := range []string{"impact", "urgency"} {
		for _, b := range defaultBands {
			if _, err := s.db.ExecContext(ctx,
				`INSERT INTO plan_priority_bands (id, project_id, dimension, level, min_score, max_score)
				 VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (project_id, dimension, level) DO NOTHING`,
				uid.New("unique()"), projectID, dimension, b.Level, b.MinScore, b.MaxScore); err != nil {
				return fmt.Errorf("plan: seed bands: %w", err)
			}
		}
	}
	return nil
}

// Answer records one answer and re-derives the item's priority from
// everything answered so far.
//
// Re-derived on every answer rather than at the end: a half-answered
// assessment still says something, and making somebody finish before seeing
// anything is how a form gets abandoned.
func (s *Service) Answer(ctx context.Context, itemID, projectID, actorID, questionID, optionID string) (*Item, error) {
	if _, err := s.Get(ctx, itemID, projectID); err != nil {
		return nil, err
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO plan_priority_answers (item_id, question_id, option_id, answered_at)
		 VALUES ($1,$2,$3,NOW())
		 ON CONFLICT (item_id, question_id) DO UPDATE SET option_id = $3, answered_at = NOW()`,
		itemID, questionID, optionID); err != nil {
		return nil, fmt.Errorf("plan: record answer: %w", err)
	}
	return s.scoreAndRate(ctx, itemID, projectID, actorID)
}

// scoreAndRate sums what has been answered and rates the item accordingly.
func (s *Service) scoreAndRate(ctx context.Context, itemID, projectID, actorID string) (*Item, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT q.dimension, o.score, o.forces_top
		   FROM plan_priority_answers a
		   JOIN plan_priority_questions q ON q.id = a.question_id
		   JOIN plan_priority_options   o ON o.id = a.option_id
		  WHERE a.item_id = $1`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scores := map[string]int{}
	forced := false
	for rows.Next() {
		var dimension string
		var score int
		var forcesTop bool
		if err := rows.Scan(&dimension, &score, &forcesTop); err != nil {
			return nil, err
		}
		scores[dimension] += score
		if forcesTop {
			forced = true
		}
	}

	impact, err := s.levelFor(ctx, projectID, "impact", scores["impact"])
	if err != nil {
		return nil, err
	}
	urgency, err := s.levelFor(ctx, projectID, "urgency", scores["urgency"])
	if err != nil {
		return nil, err
	}
	// An answer that decides on its own decides on its own.
	if forced {
		impact, urgency = LevelHigh, LevelHigh
	}
	return s.Rate(ctx, itemID, projectID, actorID, impact, urgency)
}

// levelFor turns a score into a level using the project's bands.
func (s *Service) levelFor(ctx context.Context, projectID, dimension string, score int) (int, error) {
	var level int
	err := s.db.QueryRowContext(ctx,
		`SELECT level FROM plan_priority_bands
		  WHERE project_id = $1 AND dimension = $2 AND $3 BETWEEN min_score AND max_score
		  ORDER BY level DESC LIMIT 1`, projectID, dimension, score).Scan(&level)
	if err != nil || level == 0 {
		// No band matched, which means nothing has been answered for this
		// dimension. The lowest level is the honest reading of no evidence.
		return LevelLow, nil //nolint:nilerr
	}
	return level, nil
}
