// Package githubapp speaks to GitHub as the Applad GitHub App.
//
// A GitHub App is not a user with a token. It holds a private key, signs a
// short-lived JWT to prove it is itself, and exchanges that for an access
// token scoped to one *installation* — one account that has installed it, on
// the repositories they chose. Those tokens last an hour and are minted on
// demand, which is why nothing durable is stored: the private key is the only
// long-lived secret, and it never leaves this process.
//
// The alternative Applad shipped with — a personal access token pasted into a
// row — could read every repository the person could, forever, and broke when
// they left. An installation token can only reach what was granted to it.
package githubapp

import (
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mittolabs/applad/internal/config"
)

// ErrNotConfigured is returned when the instance has no GitHub App set up.
//
// Self-hosted installs are expected to hit this: the app belongs to Applad
// Cloud, and a self-hoster who wants git deploys registers their own. It is a
// condition to report, not a failure to log.
var ErrNotConfigured = errors.New("github app: not configured")

const apiBase = "https://api.github.com"

// App is the configured GitHub App.
type App struct {
	appID         string
	slug          string
	clientID      string
	clientSecret  string
	webhookSecret string
	key           *rsa.PrivateKey

	http *http.Client

	mu     sync.Mutex
	tokens map[string]cachedToken // installation id → token
}

type cachedToken struct {
	token     string
	expiresAt time.Time
}

// Config carries the app's credentials, as issued by GitHub when the app was
// created.
type Config struct {
	AppID         string
	Slug          string
	ClientID      string
	ClientSecret  string
	WebhookSecret string
	// PrivateKey is the PEM GitHub issued. Newlines may arrive escaped as
	// "\n", which is how a PEM survives a single-line env var.
	PrivateKey string
}

// New builds an App from its credentials. A blank app id or key means this
// instance has no GitHub App, which is not an error.
func New(cfg Config) (*App, error) {
	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.PrivateKey) == "" {
		return nil, ErrNotConfigured
	}

	pem := strings.ReplaceAll(cfg.PrivateKey, `\n`, "\n")
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(pem))
	if err != nil {
		return nil, fmt.Errorf("github app: private key: %w", err)
	}

	return &App{
		appID:         strings.TrimSpace(cfg.AppID),
		slug:          strings.TrimSpace(cfg.Slug),
		clientID:      cfg.ClientID,
		clientSecret:  cfg.ClientSecret,
		webhookSecret: cfg.WebhookSecret,
		key:           key,
		http:          &http.Client{Timeout: 20 * time.Second},
		tokens:        map[string]cachedToken{},
	}, nil
}

// Slug is the app's name in URLs — github.com/apps/<slug>.
func (a *App) Slug() string { return a.slug }

// InstallURL is where somebody goes to install the app on their account.
//
// The state ties the install that comes back to the project it was started
// from; GitHub returns it verbatim to the setup URL.
func (a *App) InstallURL(state string) string {
	return fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s",
		a.slug, url.QueryEscape(state))
}

// appJWT proves to GitHub that this process holds the app's private key.
//
// Good for ten minutes at most; GitHub rejects anything longer. Issued a
// minute in the past because GitHub compares against its own clock and a
// second of drift would otherwise reject it.
func (a *App) appJWT() (string, error) {
	now := time.Now()
	return jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Issuer:    a.appID,
		IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
	}).SignedString(a.key)
}

// InstallationToken returns an access token for one installation.
//
// Tokens are cached until shortly before they expire. Minting one per clone
// would be a round trip on every build and would burn through the rate limit
// on a busy project for no benefit.
func (a *App) InstallationToken(ctx context.Context, installationID string) (string, error) {
	a.mu.Lock()
	if t, ok := a.tokens[installationID]; ok && time.Until(t.expiresAt) > 5*time.Minute {
		a.mu.Unlock()
		return t.token, nil
	}
	a.mu.Unlock()

	var out struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := a.do(ctx, "POST",
		fmt.Sprintf("/app/installations/%s/access_tokens", url.PathEscape(installationID)),
		"", &out); err != nil {
		return "", err
	}

	a.mu.Lock()
	a.tokens[installationID] = cachedToken{token: out.Token, expiresAt: out.ExpiresAt}
	a.mu.Unlock()
	return out.Token, nil
}

