// Package organizations implements multi-org support for the Applad console.
// Organizations group projects and members. Console users can belong to
// multiple orgs and switch between them.
package organizations

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Organization represents a console organization.
type Organization struct {
	ID          string    `json:"$id"`
	Name        string    `json:"name"`
	BillingPlan string    `json:"billingPlan"`
	CreatedAt   time.Time `json:"$createdAt"`
	UpdatedAt   time.Time `json:"$updatedAt"`
}

// Member represents an organization member.
type Member struct {
	ID        string    `json:"$id"`
	OrgID     string    `json:"orgId"`
	UserID    string    `json:"userId,omitempty"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`   // owner, admin, member
	Status    string    `json:"status"` // active, pending
	CreatedAt time.Time `json:"$createdAt"`
}

// Service handles organization business logic.
type Service struct {
	db *db.DB
}

// NewService creates a new organizations Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Create creates a new organization and sets the creator as owner.
func (s *Service) Create(ctx context.Context, name, creatorUserID, creatorEmail, creatorName string) (*Organization, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO organizations (id, name, created_at, updated_at) VALUES ($1, $2, $3, $4)",
		id, name, now, now)
	if err != nil {
		return nil, fmt.Errorf("organizations: create: %w", err)
	}

	// Add creator as owner
	memberID := uid.New("unique()")
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO organization_members (id, org_id, user_id, email, name, role, status, created_at) VALUES ($1, $2, $3, $4, $5, 'owner', 'active', $6)",
		memberID, id, creatorUserID, creatorEmail, creatorName, now)
	if err != nil {
		return nil, fmt.Errorf("organizations: add owner: %w", err)
	}

	// Set as user's default org
	s.db.ExecContext(ctx,
		"UPDATE console_users SET default_org_id = $1 WHERE id = $2", id, creatorUserID)

	return &Organization{ID: id, Name: name, BillingPlan: "free", CreatedAt: now, UpdatedAt: now}, nil
}

// Get returns an organization by ID.
func (s *Service) Get(ctx context.Context, id string) (*Organization, error) {
	var o Organization
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, billing_plan, created_at, updated_at FROM organizations WHERE id = $1", id,
	).Scan(&o.ID, &o.Name, &o.BillingPlan, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ListByUser returns all orgs a user belongs to.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]*Organization, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT o.id, o.name, o.billing_plan, o.created_at, o.updated_at
		 FROM organizations o
		 JOIN organization_members m ON m.org_id = o.id
		 WHERE m.user_id = $1 AND m.status = 'active'
		 ORDER BY o.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.BillingPlan, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, &o)
	}
	return orgs, nil
}

// Update updates an organization.
func (s *Service) Update(ctx context.Context, id, name string) (*Organization, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE organizations SET name = $1, updated_at = $2 WHERE id = $3",
		name, time.Now().UTC(), id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Delete removes an organization.
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM organizations WHERE id = $1", id)
	return err
}

// --- Members ---

// ListMembers returns all members of an organization.
func (s *Service) ListMembers(ctx context.Context, orgID string) ([]*Member, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, org_id, user_id, email, name, role, status, created_at FROM organization_members WHERE org_id = $1 ORDER BY created_at",
		orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*Member
	for rows.Next() {
		var m Member
		var userID, name sql.NullString
		if err := rows.Scan(&m.ID, &m.OrgID, &userID, &m.Email, &name, &m.Role, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.UserID = userID.String
		m.Name = name.String
		members = append(members, &m)
	}
	return members, nil
}

// InviteMember sends an invite to join the organization with a specific role.
func (s *Service) InviteMember(ctx context.Context, orgID, email, name, role string) (*Member, string, error) {
	id := uid.New("unique()")
	token := generateInviteToken()
	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO organization_members (id, org_id, email, name, role, status, invite_token, created_at) VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)",
		id, orgID, email, name, role, token, now)
	if err != nil {
		return nil, "", fmt.Errorf("organizations: invite: %w", err)
	}

	return &Member{ID: id, OrgID: orgID, Email: email, Name: name, Role: role, Status: "pending", CreatedAt: now}, token, nil
}

// AcceptInvite accepts a pending invite by token and links it to a console user.
func (s *Service) AcceptInvite(ctx context.Context, token, userID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE organization_members SET user_id = $1, status = 'active', invite_token = NULL WHERE invite_token = $2 AND status = 'pending'",
		userID, token)
	return err
}

// RemoveMember removes a member from an organization.
func (s *Service) RemoveMember(ctx context.Context, orgID, memberID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM organization_members WHERE id = $1 AND org_id = $2 AND role != 'owner'",
		memberID, orgID)
	return err
}

