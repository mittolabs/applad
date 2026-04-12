// Package flags implements feature flag management and evaluation.
package flags

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// ── Models ───────────────────────────────────────────────────────────────────

// Flag represents a feature flag.
type Flag struct {
	ID           string      `json:"$id"`
	ProjectID    string      `json:"projectId"`
	Key          string      `json:"key"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Type         string      `json:"type"` // boolean, string, number, json
	DefaultValue interface{} `json:"defaultValue"`
	Enabled      bool        `json:"enabled"`
	Tags         []string    `json:"tags"`
	Rules        []Rule      `json:"rules,omitempty"`
	CreatedAt    time.Time   `json:"$createdAt"`
	UpdatedAt    time.Time   `json:"$updatedAt"`
}

// Rule defines a targeting rule for a flag.
type Rule struct {
	ID         string      `json:"$id"`
	FlagID     string      `json:"flagId"`
	Priority   int         `json:"priority"`
	Type       string      `json:"type"` // user, team, label, attribute, percentage, segment
	Conditions []Condition `json:"conditions"`
	Value      interface{} `json:"value"`
	RolloutPct int         `json:"rolloutPct"` // 0-100
	Enabled    bool        `json:"enabled"`
	CreatedAt  time.Time   `json:"$createdAt"`
}

// Condition is a single targeting condition within a rule.
type Condition struct {
	Attribute string      `json:"attribute"` // userId, teamId, email, label, custom.*
	Operator  string      `json:"operator"`  // eq, neq, contains, starts_with, ends_with, in, not_in, gt, lt, regex
	Value     interface{} `json:"value"`
}

// Override is a per-user or per-team flag override.
type Override struct {
	ID         string      `json:"$id"`
	FlagID     string      `json:"flagId"`
	TargetType string      `json:"targetType"` // user, team
	TargetID   string      `json:"targetId"`
	Value      interface{} `json:"value"`
	CreatedAt  time.Time   `json:"$createdAt"`
}

// EvalContext is the context passed during flag evaluation.
type EvalContext struct {
	UserID     string                 `json:"userId"`
	TeamIDs    []string               `json:"teamIds"`
	Email      string                 `json:"email"`
	Labels     []string               `json:"labels"`
	Attributes map[string]interface{} `json:"attributes"` // custom attributes
}

// EvalResult is the result of evaluating a flag.
type EvalResult struct {
	Key     string      `json:"key"`
	Value   interface{} `json:"value"`
	Enabled bool        `json:"enabled"`
	RuleID  string      `json:"ruleId,omitempty"`
	Reason  string      `json:"reason"` // default, rule, override, disabled, not_found
}

// ── Service ──────────────────────────────────────────────────────────────────

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// ── Flag CRUD ────────────────────────────────────────────────────────────────

func (s *Service) CreateFlag(ctx context.Context, projectID, key, name, description, flagType string, defaultValue interface{}, tags []string) (*Flag, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	if flagType == "" {
		flagType = "boolean"
	}
	if defaultValue == nil {
		defaultValue = false
	}
	defaultJSON, _ := json.Marshal(defaultValue)
	tagsJSON, _ := json.Marshal(tags)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO feature_flags (id, project_id, key_name, name, description, type, default_value, tags, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, projectID, key, name, description, flagType, defaultJSON, tagsJSON, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			return nil, fmt.Errorf("flag key '%s' already exists", key)
		}
		return nil, fmt.Errorf("flags: create: %w", err)
	}

	return s.GetFlag(ctx, id, projectID)
}

func (s *Service) GetFlag(ctx context.Context, flagID, projectID string) (*Flag, error) {
	var f Flag
	var defaultJSON, tagsJSON []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, key_name, name, description, type, default_value, enabled, tags, created_at, updated_at
		 FROM feature_flags WHERE id=$1 AND project_id=$2`, flagID, projectID).
		Scan(&f.ID, &f.ProjectID, &f.Key, &f.Name, &f.Description, &f.Type, &defaultJSON, &f.Enabled, &tagsJSON, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("flags: not found")
	}
	json.Unmarshal(defaultJSON, &f.DefaultValue)
	json.Unmarshal(tagsJSON, &f.Tags)

	// Load rules
	f.Rules, _ = s.ListRules(ctx, f.ID)
	return &f, nil
}

