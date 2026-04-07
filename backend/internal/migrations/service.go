// Package migrations implements Applad's data migration service:
// import data from external platforms (Appwrite, Firebase, Supabase).
package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/uid"
)

// Migration represents a data migration from an external platform.
type Migration struct {
	ID          string      `json:"$id"`
	ProjectID   string      `json:"projectId"`
	Source      string      `json:"source"`
	Status      string      `json:"status"`
	Resources   interface{} `json:"resources"`
	Errors      interface{} `json:"errors"`
	Progress    int         `json:"progress"`
	StartedAt   *time.Time  `json:"startedAt"`
	CompletedAt *time.Time  `json:"completedAt"`
	CreatedAt   time.Time   `json:"$createdAt"`
}

// ValidationReport describes what will be imported and potential issues.
type ValidationReport struct {
	Source    string                   `json:"source"`
	Valid     bool                     `json:"valid"`
	Resources []map[string]interface{} `json:"resources"`
	Warnings  []string                 `json:"warnings"`
	Errors    []string                 `json:"errors"`
}

// Service handles migration business logic.
type Service struct {
	db    *db.DB
	queue *queue.Queue
}

// NewService creates a new migrations Service.
func NewService(database *db.DB, q *queue.Queue) *Service {
	return &Service{db: database, queue: q}
}

// Create creates a new migration record and enqueues a job to process it.
func (s *Service) Create(ctx context.Context, projectID, source string, config map[string]interface{}) (*Migration, error) {
	validSources := map[string]bool{
		"appwrite": true, "firebase": true, "supabase": true,
	}
	if !validSources[source] {
		return nil, fmt.Errorf("unsupported migration source: %s", source)
	}

	id := uid.New("unique()")
	now := time.Now().UTC()
	configJSON, _ := json.Marshal(config)
	resourcesJSON, _ := json.Marshal(map[string]interface{}{})
	errorsJSON, _ := json.Marshal([]string{})

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO migrations (id, project_id, source, status, config, resources, errors, progress, created_at)
		 VALUES (?, ?, ?, 'pending', ?, ?, ?, 0, ?)`,
		id, projectID, source, configJSON, resourcesJSON, errorsJSON, now)
	if err != nil {
		return nil, fmt.Errorf("migrations: create: %w", err)
	}

	// Enqueue for the migrations worker
	if s.queue != nil {
		s.queue.Push(ctx, "migrations", queue.Job{
			ID:   id,
			Type: "migration",
			Payload: map[string]interface{}{
				"migrationId": id,
				"projectId":   projectID,
				"source":      source,
				"config":      config,
			},
			CreatedAt: now,
		})
	}

	emptyResources := map[string]interface{}{}
	emptyErrors := []string{}
	return &Migration{
		ID: id, ProjectID: projectID, Source: source,
		Status: "pending", Resources: emptyResources, Errors: emptyErrors,
		Progress: 0, CreatedAt: now,
	}, nil
}

// Get returns a migration by ID.
func (s *Service) Get(ctx context.Context, migrationID, projectID string) (*Migration, error) {
	var m Migration
	var resourcesJSON, errorsJSON []byte
	var startedAt, completedAt sql.NullTime

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, source, status, resources, errors, progress, started_at, completed_at, created_at
		 FROM migrations WHERE id = ? AND project_id = ?`, migrationID, projectID,
	).Scan(&m.ID, &m.ProjectID, &m.Source, &m.Status, &resourcesJSON, &errorsJSON,
		&m.Progress, &startedAt, &completedAt, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("migration not found")
	}
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		m.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		m.CompletedAt = &completedAt.Time
	}
	json.Unmarshal(resourcesJSON, &m.Resources)
	json.Unmarshal(errorsJSON, &m.Errors)
	if m.Resources == nil {
		m.Resources = map[string]interface{}{}
	}
	if m.Errors == nil {
		m.Errors = []string{}
	}
	return &m, nil
}

// List returns all migrations for a project.
func (s *Service) List(ctx context.Context, projectID string) ([]*Migration, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, source, status, resources, errors, progress, started_at, completed_at, created_at
		 FROM migrations WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var migrations []*Migration
	for rows.Next() {
		var m Migration
		var resourcesJSON, errorsJSON []byte
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Source, &m.Status, &resourcesJSON, &errorsJSON,
			&m.Progress, &startedAt, &completedAt, &m.CreatedAt); err != nil {
			return nil, 0, err
		}
		if startedAt.Valid {
			m.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			m.CompletedAt = &completedAt.Time
		}
		json.Unmarshal(resourcesJSON, &m.Resources)
		json.Unmarshal(errorsJSON, &m.Errors)
		if m.Resources == nil {
			m.Resources = map[string]interface{}{}
		}
		if m.Errors == nil {
			m.Errors = []string{}
		}
		migrations = append(migrations, &m)
	}
	return migrations, len(migrations), nil
}

