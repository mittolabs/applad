package deploy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/mittolabs/applad/internal/githubapp"
	"github.com/mittolabs/applad/internal/uid"
)

/*
 * Connecting a project to GitHub through the Applad GitHub App.
 *
 * A connection is not a stored credential any more. It is a record that this
 * project is allowed to act through a particular installation — one account
 * that installed Applad, on the repositories they picked. The token to reach
 * those repositories is minted per use and never written down.
 *
 * The check that matters is here rather than at GitHub: the app can reach
 * every repository anyone has installed it on, so "can this project use this
 * installation" is Applad's question to answer, not GitHub's.
 */

// ErrInstallNotForProject is returned when a project asks to act through an
// installation it never connected.
var ErrInstallNotForProject = errors.New("deploy: repository belongs to a GitHub account this project has not connected")

// SetGitHubApp gives the service the configured app. Nil means this instance
// has none, and every path below reports that rather than failing obscurely.
func (s *Service) SetGitHubApp(a *githubapp.App) { s.github = a }

// GitHubApp returns the configured app, or nil.
func (s *Service) GitHubApp() *githubapp.App { return s.github }

// GitHubInstallURL is where somebody goes to connect an account.
//
// The state is a one-time value held in Redis against the project that started
// the flow. Passing the project id itself would let a link tricked into a
// victim's browser attach an installation to somebody else's project.
func (s *Service) GitHubInstallURL(ctx context.Context, projectID string) (string, error) {
	if s.github == nil {
		return "", githubapp.ErrNotConfigured
	}
	state := uid.RandomHex(16)
	if s.rdb != nil {
		if err := s.rdb.Set(ctx, "gh_install:"+state, projectID, 15*time.Minute).Err(); err != nil {
			return "", fmt.Errorf("deploy: github install state: %w", err)
		}
	}
	return s.github.InstallURL(state), nil
}

// CompleteGitHubInstall records an installation against the project that
// started the flow.
//
// Called when GitHub returns somebody to the console after they install or
// reconfigure the app. Reconfiguring sends the same installation again, so this
// updates in place rather than accumulating a row per visit.
func (s *Service) CompleteGitHubInstall(ctx context.Context, projectID, installationID, state string) (*GitConnection, error) {
	if s.github == nil {
		return nil, githubapp.ErrNotConfigured
	}
	if installationID == "" {
		return nil, fmt.Errorf("deploy: no installation id")
	}

	// The state proves this install was started by this project. Without
	// Redis (a self-hosted single process) there is nowhere to have kept it,
	// so the console's own auth is all there is.
	if s.rdb != nil && state != "" {
		owner, err := s.rdb.Get(ctx, "gh_install:"+state).Result()
		if err != nil || owner != projectID {
			return nil, fmt.Errorf("deploy: install state does not match this project")
		}
		s.rdb.Del(ctx, "gh_install:"+state) //nolint:errcheck
	}

	inst, err := s.github.GetInstallation(ctx, installationID)
	if err != nil {
		return nil, err
	}

	var existing string
	err = s.db.QueryRowContext(ctx,
		`SELECT id FROM git_connections WHERE project_id = $1 AND installation_id = $2`,
		projectID, installationID).Scan(&existing)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	now := time.Now().UTC()
	if existing != "" {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE git_connections SET account_name = $1, account_type = $2, updated_at = $3 WHERE id = $4`,
			inst.AccountName, inst.AccountType, now, existing); err != nil {
			return nil, err
		}
		return s.GetGitConnectionByID(ctx, existing)
	}

	id := uid.New("unique()")
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO git_connections (id, project_id, provider, installation_id, access_token,
		    refresh_token, account_name, account_type, created_at, updated_at)
		 VALUES ($1, $2, 'github', $3, '', '', $4, $5, $6, $6)`,
		id, projectID, installationID, inst.AccountName, inst.AccountType, now); err != nil {
		return nil, fmt.Errorf("deploy: record installation: %w", err)
	}

	return &GitConnection{
		ID: id, ProjectID: projectID, Provider: "github",
		InstallationID: installationID, AccountName: inst.AccountName,
		AccountType: inst.AccountType, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// CloneTokenForRepo returns a token that can clone repoURL on behalf of a
// project, or "" when the repository is public and needs none.
//
// Two things have to hold: the app must be able to reach the repository at
// all, and the installation that grants it must be one this project connected.
// The second is what stops a project naming somebody else's private repo and
// having Applad fetch it.
func (s *Service) CloneTokenForRepo(ctx context.Context, projectID, repoURL string) (string, error) {
	if s.github == nil {
		return "", nil
	}
	owner, repo, ok := githubapp.ParseRepoURL(repoURL)
	if !ok {
		return "", nil // Not GitHub — GitLab, Bitbucket, a bare URL.
	}

	inst, err := s.github.InstallationForRepo(ctx, owner, repo)
	if err != nil {
		// Applad is not installed there. A public repo still clones without a
		// token, so this is not fatal; a private one fails later with git's
		// own message, which names the repository.
		return "", nil
	}

	installationID := strconv.FormatInt(inst.ID, 10)
	var connected int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM git_connections WHERE project_id = $1 AND installation_id = $2`,
		projectID, installationID).Scan(&connected); err != nil {
		return "", err
	}
	if connected == 0 {
		return "", ErrInstallNotForProject
	}

	return s.github.InstallationToken(ctx, installationID)
}

// ConnectionByInstallation finds the connections an inbound webhook belongs to.
//
// The app has a single webhook URL for every installation, so a delivery
// arrives identified only by which installation it came from — and one
// installation may serve several projects.
func (s *Service) ConnectionsByInstallation(ctx context.Context, installationID string) ([]*GitConnection, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, provider, installation_id, account_name, account_type, created_at, updated_at
		   FROM git_connections WHERE installation_id = $1`, installationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*GitConnection
	for rows.Next() {
		var c GitConnection
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Provider, &c.InstallationID,
			&c.AccountName, &c.AccountType, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// RemoveInstallation forgets every connection to an installation.
//
// Sent when somebody uninstalls Applad from their account. Keeping the rows
// would leave projects listing a connection that can no longer reach anything.
func (s *Service) RemoveInstallation(ctx context.Context, installationID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM git_connections WHERE installation_id = $1`, installationID)
	return err
}
