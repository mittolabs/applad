package teams

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/uid"
)

// Service handles teams business logic.
type Service struct {
	db *db.DB
}

// NewService creates a new teams Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

func (s *Service) Create(ctx context.Context, projectID, teamID, name string, roles []string) (*model.Team, error) {
	id := uid.New(teamID)
	now := time.Now().UTC()
	prefsJSON, _ := json.Marshal(map[string]interface{}{})
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO teams (id, project_id, name, prefs, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		id, projectID, name, prefsJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("teams: create: %w", err)
	}
	return &model.Team{ID: id, Name: name, Prefs: map[string]interface{}{}, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Service) Get(ctx context.Context, teamID, projectID string) (*model.Team, error) {
	var t model.Team
	var prefsJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, prefs, created_at, updated_at FROM teams WHERE id = $1 AND project_id = $2",
		teamID, projectID).Scan(&t.ID, &t.Name, &prefsJSON, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(prefsJSON, &t.Prefs) //nolint:errcheck
	if t.Prefs == nil {
		t.Prefs = map[string]interface{}{}
	}
	// count members
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memberships WHERE team_id = $1 AND joined = 1", teamID).Scan(&t.Total) //nolint:errcheck
	return &t, nil
}

func (s *Service) List(ctx context.Context, projectID string, limit, offset int, search string) ([]*model.Team, int, error) {
	if limit <= 0 {
		limit = 25
	}
	n := 1
	query := "SELECT id, name, prefs, created_at, updated_at FROM teams WHERE project_id = $1"
	args := []interface{}{projectID}
	if search != "" {
		n++
		query += fmt.Sprintf(" AND name LIKE $%d", n)
		args = append(args, "%"+search+"%")
	}
	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", n+1, n+2)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var teams []*model.Team
	for rows.Next() {
		var t model.Team
		var prefsJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &prefsJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(prefsJSON, &t.Prefs) //nolint:errcheck
		if t.Prefs == nil {
			t.Prefs = map[string]interface{}{}
		}
		teams = append(teams, &t)
	}
	var total int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM teams WHERE project_id = $1", projectID).Scan(&total) //nolint:errcheck
	return teams, total, nil
}

func (s *Service) Update(ctx context.Context, teamID, projectID, name string) (*model.Team, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE teams SET name = $1 WHERE id = $2 AND project_id = $3", name, teamID, projectID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, teamID, projectID)
}

func (s *Service) Delete(ctx context.Context, teamID, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM teams WHERE id = $1 AND project_id = $2", teamID, projectID)
	return err
}

func (s *Service) CreateMembership(ctx context.Context, teamID, projectID, email string, roles []string) (*model.Membership, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	secret := uid.RandomHex(32)
	rolesJSON, _ := json.Marshal(roles)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO memberships (id, team_id, invited_email, roles, invited, joined, secret, created_at) VALUES ($1, $2, $3, $4, 1, 0, $5, $6)",
		id, teamID, email, rolesJSON, secret, now)
	if err != nil {
		return nil, fmt.Errorf("teams: create membership: %w", err)
	}
	team, _ := s.Get(ctx, teamID, projectID)
	teamName := ""
	if team != nil {
		teamName = team.Name
	}
	if roles == nil {
		roles = []string{}
	}
	return &model.Membership{
		ID: id, TeamID: teamID, TeamName: teamName,
		UserEmail: email, Roles: roles,
		Invited: true, Joined: false, Confirm: false,
		CreatedAt: now,
	}, nil
}

func (s *Service) ListMemberships(ctx context.Context, teamID, projectID string) ([]*model.Membership, int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT m.id, m.team_id, m.user_id, m.invited_email, m.roles, m.invited, m.joined, m.created_at, t.name FROM memberships m JOIN teams t ON t.id = m.team_id WHERE m.team_id = $1 ORDER BY m.created_at DESC",
		teamID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var memberships []*model.Membership
	for rows.Next() {
		var m model.Membership
		var userID sql.NullString
		var invitedEmail sql.NullString
		var rolesJSON []byte
		if err := rows.Scan(&m.ID, &m.TeamID, &userID, &invitedEmail, &rolesJSON, &m.Invited, &m.Joined, &m.CreatedAt, &m.TeamName); err != nil {
			return nil, 0, err
		}
		m.UserID = userID.String
		m.UserEmail = invitedEmail.String
		json.Unmarshal(rolesJSON, &m.Roles) //nolint:errcheck
		if m.Roles == nil {
			m.Roles = []string{}
		}
		memberships = append(memberships, &m)
	}
	return memberships, len(memberships), nil
}

func (s *Service) DeleteMembership(ctx context.Context, membershipID, teamID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM memberships WHERE id = $1 AND team_id = $2", membershipID, teamID)
	return err
}
