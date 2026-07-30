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

func (s *Service) Create(ctx context.Context, projectID, teamID, name, creatorUserID string, roles []string) (*model.Team, error) {
	id := uid.New(teamID)
	now := time.Now().UTC()
	prefsJSON, _ := json.Marshal(map[string]interface{}{})
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO teams (id, project_id, name, prefs, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		id, projectID, name, prefsJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("teams: create: %w", err)
	}
	// Whoever creates a team is its first, already-joined owner. Without this a
	// creator holds no membership, so team-scoped RLS (read("team:<id>")) would
	// shut them out of the very team they just made. Server-side calls with no
	// user (an API key) create an unowned team, as before.
	if creatorUserID != "" {
		mid := uid.New("unique()")
		// The creator is always an owner, but any caller-supplied roles ride along
		// too, so create(..., roles) is not inert: the stored set is the union of
		// {"owner"} and the provided list, with "owner" guaranteed present.
		ownerRoleList := []string{"owner"}
		for _, r := range roles {
			if r != "" && r != "owner" {
				ownerRoleList = append(ownerRoleList, r)
			}
		}
		ownerRoles, _ := json.Marshal(ownerRoleList)
		if _, err := s.db.ExecContext(ctx,
			"INSERT INTO memberships (id, team_id, user_id, roles, invited, joined, created_at) VALUES ($1, $2, $3, $4, TRUE, TRUE, $5)",
			mid, id, creatorUserID, ownerRoles, now); err != nil {
			return nil, fmt.Errorf("teams: enrol creator: %w", err)
		}
	}
	return &model.Team{ID: id, Name: name, Prefs: map[string]interface{}{}, CreatedAt: now, UpdatedAt: now}, nil
}

