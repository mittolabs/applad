package plan

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler serves the plan API.
type Handler struct{ svc *Service }

// NewHandler creates a plan handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Routes returns the plan router, mounted at /v1/plan.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/items", h.list)
	r.Post("/items", h.create)
	r.Get("/items/{itemId}", h.get)
	r.Patch("/items/{itemId}", h.update)
	r.Delete("/items/{itemId}", h.delete)
	r.Post("/items/{itemId}/links", h.addLink)
	r.Delete("/links/{linkId}", h.removeLink)
	// What the console offers in its dropdowns, from the same list the server
	// validates against — so the two cannot drift into disagreeing.
	r.Get("/meta", h.meta)
	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	q := r.URL.Query()

	items, err := h.svc.List(r.Context(), projectID, Filter{
		Status:        q.Get("status"),
		Assignee:      q.Get("assignee"),
		Label:         q.Get("label"),
		Search:        q.Get("search"),
		ParentID:      q.Get("parentId"),
		IncludeClosed: q.Get("includeClosed") == "true",
	})
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(items), "items": items})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())

	var body itemBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}

	item, err := h.svc.Create(r.Context(), projectID, middleware.UserFromContext(r.Context()), body.input())
	if err != nil {
		apperr.BadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.Get(r.Context(), chi.URLParam(r, "itemId"),
		middleware.ProjectFromContext(r.Context()))
	if err != nil {
		apperr.NotFound(w, "plan item")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())

	var body itemBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}

	item, err := h.svc.Update(r.Context(), chi.URLParam(r, "itemId"), projectID, body.input())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "plan item")
			return
		}
		apperr.BadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "itemId"),
		middleware.ProjectFromContext(r.Context())); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind  string `json:"kind"`
		Ref   string `json:"ref"`
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}

	link, err := h.svc.AddLink(r.Context(), chi.URLParam(r, "itemId"),
		middleware.ProjectFromContext(r.Context()), body.Kind, body.Ref, body.Label)
	if err != nil {
		apperr.BadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

func (h *Handler) removeLink(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveLink(r.Context(), chi.URLParam(r, "linkId"),
		middleware.ProjectFromContext(r.Context())); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) meta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"statuses":   Statuses,
		"priorities": Priorities,
	})
}

// itemBody is the wire shape. Pointers throughout, because a field that was
// not mentioned and a field that was cleared are different requests, and a
// PATCH that cannot tell them apart erases whatever it was not told about.
type itemBody struct {
	Title      *string   `json:"title"`
	Body       *string   `json:"body"`
	Status     *string   `json:"status"`
	Priority   *string   `json:"priority"`
	AssigneeID *string   `json:"assigneeId"`
	ParentID   *string   `json:"parentId"`
	Labels     *[]string `json:"labels"`
	Rank       *int64    `json:"rank"`
}

func (b itemBody) input() Input {
	return Input{
		Title: b.Title, Body: b.Body, Status: b.Status, Priority: b.Priority,
		AssigneeID: b.AssigneeID, ParentID: b.ParentID, Labels: b.Labels, Rank: b.Rank,
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