// Installation describes one account that has installed the app.
type Installation struct {
	ID          int64  `json:"id"`
	AccountName string `json:"-"`
	AccountType string `json:"-"`
	Account     struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"account"`
	RepositorySelection string `json:"repository_selection"`
}

// GetInstallation reads one installation by id.
func (a *App) GetInstallation(ctx context.Context, installationID string) (*Installation, error) {
	var inst Installation
	if err := a.do(ctx, "GET",
		"/app/installations/"+url.PathEscape(installationID), "", &inst); err != nil {
		return nil, err
	}
	inst.AccountName, inst.AccountType = inst.Account.Login, inst.Account.Type
	return &inst, nil
}

// InstallationForRepo finds which installation covers a repository.
//
// This is what makes a clone possible without storing anything per repo: given
// a URL, ask GitHub whether the app can reach it and through whom. The caller
// still has to check that installation belongs to the project asking, or one
// project could deploy another's private code.
func (a *App) InstallationForRepo(ctx context.Context, owner, repo string) (*Installation, error) {
	var inst Installation
	if err := a.do(ctx, "GET",
		fmt.Sprintf("/repos/%s/%s/installation", url.PathEscape(owner), url.PathEscape(repo)),
		"", &inst); err != nil {
		return nil, err
	}
	inst.AccountName, inst.AccountType = inst.Account.Login, inst.Account.Type
	return &inst, nil
}

// Repository is a repository the app can reach through an installation.
type Repository struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	CloneURL      string `json:"clone_url"`
	HTMLURL       string `json:"html_url"`
	UpdatedAt     string `json:"updated_at"`
}

// ListRepositories returns every repository an installation can see.
//
// Paged through to the end: an account that granted access to all of a large
// org would otherwise show the first thirty and appear to be missing the rest.
func (a *App) ListRepositories(ctx context.Context, installationID string) ([]Repository, error) {
	token, err := a.InstallationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}

	var all []Repository
	for page := 1; page <= 20; page++ {
		req, err := http.NewRequestWithContext(ctx, "GET",
			fmt.Sprintf("%s/installation/repositories?per_page=100&page=%d", apiBase, page), nil)
		if err != nil {
			return nil, err
		}
		setHeaders(req, "Bearer "+token)

		resp, err := a.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("github app: list repositories: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, apiError("list repositories", resp.StatusCode, body)
		}

		var out struct {
			TotalCount   int          `json:"total_count"`
			Repositories []Repository `json:"repositories"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, fmt.Errorf("github app: list repositories: %w", err)
		}
		all = append(all, out.Repositories...)
		if len(out.Repositories) < 100 || len(all) >= out.TotalCount {
			break
		}
	}
	return all, nil
}

// CloneURL returns a URL that clones a private repository.
//
// Git has no notion of a bearer token, so the installation token rides in the
// URL as a password. It expires within the hour, which is the point: a build
// log that leaked one would leak something already dead.
func CloneURL(repoURL, token string) string {
	if token == "" || !strings.HasPrefix(repoURL, "https://") {
		return repoURL
	}
	return "https://x-access-token:" + token + "@" + strings.TrimPrefix(repoURL, "https://")
}

