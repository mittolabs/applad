// Package regions provides multi-region deployment support and data residency
// configuration (GDPR / HIPAA) per project.
package regions

import (
	"context"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Region is a geographic deployment region.
type Region struct {
	ID        string    `json:"$id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Location  string    `json:"location"`
	Endpoint  string    `json:"endpoint"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"$createdAt"`
}

// ProjectRegion links a project to a region with compliance flags.
type ProjectRegion struct {
	ID            string    `json:"$id"`
	ProjectID     string    `json:"projectId"`
	RegionID      string    `json:"regionId"`
	Region        *Region   `json:"region,omitempty"`
	PrimaryRegion bool      `json:"primaryRegion"`
	GDPR          bool      `json:"gdpr"`
	HIPAA         bool      `json:"hipaa"`
	CreatedAt     time.Time `json:"$createdAt"`
}

// Service manages regions and project region assignments.
type Service struct {
	db *db.DB
}

// NewService creates a new regions Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// ListRegions returns all available regions.
func (s *Service) ListRegions(ctx context.Context) ([]*Region, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, code, location, endpoint,
		        COALESCE(latitude,0), COALESCE(longitude,0), status, created_at
		 FROM regions WHERE status = 'active' ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Region
	for rows.Next() {
		r, err := scanRegion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// GetRegion fetches a region by ID or code.
func (s *Service) GetRegion(ctx context.Context, idOrCode string) (*Region, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, code, location, endpoint,
		        COALESCE(latitude,0), COALESCE(longitude,0), status, created_at
		 FROM regions WHERE id = $1 OR code = $2`, idOrCode, idOrCode)
	return scanRegion(row)
}

// ── Project region assignments ────────────────────────────────────────────────

// AssignRegion adds a region to a project.
func (s *Service) AssignRegion(ctx context.Context, projectID, regionID string, primary, gdpr, hipaa bool) (*ProjectRegion, error) {
	// If primary, clear existing primary
	if primary {
		s.db.ExecContext(ctx, //nolint:errcheck
			"UPDATE project_regions SET primary_region=FALSE WHERE project_id=$1", projectID)
	}
	pr := &ProjectRegion{
		ID: uid.New(""), ProjectID: projectID, RegionID: regionID,
		PrimaryRegion: primary, GDPR: gdpr, HIPAA: hipaa,
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO project_regions (id, project_id, region_id, primary_region, gdpr, hipaa, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (project_id, region_id) DO UPDATE SET primary_region=EXCLUDED.primary_region, gdpr=EXCLUDED.gdpr, hipaa=EXCLUDED.hipaa`,
		pr.ID, pr.ProjectID, pr.RegionID,
		pr.PrimaryRegion, pr.GDPR, pr.HIPAA,
		pr.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("regions: assign: %w", err)
	}
	pr.Region, _ = s.GetRegion(ctx, regionID)
	return pr, nil
}

// ListProjectRegions returns all regions assigned to a project.
func (s *Service) ListProjectRegions(ctx context.Context, projectID string) ([]*ProjectRegion, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT pr.id, pr.project_id, pr.region_id, pr.primary_region, pr.gdpr, pr.hipaa, pr.created_at,
		        r.id, r.name, r.code, r.location, r.endpoint,
		        COALESCE(r.latitude,0), COALESCE(r.longitude,0), r.status, r.created_at
		 FROM project_regions pr
		 JOIN regions r ON r.id = pr.region_id
		 WHERE pr.project_id = $1 ORDER BY pr.primary_region DESC, pr.created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ProjectRegion
	for rows.Next() {
		pr := &ProjectRegion{Region: &Region{}}
		if err := rows.Scan(&pr.ID, &pr.ProjectID, &pr.RegionID,
			&pr.PrimaryRegion, &pr.GDPR, &pr.HIPAA, &pr.CreatedAt,
			&pr.Region.ID, &pr.Region.Name, &pr.Region.Code, &pr.Region.Location, &pr.Region.Endpoint,
			&pr.Region.Latitude, &pr.Region.Longitude, &pr.Region.Status, &pr.Region.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, pr)
	}
	return out, nil
}

// RemoveRegion removes a region assignment from a project.
func (s *Service) RemoveRegion(ctx context.Context, projectID, regionID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM project_regions WHERE project_id=$1 AND region_id=$2", projectID, regionID)
	return err
}

// GetPrimaryRegion returns the active primary region for a project.
func (s *Service) GetPrimaryRegion(ctx context.Context, projectID string) (*ProjectRegion, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT pr.id, pr.project_id, pr.region_id, pr.primary_region, pr.gdpr, pr.hipaa, pr.created_at,
		        r.id, r.name, r.code, r.location, r.endpoint,
		        COALESCE(r.latitude,0), COALESCE(r.longitude,0), r.status, r.created_at
		 FROM project_regions pr
		 JOIN regions r ON r.id = pr.region_id
		 WHERE pr.project_id = $1 AND pr.primary_region = TRUE
		 LIMIT 1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("regions: no primary region")
	}
	pr := &ProjectRegion{Region: &Region{}}
	if err := rows.Scan(&pr.ID, &pr.ProjectID, &pr.RegionID,
		&pr.PrimaryRegion, &pr.GDPR, &pr.HIPAA, &pr.CreatedAt,
		&pr.Region.ID, &pr.Region.Name, &pr.Region.Code, &pr.Region.Location, &pr.Region.Endpoint,
		&pr.Region.Latitude, &pr.Region.Longitude, &pr.Region.Status, &pr.Region.CreatedAt,
	); err != nil {
		return nil, err
	}
	return pr, nil
}

// RegionHealth returns a lightweight synthetic health report for a region.
func (s *Service) RegionHealth(ctx context.Context, idOrCode string) (map[string]interface{}, error) {
	region, err := s.GetRegion(ctx, idOrCode)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"regionId": region.ID,
		"code":     region.Code,
		"status":   region.Status,
		"latencyMs": 42,
		"healthy":  region.Status == "active",
	}, nil
}

// ── scanner ───────────────────────────────────────────────────────────────────

func scanRegion(row interface{ Scan(...interface{}) error }) (*Region, error) {
	r := &Region{}
	if err := row.Scan(&r.ID, &r.Name, &r.Code, &r.Location, &r.Endpoint,
		&r.Latitude, &r.Longitude, &r.Status, &r.CreatedAt); err != nil {
		return nil, err
	}
	return r, nil
}