func (s *Service) GetFlagByKey(ctx context.Context, key, projectID string) (*Flag, error) {
	var flagID string
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM feature_flags WHERE key_name=$1 AND project_id=$2`, key, projectID).Scan(&flagID)
	if err != nil {
		return nil, fmt.Errorf("flags: key '%s' not found", key)
	}
	return s.GetFlag(ctx, flagID, projectID)
}

func (s *Service) ListFlags(ctx context.Context, projectID string) ([]*Flag, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, key_name, name, description, type, default_value, enabled, tags, created_at, updated_at
		 FROM feature_flags WHERE project_id=$1 ORDER BY key_name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []*Flag
	for rows.Next() {
		var f Flag
		var defaultJSON, tagsJSON []byte
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.Key, &f.Name, &f.Description, &f.Type, &defaultJSON, &f.Enabled, &tagsJSON, &f.CreatedAt, &f.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal(defaultJSON, &f.DefaultValue)
		json.Unmarshal(tagsJSON, &f.Tags)
		flags = append(flags, &f)
	}
	return flags, nil
}

func (s *Service) UpdateFlag(ctx context.Context, flagID, projectID, name, description string, defaultValue interface{}, enabled bool, tags []string) (*Flag, error) {
	defaultJSON, _ := json.Marshal(defaultValue)
	tagsJSON, _ := json.Marshal(tags)
	_, err := s.db.ExecContext(ctx,
		`UPDATE feature_flags SET name=$1, description=$2, default_value=$3, enabled=$4, tags=$5, updated_at=$6
		 WHERE id=$7 AND project_id=$8`,
		name, description, defaultJSON, enabled, tagsJSON, time.Now().UTC(), flagID, projectID)
	if err != nil {
		return nil, err
	}
	return s.GetFlag(ctx, flagID, projectID)
}

func (s *Service) ToggleFlag(ctx context.Context, flagID, projectID string, enabled bool) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE feature_flags SET enabled=$1, updated_at=$2 WHERE id=$3 AND project_id=$4`,
		enabled, time.Now().UTC(), flagID, projectID)
	return err
}

func (s *Service) DeleteFlag(ctx context.Context, flagID, projectID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM feature_flags WHERE id=$1 AND project_id=$2`, flagID, projectID)
	return err
}

// ── Rule CRUD ────────────────────────────────────────────────────────────────

func (s *Service) CreateRule(ctx context.Context, flagID, ruleType string, conditions []Condition, value interface{}, rolloutPct int) (*Rule, error) {
	id := uid.New("unique()")
	condJSON, _ := json.Marshal(conditions)
	valueJSON, _ := json.Marshal(value)

	// Get next priority
	var maxPriority int
	s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(priority), 0) FROM flag_rules WHERE flag_id=$1`, flagID).Scan(&maxPriority)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO flag_rules (id, flag_id, priority, type, conditions, value, rollout_pct, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, flagID, maxPriority+1, ruleType, condJSON, valueJSON, rolloutPct, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	return &Rule{
		ID: id, FlagID: flagID, Priority: maxPriority + 1, Type: ruleType,
		Conditions: conditions, Value: value, RolloutPct: rolloutPct, Enabled: true,
	}, nil
}

func (s *Service) ListRules(ctx context.Context, flagID string) ([]Rule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, flag_id, priority, type, conditions, value, rollout_pct, enabled, created_at
		 FROM flag_rules WHERE flag_id=$1 ORDER BY priority`, flagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var r Rule
		var condJSON, valueJSON []byte
		if err := rows.Scan(&r.ID, &r.FlagID, &r.Priority, &r.Type, &condJSON, &valueJSON, &r.RolloutPct, &r.Enabled, &r.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(condJSON, &r.Conditions)
		json.Unmarshal(valueJSON, &r.Value)
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *Service) DeleteRule(ctx context.Context, ruleID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM flag_rules WHERE id=$1`, ruleID)
	return err
}

