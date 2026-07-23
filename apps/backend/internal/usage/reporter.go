package usage

import (
	"context"

	"github.com/mittolabs/applad/internal/db"
)

// Reporter answers consumption questions against core's own schema. It exists
// so that anything resolving quotas reads through a stable interface instead of
// querying core's tables directly, which would make every schema change a
// silent break for code compiled from elsewhere.
type Reporter struct {
	db *db.DB
}

// NewReporter creates a Reporter.
func NewReporter(database *db.DB) *Reporter { return &Reporter{db: database} }

// CountProjects returns how many projects an organization owns.
func (r *Reporter) CountProjects(ctx context.Context, orgID string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM projects WHERE org_id = $1`, orgID).Scan(&n)
	return n, err
}

// CountMembers returns how many members an organization has, invited or active.
func (r *Reporter) CountMembers(ctx context.Context, orgID string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM organization_members WHERE org_id = $1`, orgID).Scan(&n)
	return n, err
}

// StorageBytes returns the bytes a project currently stores.
func (r *Reporter) StorageBytes(ctx context.Context, projectID string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(sum(size), 0) FROM files WHERE project_id = $1`, projectID).Scan(&n)
	return n, err
}