// AcceptMembership turns an invite into a joined membership. The secret handed
// out at invite time is the credential; the joining user's identity is taken
// from their authenticated session, never from the request body, so nobody can
// join a team as someone else. It binds user_id, marks the row joined, and
// clears the one-time secret.
func (s *Service) AcceptMembership(ctx context.Context, teamID, membershipID, userID, secret string) (*model.Membership, error) {
	if userID == "" || secret == "" {
		return nil, fmt.Errorf("teams: accept requires an authenticated user and the invite secret")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE memberships SET user_id = $1, joined = TRUE, invited = TRUE, secret = NULL
		  WHERE id = $2 AND team_id = $3 AND secret = $4 AND joined = FALSE`,
		userID, membershipID, teamID, secret)
	if err != nil {
		return nil, fmt.Errorf("teams: accept membership: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("teams: invite is invalid, already used, or expired")
	}
	ms, _, err := s.ListMemberships(ctx, teamID, "")
	if err == nil {
		for _, m := range ms {
			if m.ID == membershipID {
				return m, nil
			}
		}
	}
	return &model.Membership{ID: membershipID, TeamID: teamID, UserID: userID, Joined: true, Invited: true}, nil
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
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memberships WHERE team_id = $1 AND joined = TRUE", teamID).Scan(&t.Total) //nolint:errcheck
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

// UpdatePrefs replaces a team's preferences with the supplied object and
// returns the refreshed team. Prefs are stored whole (not merged), matching the
// user-preferences shape in internal/auth. A nil map is written as an empty
// object so the column never holds SQL NULL.
func (s *Service) UpdatePrefs(ctx context.Context, teamID, projectID string, prefs map[string]interface{}) (*model.Team, error) {
	if prefs == nil {
		prefs = map[string]interface{}{}
	}
	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return nil, fmt.Errorf("teams: marshal prefs: %w", err)
	}
	res, err := s.db.ExecContext(ctx,
		"UPDATE teams SET prefs = $1, updated_at = $2 WHERE id = $3 AND project_id = $4",
		prefsJSON, time.Now().UTC(), teamID, projectID)
	if err != nil {
		return nil, fmt.Errorf("teams: update prefs: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("team not found")
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
		"INSERT INTO memberships (id, team_id, invited_email, roles, invited, joined, secret, created_at) VALUES ($1, $2, $3, $4, TRUE, FALSE, $5, $6)",
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
		Secret:    secret,
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

// GetMembership returns a single membership by id, scoped to its team and to the
// caller's project. It joins the team in both for the team name and to bind the
// row to project_id, so a caller (including a server API key) cannot read a
// membership belonging to another project by guessing its id. A row that does
// not exist (or belongs to another team or project) yields "membership not
// found", which the handler maps to a 404.
func (s *Service) GetMembership(ctx context.Context, teamID, membershipID, projectID string) (*model.Membership, error) {
	var m model.Membership
	var userID sql.NullString
	var invitedEmail sql.NullString
	var rolesJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT m.id, m.team_id, m.user_id, m.invited_email, m.roles, m.invited, m.joined, m.created_at, t.name FROM memberships m JOIN teams t ON t.id = m.team_id WHERE m.id = $1 AND m.team_id = $2 AND t.project_id = $3",
		membershipID, teamID, projectID).Scan(&m.ID, &m.TeamID, &userID, &invitedEmail, &rolesJSON, &m.Invited, &m.Joined, &m.CreatedAt, &m.TeamName)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("membership not found")
	}
	if err != nil {
		return nil, err
	}
	m.UserID = userID.String
	m.UserEmail = invitedEmail.String
	json.Unmarshal(rolesJSON, &m.Roles) //nolint:errcheck
	if m.Roles == nil {
		m.Roles = []string{}
	}
	return &m, nil
}

// UpdateMembershipRoles replaces a membership's roles with the supplied list and
// returns the refreshed record. The roles column is written whole (not merged),
// matching how CreateMembership stores it. The handler gates this owner-only, so
// a non-owner cannot use it to grant themselves "owner"; the service records
// exactly what it is given. A missing row yields "membership not found".
func (s *Service) UpdateMembershipRoles(ctx context.Context, teamID, membershipID, projectID string, roles []string) (*model.Membership, error) {
	if roles == nil {
		roles = []string{}
	}
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return nil, fmt.Errorf("teams: marshal roles: %w", err)
	}
	// The EXISTS clause binds the target team to the caller's project, so a key or
	// session scoped to project B cannot rewrite the roles of a membership in
	// project A even if it learns the team and membership ids. A cross-project id
	// updates nothing and surfaces as "membership not found".
	res, err := s.db.ExecContext(ctx,
		"UPDATE memberships SET roles = $1 WHERE id = $2 AND team_id = $3 AND EXISTS (SELECT 1 FROM teams WHERE teams.id = $3 AND teams.project_id = $4)",
		rolesJSON, membershipID, teamID, projectID)
	if err != nil {
		return nil, fmt.Errorf("teams: update membership roles: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("membership not found")
	}
	return s.GetMembership(ctx, teamID, membershipID, projectID)
}

// RolesForUser returns the RLS role tokens a user holds through team
// membership, scoped to one project. For each team the user has actually joined
// it yields "team:<id>" and, for each membership role, "team:<id>/<role>" (e.g.
// "team:abc/owner"). This is what lets a row permission like read("team:abc")
// admit exactly that team's members. It reads only joined memberships, so an
// unaccepted invite grants nothing, and it is called server-side with the
// authenticated user id — never a value the client supplied.
// ListForUser lists only the teams a user has joined. A plain signed-in user
// must not see every team in the project (that would leak the name of every
// other user's channel/workspace), so their listing is scoped to membership.
// Admins and API keys use List for the unscoped view.
func (s *Service) ListForUser(ctx context.Context, projectID, userID string, limit, offset int, search string) ([]*model.Team, int, error) {
	if limit <= 0 {
		limit = 25
	}
	args := []interface{}{projectID, userID}
	query := `SELECT t.id, t.name, t.prefs, t.created_at, t.updated_at
		        FROM teams t
		        JOIN memberships m ON m.team_id = t.id
		       WHERE t.project_id = $1 AND m.user_id = $2 AND m.joined = TRUE`
	n := 2
	if search != "" {
		n++
		query += fmt.Sprintf(" AND t.name LIKE $%d", n)
		args = append(args, "%"+search+"%")
	}
	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY t.created_at DESC LIMIT $%d OFFSET $%d", n+1, n+2)
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
	return teams, len(teams), rows.Err()
}

func (s *Service) RolesForUser(ctx context.Context, projectID, userID string) ([]string, error) {
	if userID == "" {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.team_id, m.roles FROM memberships m
		   JOIN teams t ON t.id = m.team_id
		  WHERE m.user_id = $1 AND m.joined = TRUE AND t.project_id = $2`,
		userID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var teamID string
		var rolesJSON []byte
		if err := rows.Scan(&teamID, &rolesJSON); err != nil {
			return nil, err
		}
		out = append(out, "team:"+teamID)
		var roles []string
		_ = json.Unmarshal(rolesJSON, &roles)
		for _, r := range roles {
			if r != "" {
				out = append(out, "team:"+teamID+"/"+r)
			}
		}
	}
	return out, rows.Err()
}

// MembershipOf reports whether userID is a joined member of teamID and, if so,
// whether they hold the "owner" role. It reads only joined memberships, so an
// unaccepted invite grants nothing. This is what the handler uses to authorize
// team reads and mutations for an end-user session: without it any project user
// could rename or delete another user's team, dump its roster, or self-add to a
// privileged team (whose roles feed database RLS).
func (s *Service) MembershipOf(ctx context.Context, teamID, userID string) (member bool, owner bool, err error) {
	if teamID == "" || userID == "" {
		return false, false, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT roles FROM memberships WHERE team_id = $1 AND user_id = $2 AND joined = TRUE`,
		teamID, userID)
	if err != nil {
		return false, false, err
	}
	defer rows.Close()
	for rows.Next() {
		member = true
		var rolesJSON []byte
		if err := rows.Scan(&rolesJSON); err != nil {
			return false, false, err
		}
		var roles []string
		_ = json.Unmarshal(rolesJSON, &roles)
		for _, role := range roles {
			if role == "owner" {
				owner = true
			}
		}
	}
	return member, owner, rows.Err()
}

// DeleteMembership removes a membership, scoped to its team and the caller's
// project. The EXISTS clause makes the projectID argument load-bearing (it was
// previously accepted and ignored): a caller scoped to another project cannot
// delete a membership in this team by guessing ids, because the team must belong
// to the caller's project for any row to be removed.
func (s *Service) DeleteMembership(ctx context.Context, membershipID, teamID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM memberships WHERE id = $1 AND team_id = $2 AND EXISTS (SELECT 1 FROM teams WHERE teams.id = $2 AND teams.project_id = $3)",
		membershipID, teamID, projectID)
	return err
}
