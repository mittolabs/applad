// Package audit provides a tamper-evident, append-only audit log for all API activity.
package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Log is a single audit log entry.
type Log struct {
	ID           string                 `json:"$id"`
	ProjectID    string                 `json:"projectId"`
	UserID       string                 `json:"userId,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resourceType"`
	ResourceID   string                 `json:"resourceId,omitempty"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	StatusCode   int                    `json:"statusCode"`
	IPAddress    string                 `json:"ipAddress,omitempty"`
	UserAgent    string                 `json:"userAgent,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"$createdAt"`
}

// Service handles audit log persistence and queries.
type Service struct {
	db *db.DB
}

// NewService creates a new audit Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Record persists an audit log entry. Non-blocking: errors are silently dropped.
func (s *Service) Record(ctx context.Context, entry Log) {
	if entry.ProjectID == "" {
		return
	}
	entry.ID = uid.New("")
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	var metaJSON []byte
	if entry.Metadata != nil {
		metaJSON, _ = json.Marshal(entry.Metadata)
	}
	s.db.ExecContext(ctx, //nolint:errcheck
		`INSERT INTO audit_logs
		 (id, project_id, user_id, action, resource_type, resource_id, method, path, status_code, ip_address, user_agent, metadata, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		entry.ID, entry.ProjectID, nullStr(entry.UserID), entry.Action,
		entry.ResourceType, nullStr(entry.ResourceID), entry.Method, entry.Path,
		entry.StatusCode, nullStr(entry.IPAddress), nullStr(entry.UserAgent),
		nullBytes(metaJSON), entry.CreatedAt,
	)
}

// Get fetches a single audit log entry by ID.
func (s *Service) Get(ctx context.Context, logID, projectID string) (*Log, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, COALESCE(user_id,''), action, resource_type, COALESCE(resource_id,''),
		        method, path, status_code, COALESCE(ip_address,''), COALESCE(user_agent,''), metadata, created_at
		 FROM audit_logs WHERE id = ? AND project_id = ?`, logID, projectID)
	l := &Log{}
	var metaRaw []byte
	if err := row.Scan(&l.ID, &l.ProjectID, &l.UserID, &l.Action, &l.ResourceType,
		&l.ResourceID, &l.Method, &l.Path, &l.StatusCode, &l.IPAddress, &l.UserAgent,
		&metaRaw, &l.CreatedAt); err != nil {
		return nil, err
	}
	if len(metaRaw) > 0 {
		json.Unmarshal(metaRaw, &l.Metadata) //nolint:errcheck
	}
	return l, nil
}

// List returns audit logs for a project with optional filters.
func (s *Service) List(ctx context.Context, projectID, action, resourceType, userID, method string, limit, offset int) ([]*Log, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	args := []interface{}{projectID}
	where := "WHERE project_id = ?"
	if action != "" {
		where += " AND action = ?"
		args = append(args, action)
	}
	if resourceType != "" {
		where += " AND resource_type = ?"
		args = append(args, resourceType)
	}
	if userID != "" {
		where += " AND user_id = ?"
		args = append(args, userID)
	}
	if method != "" {
		where += " AND method = ?"
		args = append(args, method)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_logs "+where, countArgs...).Scan(&total) //nolint:errcheck

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, COALESCE(user_id,''), action, resource_type, COALESCE(resource_id,''), method, path, status_code, COALESCE(ip_address,''), COALESCE(user_agent,''), metadata, created_at "+
			"FROM audit_logs "+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*Log
	for rows.Next() {
		l := &Log{}
		var metaRaw []byte
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.UserID, &l.Action, &l.ResourceType,
			&l.ResourceID, &l.Method, &l.Path, &l.StatusCode, &l.IPAddress, &l.UserAgent,
			&metaRaw, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		if len(metaRaw) > 0 {
			json.Unmarshal(metaRaw, &l.Metadata) //nolint:errcheck
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
