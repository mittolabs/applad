package deploy

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/githubapp"
	"github.com/mittolabs/applad/internal/middleware"
)

/*
 * The GitHub App's own endpoints.
 *
 * A GitHub App has one webhook URL for every account that installs it, so
 * deliveries arrive identified by installation rather than by a URL Applad
 * handed out. That is the whole difference from the per-connection webhook
 * kept alongside it for GitLab and for anyone wiring a repository up by hand:
 * there, the URL is the identity and the secret is per connection; here, one
 * secret verifies everything and the payload says who it is about.
 */

// gitHubAppRoutes adds the project-scoped endpoints for connecting an account.
func gitHubAppRoutes(r chi.Router, h *Handler) {
	r.Get("/git/github/install-url", h.gitHubInstallURL)
	r.Post("/git/github/installations", h.gitHubCompleteInstall)
}

// gitHubInstallURL hands back where to send somebody to install the app.
func (h *Handler) gitHubInstallURL(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())

	url, err := h.svc.GitHubInstallURL(r.Context(), projectID)
	if errors.Is(err, githubapp.ErrNotConfigured) {
		// Not a failure: a self-hosted instance without an app of its own
		// should be told that plainly rather than shown a broken button.
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"configured": false,
			"reason":     "This Applad instance has no GitHub App configured.",
		})
		return
	}
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"configured": true,
		"url":        url,
		"slug":       h.svc.GitHubApp().Slug(),
	})
}

// gitHubCompleteInstall records an installation against the project that
// started it. Called by the console when GitHub sends somebody back.
func (h *Handler) gitHubCompleteInstall(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())

	var body struct {
		InstallationID string `json:"installationId"`
		State          string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}

	conn, err := h.svc.CompleteGitHubInstall(r.Context(), projectID, body.InstallationID, body.State)
	switch {
	case errors.Is(err, githubapp.ErrNotConfigured):
		apperr.Write(w, http.StatusNotImplemented, "github_app_not_configured",
			"This Applad instance has no GitHub App configured.")
		return
	case err != nil:
		apperr.BadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, conn)
}

// handleGitHubAppWebhook receives every delivery for the app.
//
// Mounted at POST /v1/deploy/git/webhook — no connection id, because GitHub
// only knows the one URL. The signature is checked against the app's secret
// before the body is trusted for anything, including deciding whose it is.
func (h *Handler) handleGitHubAppWebhook(w http.ResponseWriter, r *http.Request) {
	app := h.svc.GitHubApp()
	if app == nil {
		apperr.Write(w, http.StatusNotImplemented, "github_app_not_configured",
			"This Applad instance has no GitHub App configured.")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 5<<20))
	if err != nil {
		apperr.BadRequest(w, "failed to read request body")
		return
	}
	if !app.VerifyWebhook(r.Header.Get("X-Hub-Signature-256"), body) {
		apperr.Write(w, http.StatusUnauthorized, "webhook_unauthorized",
			"signature verification failed")
		return
	}

	event := r.Header.Get("X-GitHub-Event")

	var envelope struct {
		Action       string `json:"action"`
		Installation struct {
			ID int64 `json:"id"`
		} `json:"installation"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		apperr.BadRequest(w, "unparseable payload")
		return
	}
	installationID := strconv.FormatInt(envelope.Installation.ID, 10)

	// Somebody removed Applad from their account. Their connections cannot
	// reach anything any more, so they should stop being listed as if they can.
	if event == "installation" && (envelope.Action == "deleted" || envelope.Action == "suspend") {
		if err := h.svc.RemoveInstallation(r.Context(), installationID); err != nil {
			slog.Error("deploy: forget installation", "installation_id", installationID, "error", err)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}

	if event != "push" && event != "pull_request" {
		// ping, installation created, repositories added — nothing to build.
		writeJSON(w, http.StatusOK, map[string]interface{}{"triggered": 0})
		return
	}

	conns, err := h.svc.ConnectionsByInstallation(r.Context(), installationID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	// One installation can serve several projects — an agency with the app on
	// one org and a project per client. Each gets the event.
	triggered := 0
	for _, conn := range conns {
		n, err := h.svc.DispatchGitEvent(r.Context(), conn, event, body)
		if err != nil {
			slog.Error("deploy: github webhook", "connection_id", conn.ID, "event", event, "error", err)
			continue
		}
		triggered += n
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"triggered": triggered})
}
