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
	Role      string    `json:"role"` // owner, admin, member
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
		"INSERT INTO organizations (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)",
		id, name, now, now)
	if err != nil {
		return nil, fmt.Errorf("organizations: create: %w", err)
	}

	// Add creator as owner
	memberID := uid.New("unique()")
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO organization_members (id, org_id, user_id, email, name, role, status, created_at) VALUES (?, ?, ?, ?, ?, 'owner', 'active', ?)",
		memberID, id, creatorUserID, creatorEmail, creatorName, now)
	if err != nil {
		return nil, fmt.Errorf("organizations: add owner: %w", err)
	}

	// Set as user's default org
	s.db.ExecContext(ctx,
		"UPDATE console_users SET default_org_id = ? WHERE id = ?", id, creatorUserID)

	return &Organization{ID: id, Name: name, BillingPlan: "free", CreatedAt: now, UpdatedAt: now}, nil
}

// Get returns an organization by ID.
func (s *Service) Get(ctx context.Context, id string) (*Organization, error) {
	var o Organization
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, billing_plan, created_at, updated_at FROM organizations WHERE id = ?", id,
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
		 WHERE m.user_id = ? AND m.status = 'active'
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
		"UPDATE organizations SET name = ?, updated_at = ? WHERE id = ?",
		name, time.Now().UTC(), id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Delete removes an organization.
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM organizations WHERE id = ?", id)
	return err
}

// --- Members ---

// ListMembers returns all members of an organization.
func (s *Service) ListMembers(ctx context.Context, orgID string) ([]*Member, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, org_id, user_id, email, name, role, status, created_at FROM organization_members WHERE org_id = ? ORDER BY created_at",
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

// InviteMember sends an invite to join the organization.
func (s *Service) InviteMember(ctx context.Context, orgID, email, name string) (*Member, string, error) {
	id := uid.New("unique()")
	token := generateInviteToken()
	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO organization_members (id, org_id, email, name, role, status, invite_token, created_at) VALUES (?, ?, ?, ?, 'member', 'pending', ?, ?)",
		id, orgID, email, name, token, now)
	if err != nil {
		return nil, "", fmt.Errorf("organizations: invite: %w", err)
	}

	return &Member{ID: id, OrgID: orgID, Email: email, Name: name, Role: "member", Status: "pending", CreatedAt: now}, token, nil
}

// AcceptInvite accepts a pending invite by token and links it to a console user.
func (s *Service) AcceptInvite(ctx context.Context, token, userID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE organization_members SET user_id = ?, status = 'active', invite_token = NULL WHERE invite_token = ? AND status = 'pending'",
		userID, token)
	return err
}

// RemoveMember removes a member from an organization.
func (s *Service) RemoveMember(ctx context.Context, orgID, memberID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM organization_members WHERE id = ? AND org_id = ? AND role != 'owner'",
		memberID, orgID)
	return err
}

// UpdateMemberRole changes a member's role.
func (s *Service) UpdateMemberRole(ctx context.Context, orgID, memberID, role string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE organization_members SET role = ? WHERE id = ? AND org_id = ?",
		role, memberID, orgID)
	return err
}

// --- Project linking ---

// ListProjects returns all projects in an organization.
func (s *Service) ListProjects(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, description, created_at, updated_at FROM projects WHERE org_id = ? ORDER BY created_at DESC", orgID)
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
		"INSERT INTO projects (id, org_id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, orgID, name, description, now, now)
	if err != nil {
		return nil, fmt.Errorf("organizations: create project: %w", err)
	}
	return map[string]interface{}{
		"$id": id, "name": name, "description": description,
		"$createdAt": now, "$updatedAt": now,
	}, nil
}

func generateInviteToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