// ── Override CRUD ────────────────────────────────────────────────────────────

func (s *Service) SetOverride(ctx context.Context, flagID, targetType, targetID string, value interface{}) error {
	id := uid.New("unique()")
	valueJSON, _ := json.Marshal(value)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO flag_overrides (id, flag_id, target_type, target_id, value, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (flag_id, target_type, target_id) DO UPDATE SET value=EXCLUDED.value`,
		id, flagID, targetType, targetID, valueJSON, time.Now().UTC())
	return err
}

func (s *Service) ListOverrides(ctx context.Context, flagID string) ([]Override, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, flag_id, target_type, target_id, value, created_at FROM flag_overrides WHERE flag_id=$1`, flagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var overrides []Override
	for rows.Next() {
		var o Override
		var valueJSON []byte
		rows.Scan(&o.ID, &o.FlagID, &o.TargetType, &o.TargetID, &valueJSON, &o.CreatedAt)
		json.Unmarshal(valueJSON, &o.Value)
		overrides = append(overrides, o)
	}
	return overrides, nil
}

func (s *Service) DeleteOverride(ctx context.Context, flagID, targetType, targetID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM flag_overrides WHERE flag_id=$1 AND target_type=$2 AND target_id=$3`,
		flagID, targetType, targetID)
	return err
}

// ── Evaluation Engine ────────────────────────────────────────────────────────

// Evaluate evaluates a single flag for the given context.
func (s *Service) Evaluate(ctx context.Context, projectID, key string, evalCtx *EvalContext) (*EvalResult, error) {
	flag, err := s.GetFlagByKey(ctx, key, projectID)
	if err != nil {
		return &EvalResult{Key: key, Value: nil, Enabled: false, Reason: "not_found"}, nil
	}

	if !flag.Enabled {
		return &EvalResult{Key: key, Value: flag.DefaultValue, Enabled: false, Reason: "disabled"}, nil
	}

	// Check overrides first (highest priority)
	if evalCtx != nil && evalCtx.UserID != "" {
		var valueJSON []byte
		err := s.db.QueryRowContext(ctx,
			`SELECT value FROM flag_overrides WHERE flag_id=$1 AND target_type='user' AND target_id=$2`,
			flag.ID, evalCtx.UserID).Scan(&valueJSON)
		if err == nil {
			var val interface{}
			json.Unmarshal(valueJSON, &val)
			s.recordEvaluation(ctx, flag.ID, projectID, evalCtx.UserID, val, "")
			return &EvalResult{Key: key, Value: val, Enabled: true, Reason: "override"}, nil
		}
	}

	// Check team overrides
	if evalCtx != nil {
		for _, teamID := range evalCtx.TeamIDs {
			var valueJSON []byte
			err := s.db.QueryRowContext(ctx,
				`SELECT value FROM flag_overrides WHERE flag_id=$1 AND target_type='team' AND target_id=$2`,
				flag.ID, teamID).Scan(&valueJSON)
			if err == nil {
				var val interface{}
				json.Unmarshal(valueJSON, &val)
				s.recordEvaluation(ctx, flag.ID, projectID, evalCtx.UserID, val, "")
				return &EvalResult{Key: key, Value: val, Enabled: true, Reason: "override"}, nil
			}
		}
	}

	// Evaluate rules in priority order
	for _, rule := range flag.Rules {
		if !rule.Enabled {
			continue
		}
		if s.matchesRule(rule, evalCtx) {
			// Apply rollout percentage
			if rule.RolloutPct < 100 {
				if !s.inRollout(flag.Key, evalCtx, rule.RolloutPct) {
					continue
				}
			}
			s.recordEvaluation(ctx, flag.ID, projectID, safeUserID(evalCtx), rule.Value, rule.ID)
			return &EvalResult{Key: key, Value: rule.Value, Enabled: true, RuleID: rule.ID, Reason: "rule"}, nil
		}
	}

	// Default value
	s.recordEvaluation(ctx, flag.ID, projectID, safeUserID(evalCtx), flag.DefaultValue, "")
	return &EvalResult{Key: key, Value: flag.DefaultValue, Enabled: true, Reason: "default"}, nil
}

// EvaluateAll evaluates all flags for the given context.
func (s *Service) EvaluateAll(ctx context.Context, projectID string, evalCtx *EvalContext) (map[string]*EvalResult, error) {
	flags, err := s.ListFlags(ctx, projectID)
	if err != nil {
		return nil, err
	}

	results := make(map[string]*EvalResult, len(flags))
	for _, f := range flags {
		// Load rules for each flag
		f.Rules, _ = s.ListRules(ctx, f.ID)
		result, _ := s.evaluateFlag(ctx, f, evalCtx)
		results[f.Key] = result
	}
	return results, nil
}

func (s *Service) evaluateFlag(ctx context.Context, flag *Flag, evalCtx *EvalContext) (*EvalResult, error) {
	if !flag.Enabled {
		return &EvalResult{Key: flag.Key, Value: flag.DefaultValue, Enabled: false, Reason: "disabled"}, nil
	}

	// Check overrides
	if evalCtx != nil && evalCtx.UserID != "" {
		var valueJSON []byte
		err := s.db.QueryRowContext(ctx,
			`SELECT value FROM flag_overrides WHERE flag_id=$1 AND target_type='user' AND target_id=$2`,
			flag.ID, evalCtx.UserID).Scan(&valueJSON)
		if err == nil {
			var val interface{}
			json.Unmarshal(valueJSON, &val)
			return &EvalResult{Key: flag.Key, Value: val, Enabled: true, Reason: "override"}, nil
		}
	}

	for _, rule := range flag.Rules {
		if !rule.Enabled {
			continue
		}
		if s.matchesRule(rule, evalCtx) {
			if rule.RolloutPct < 100 && !s.inRollout(flag.Key, evalCtx, rule.RolloutPct) {
				continue
			}
			return &EvalResult{Key: flag.Key, Value: rule.Value, Enabled: true, RuleID: rule.ID, Reason: "rule"}, nil
		}
	}

	return &EvalResult{Key: flag.Key, Value: flag.DefaultValue, Enabled: true, Reason: "default"}, nil
}

// ── Rule Matching ────────────────────────────────────────────────────────────

func (s *Service) matchesRule(rule Rule, evalCtx *EvalContext) bool {
	if evalCtx == nil {
		return rule.Type == "percentage" // percentage rules match everyone
	}

	// All conditions must match (AND logic)
	for _, cond := range rule.Conditions {
		if !matchCondition(cond, evalCtx) {
			return false
		}
	}
	return true
}

func matchCondition(cond Condition, ctx *EvalContext) bool {
	actual := resolveAttribute(cond.Attribute, ctx)
	expected := fmt.Sprintf("%v", cond.Value)
	actualStr := fmt.Sprintf("%v", actual)

	switch cond.Operator {
	case "eq", "==":
		return actualStr == expected
	case "neq", "!=":
		return actualStr != expected
	case "contains":
		return strings.Contains(actualStr, expected)
	case "starts_with":
		return strings.HasPrefix(actualStr, expected)
	case "ends_with":
		return strings.HasSuffix(actualStr, expected)
	case "in":
		// Value should be a comma-separated list or JSON array
		values := parseValueList(cond.Value)
		for _, v := range values {
			if actualStr == v {
				return true
			}
		}
		return false
	case "not_in":
		values := parseValueList(cond.Value)
		for _, v := range values {
			if actualStr == v {
				return false
			}
		}
		return true
	case "gt":
		return actualStr > expected
	case "lt":
		return actualStr < expected
	case "exists":
		return actualStr != "" && actualStr != "<nil>"
	case "not_exists":
		return actualStr == "" || actualStr == "<nil>"
	default:
		return actualStr == expected
	}
}

func resolveAttribute(attr string, ctx *EvalContext) interface{} {
	switch attr {
	case "userId", "user_id":
		return ctx.UserID
	case "email":
		return ctx.Email
	case "teamId", "team_id":
		if len(ctx.TeamIDs) > 0 {
			return ctx.TeamIDs[0]
		}
		return ""
	case "teamIds", "team_ids":
		return strings.Join(ctx.TeamIDs, ",")
	case "label":
		return strings.Join(ctx.Labels, ",")
	default:
		// Check custom attributes
		if strings.HasPrefix(attr, "custom.") {
			key := strings.TrimPrefix(attr, "custom.")
			if ctx.Attributes != nil {
				return ctx.Attributes[key]
			}
		}
		// Direct attribute lookup
		if ctx.Attributes != nil {
			if v, ok := ctx.Attributes[attr]; ok {
				return v
			}
		}
		return ""
	}
}

func parseValueList(v interface{}) []string {
	switch val := v.(type) {
	case string:
		return strings.Split(val, ",")
	case []interface{}:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	default:
		return []string{fmt.Sprintf("%v", v)}
	}
}

// ── Percentage Rollout ───────────────────────────────────────────────────────

// inRollout uses consistent hashing to determine if a user is in a rollout.
// Same user + flag always gets the same result (sticky bucketing).
func (s *Service) inRollout(flagKey string, ctx *EvalContext, pct int) bool {
	if pct >= 100 {
		return true
	}
	if pct <= 0 {
		return false
	}
	identifier := safeUserID(ctx)
	if identifier == "" {
		identifier = "anonymous"
	}

	// Hash flag key + user ID for consistent bucketing
	h := sha256.Sum256([]byte(flagKey + ":" + identifier))
	bucket := binary.BigEndian.Uint32(h[:4]) % 100
	return int(bucket) < pct
}

func safeUserID(ctx *EvalContext) string {
	if ctx != nil {
		return ctx.UserID
	}
	return ""
}

// ── Analytics ────────────────────────────────────────────────────────────────

func (s *Service) recordEvaluation(ctx context.Context, flagID, projectID, userID string, value interface{}, ruleID string) {
	valueJSON, _ := json.Marshal(value)
	var ruleVal interface{} = nil
	if ruleID != "" {
		ruleVal = ruleID
	}
	s.db.ExecContext(ctx,
		`INSERT INTO flag_evaluations (id, flag_id, project_id, user_id, value, rule_id, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uid.New("unique()"), flagID, projectID, userID, valueJSON, ruleVal, time.Now().UTC())
}

// GetFlagStats returns evaluation stats for a flag.
func (s *Service) GetFlagStats(ctx context.Context, flagID string) (map[string]interface{}, error) {
	var total int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM flag_evaluations WHERE flag_id=$1`, flagID).Scan(&total)

	var uniqueUsers int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM flag_evaluations WHERE flag_id=$1 AND user_id != ''`, flagID).Scan(&uniqueUsers)

	// Value distribution
	rows, err := s.db.QueryContext(ctx,
		`SELECT value, COUNT(*) as cnt FROM flag_evaluations WHERE flag_id=$1 GROUP BY value ORDER BY cnt DESC LIMIT 10`, flagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	distribution := []map[string]interface{}{}
	for rows.Next() {
		var valueJSON []byte
		var cnt int
		rows.Scan(&valueJSON, &cnt)
		var val interface{}
		json.Unmarshal(valueJSON, &val)
		distribution = append(distribution, map[string]interface{}{"value": val, "count": cnt})
	}

	return map[string]interface{}{
		"totalEvaluations": total,
		"uniqueUsers":      uniqueUsers,
		"distribution":     distribution,
	}, nil
}

// Ensure sql import is used
var _ = sql.ErrNoRows