// ParseRepoURL pulls the owner and repository out of a GitHub URL, in any of
// the forms somebody might paste.
func ParseRepoURL(raw string) (owner, repo string, ok bool) {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")

	switch {
	case strings.HasPrefix(s, "git@github.com:"):
		s = strings.TrimPrefix(s, "git@github.com:")
	case strings.Contains(s, "github.com/"):
		s = s[strings.Index(s, "github.com/")+len("github.com/"):]
	default:
		return "", "", false
	}

	parts := strings.Split(s, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// VerifyWebhook reports whether a delivery really came from GitHub.
//
// The app has one webhook secret for every installation, so this is the only
// thing standing between the endpoint and anyone who can guess the URL.
func (a *App) VerifyWebhook(signature string, body []byte) bool {
	if a.webhookSecret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(a.webhookSecret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// PostCommitStatus reports a deploy against the commit that caused it.
func (a *App) PostCommitStatus(ctx context.Context, installationID, owner, repo, sha, state, targetURL, description string) error {
	token, err := a.InstallationToken(ctx, installationID)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{
		"state":       state,
		"target_url":  targetURL,
		"description": description,
		"context":     "applad/deploy",
	})
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/repos/%s/%s/statuses/%s", apiBase, owner, repo, sha),
		strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	setHeaders(req, "Bearer "+token)

	resp, err := a.http.Do(req)
	if err != nil {
		return fmt.Errorf("github app: commit status: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return apiError("commit status", resp.StatusCode, body)
	}
	return nil
}

// do performs a request authenticated as the app itself.
func (a *App) do(ctx context.Context, method, path, body string, out interface{}) error {
	jwtToken, err := a.appJWT()
	if err != nil {
		return fmt.Errorf("github app: sign jwt: %w", err)
	}

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, apiBase+path, reader)
	if err != nil {
		return err
	}
	setHeaders(req, "Bearer "+jwtToken)

	resp, err := a.http.Do(req)
	if err != nil {
		return fmt.Errorf("github app: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return apiError(method+" "+path, resp.StatusCode, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("github app: %s %s: %w", method, path, err)
		}
	}
	return nil
}

func setHeaders(req *http.Request, auth string) {
	req.Header.Set("Authorization", auth)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "applad")
	if req.Method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}
}

// apiError turns GitHub's error body into one line, since the full body is a
// documentation URL and a request id nobody can act on.
func apiError(what string, status int, body []byte) error {
	var out struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &out)
	if out.Message == "" {
		out.Message = strings.TrimSpace(string(body))
	}
	if len(out.Message) > 200 {
		out.Message = out.Message[:200]
	}
	return fmt.Errorf("github app: %s: %d %s", what, status, out.Message)
}

// FromConfig builds the app from loaded configuration, returning
// ErrNotConfigured when this instance has no GitHub App — the normal case for
// a self-hosted install.
func FromConfig(c *config.Config) (*App, error) {
	return New(Config{
		AppID:         c.GitHubAppID,
		Slug:          c.GitHubAppSlug,
		ClientID:      c.GitHubAppClientID,
		ClientSecret:  c.GitHubAppClientSecret,
		WebhookSecret: c.GitHubAppWebhookSecret,
		PrivateKey:    c.GitHubAppPrivateKey,
	})
}

// Branch is one branch of a repository.
type Branch struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
}

// ListBranches returns a repository's branches and which one is its default.
//
// Typed by hand, a branch is a guess that fails at clone time — "Remote branch
// main not found in upstream origin" for a repository whose default is
// something else entirely.
func (a *App) ListBranches(ctx context.Context, installationID, owner, repo string) ([]Branch, string, error) {
	token, err := a.InstallationToken(ctx, installationID)
	if err != nil {
		return nil, "", err
	}

	var meta struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := a.get(ctx, token, fmt.Sprintf("/repos/%s/%s", url.PathEscape(owner), url.PathEscape(repo)), &meta); err != nil {
		return nil, "", err
	}

	var branches []Branch
	for page := 1; page <= 10; page++ {
		var batch []Branch
		if err := a.get(ctx, token,
			fmt.Sprintf("/repos/%s/%s/branches?per_page=100&page=%d", url.PathEscape(owner), url.PathEscape(repo), page),
			&batch); err != nil {
			return nil, "", err
		}
		branches = append(branches, batch...)
		if len(batch) < 100 {
			break
		}
	}
	return branches, meta.DefaultBranch, nil
}

// get performs a request authenticated as an installation.
func (a *App) get(ctx context.Context, token, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", apiBase+path, nil)
	if err != nil {
		return err
	}
	setHeaders(req, "Bearer "+token)

	resp, err := a.http.Do(req)
	if err != nil {
		return fmt.Errorf("github app: GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return apiError("GET "+path, resp.StatusCode, body)
	}
	return json.Unmarshal(body, out)
}