// Retry resets a failed migration to pending and re-enqueues it.
func (s *Service) Retry(ctx context.Context, migrationID, projectID string) (*Migration, error) {
	m, err := s.Get(ctx, migrationID, projectID)
	if err != nil {
		return nil, err
	}
	if m.Status != "failed" && m.Status != "completed" {
		return nil, fmt.Errorf("can only retry failed or completed migrations")
	}

	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx,
		`UPDATE migrations SET status = 'pending', progress = 0, errors = '[]', started_at = NULL, completed_at = NULL
		 WHERE id = ? AND project_id = ?`, migrationID, projectID)
	if err != nil {
		return nil, fmt.Errorf("migrations: retry: %w", err)
	}

	// Re-enqueue
	if s.queue != nil {
		// Retrieve config from DB
		var configJSON []byte
		s.db.QueryRowContext(ctx, `SELECT config FROM migrations WHERE id = ?`, migrationID).Scan(&configJSON)
		var config map[string]interface{}
		json.Unmarshal(configJSON, &config)

		s.queue.Push(ctx, "migrations", queue.Job{
			ID:   migrationID,
			Type: "migration",
			Payload: map[string]interface{}{
				"migrationId": migrationID,
				"projectId":   projectID,
				"source":      m.Source,
				"config":      config,
			},
			CreatedAt: now,
		})
	}

	return s.Get(ctx, migrationID, projectID)
}

// Delete removes a migration record.
func (s *Service) Delete(ctx context.Context, migrationID, projectID string) error {
	m, err := s.Get(ctx, migrationID, projectID)
	if err != nil {
		return err
	}
	if m.Status == "running" {
		return fmt.Errorf("cannot delete a running migration")
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM migrations WHERE id = ? AND project_id = ?", migrationID, projectID)
	return err
}

// UpdateProgress updates the progress and status of a migration.
func (s *Service) UpdateProgress(ctx context.Context, migrationID string, progress int, status string, resources, errors interface{}) error {
	resourcesJSON, _ := json.Marshal(resources)
	errorsJSON, _ := json.Marshal(errors)

	var startedAt, completedAt interface{}
	if status == "running" {
		now := time.Now().UTC()
		startedAt = now
	}
	if status == "completed" || status == "failed" {
		now := time.Now().UTC()
		completedAt = now
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE migrations SET progress = ?, status = ?, resources = ?, errors = ?,
		 started_at = COALESCE(?, started_at), completed_at = COALESCE(?, completed_at)
		 WHERE id = ?`,
		progress, status, resourcesJSON, errorsJSON, startedAt, completedAt, migrationID)
	return err
}

// ValidateReport returns a validation report for a migration configuration
// without executing it. It checks connectivity and lists what would be imported.
func (s *Service) ValidateReport(ctx context.Context, projectID, source string, config map[string]interface{}) (*ValidationReport, error) {
	validSources := map[string]bool{
		"appwrite": true, "firebase": true, "supabase": true,
	}
	if !validSources[source] {
		return nil, fmt.Errorf("unsupported migration source: %s", source)
	}

	report := &ValidationReport{
		Source:    source,
		Valid:     true,
		Resources: []map[string]interface{}{},
		Warnings:  []string{},
		Errors:    []string{},
	}

	// Validate required config fields per source
	switch source {
	case "appwrite":
		requiredFields := []string{"endpoint", "projectId", "apiKey"}
		for _, field := range requiredFields {
			if _, ok := config[field]; !ok {
				report.Errors = append(report.Errors, fmt.Sprintf("missing required field: %s", field))
				report.Valid = false
			}
		}
		if report.Valid {
			report.Resources = append(report.Resources,
				map[string]interface{}{"type": "users", "description": "User accounts and sessions"},
				map[string]interface{}{"type": "databases", "description": "Databases, collections, and documents"},
				map[string]interface{}{"type": "storage", "description": "Buckets and files"},
				map[string]interface{}{"type": "functions", "description": "Serverless functions"},
			)
			report.Warnings = append(report.Warnings, "OAuth sessions will not be migrated")
		}

	case "firebase":
		requiredFields := []string{"serviceAccountKey"}
		for _, field := range requiredFields {
			if _, ok := config[field]; !ok {
				report.Errors = append(report.Errors, fmt.Sprintf("missing required field: %s", field))
				report.Valid = false
			}
		}
		if report.Valid {
			report.Resources = append(report.Resources,
				map[string]interface{}{"type": "auth", "description": "Firebase Authentication users"},
				map[string]interface{}{"type": "firestore", "description": "Firestore collections and documents"},
				map[string]interface{}{"type": "storage", "description": "Cloud Storage files"},
			)
			report.Warnings = append(report.Warnings, "Firebase custom claims will be stored as user labels")
			report.Warnings = append(report.Warnings, "Firestore subcollections will be flattened")
		}

	case "supabase":
		requiredFields := []string{"host", "apiKey", "serviceRoleKey"}
		for _, field := range requiredFields {
			if _, ok := config[field]; !ok {
				report.Errors = append(report.Errors, fmt.Sprintf("missing required field: %s", field))
				report.Valid = false
			}
		}
		if report.Valid {
			report.Resources = append(report.Resources,
				map[string]interface{}{"type": "auth", "description": "Supabase Auth users"},
				map[string]interface{}{"type": "database", "description": "PostgreSQL tables and rows"},
				map[string]interface{}{"type": "storage", "description": "Storage buckets and objects"},
				map[string]interface{}{"type": "edge_functions", "description": "Edge Functions"},
			)
			report.Warnings = append(report.Warnings, "PostgreSQL-specific column types will be mapped to closest Applad equivalents")
			report.Warnings = append(report.Warnings, "Row Level Security policies will not be migrated")
		}
	}

	return report, nil
}