// UpdateMemberRole changes a member's role.
func (s *Service) UpdateMemberRole(ctx context.Context, orgID, memberID, role string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE organization_members SET role = $1 WHERE id = $2 AND org_id = $3",
		role, memberID, orgID)
	return err
}

// --- Project linking ---

// ListProjects returns all projects in an organization.
func (s *Service) ListProjects(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, description, created_at, updated_at FROM projects WHERE org_id = $1 ORDER BY created_at DESC", orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []map[string]interface{}
	for rows.Next() {
		var id, name string
		var desc sql.NullString
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &desc, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, map[string]interface{}{
			"$id": id, "name": name, "description": desc.String,
			"$createdAt": createdAt, "$updatedAt": updatedAt,
		})
	}
	return projects, nil
}

// CreateProject creates a project under an organization.
func (s *Service) CreateProject(ctx context.Context, orgID, name, description string) (map[string]interface{}, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO projects (id, org_id, name, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		id, orgID, name, description, now, now)
	if err != nil {
		return nil, fmt.Errorf("organizations: create project: %w", err)
	}
	return map[string]interface{}{
		"$id": id, "name": name, "description": description,
		"$createdAt": now, "$updatedAt": now,
	}, nil
}

// --- Org-level stats and activity ---

// OrgStats holds aggregate statistics across all projects in an organization.
type OrgStats struct {
	OrgID           string `json:"orgId"`
	TotalProjects   int    `json:"totalProjects"`
	TotalMembers    int    `json:"totalMembers"`
	TotalUsers      int64  `json:"totalUsers"`
	TotalStorage    int64  `json:"totalStorage"`
	TotalExecutions int64  `json:"totalExecutions"`
}

// GetOrgStats aggregates statistics across all projects in the organization.
func (s *Service) GetOrgStats(ctx context.Context, orgID string) (*OrgStats, error) {
	stats := &OrgStats{OrgID: orgID}
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects WHERE org_id = $1", orgID).Scan(&stats.TotalProjects)                                                            //nolint:errcheck
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM organization_members WHERE org_id = $1", orgID).Scan(&stats.TotalMembers)                                                 //nolint:errcheck
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE project_id IN (SELECT id FROM projects WHERE org_id = $1)", orgID).Scan(&stats.TotalUsers)                    //nolint:errcheck
	s.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(size),0) FROM files WHERE project_id IN (SELECT id FROM projects WHERE org_id = $1)", orgID).Scan(&stats.TotalStorage)     //nolint:errcheck
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM function_executions WHERE project_id IN (SELECT id FROM projects WHERE org_id = $1)", orgID).Scan(&stats.TotalExecutions) //nolint:errcheck
	return stats, nil
}

// ActivityEntry is a single activity log entry scoped to an organization.
type ActivityEntry struct {
	ID           string    `json:"$id"`
	ProjectID    string    `json:"projectId"`
	ProjectName  string    `json:"projectName"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resourceType"`
	ResourceID   string    `json:"resourceId,omitempty"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	StatusCode   int       `json:"statusCode"`
	IPAddress    string    `json:"ipAddress,omitempty"`
	CreatedAt    time.Time `json:"$createdAt"`
}

// ListActivity returns audit log entries across all projects in the organization.
func (s *Service) ListActivity(ctx context.Context, orgID string, limit, offset int) ([]*ActivityEntry, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var total int
	s.db.QueryRowContext(ctx, //nolint:errcheck
		"SELECT COUNT(*) FROM audit_logs WHERE project_id IN (SELECT id FROM projects WHERE org_id = $1)", orgID,
	).Scan(&total)

	rows, err := s.db.QueryContext(ctx,
		`SELECT al.id, al.project_id, COALESCE(p.name,''), al.action, al.resource_type,
		        COALESCE(al.resource_id,''), al.method, al.path, al.status_code,
		        COALESCE(al.ip_address,''), al.created_at
		 FROM audit_logs al
		 LEFT JOIN projects p ON p.id = al.project_id
		 WHERE al.project_id IN (SELECT id FROM projects WHERE org_id = $1)
		 ORDER BY al.created_at DESC LIMIT $2 OFFSET $3`, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("organizations: list activity: %w", err)
	}
	defer rows.Close()

	var entries []*ActivityEntry
	for rows.Next() {
		e := &ActivityEntry{}
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.ProjectName, &e.Action, &e.ResourceType,
			&e.ResourceID, &e.Method, &e.Path, &e.StatusCode, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []*ActivityEntry{}
	}
	return entries, total, nil
}

func generateInviteToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
