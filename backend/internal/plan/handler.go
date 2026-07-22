package plan

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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
	r.Get("/items/{itemId}/criteria", h.listCriteria)
	r.Post("/items/{itemId}/criteria", h.addCriterion)
	r.Patch("/criteria/{criterionId}", h.updateCriterion)
	r.Delete("/criteria/{criterionId}", h.deleteCriterion)

	r.Post("/items/{itemId}/rate", h.rate)
	r.Get("/questions", h.listQuestions)
	r.Get("/items/{itemId}/questions", h.listQuestions)
	r.Post("/items/{itemId}/answers", h.answer)
	r.Get("/matrix", h.getMatrix)
	r.Put("/matrix", h.setMatrixCell)
	r.Get("/items/{itemId}/activity", h.listActivity)
	r.Get("/items/{itemId}/comments", h.listComments)
	r.Post("/items/{itemId}/comments", h.addComment)
	r.Delete("/comments/{commentId}", h.deleteComment)

	r.Get("/milestones", h.listMilestones)
	r.Post("/milestones", h.createMilestone)
	r.Patch("/milestones/{milestoneId}", h.updateMilestone)
	r.Delete("/milestones/{milestoneId}", h.deleteMilestone)

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
		MilestoneID:   q.Get("milestoneId"),
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

	item, err := h.svc.UpdateAs(r.Context(), chi.URLParam(r, "itemId"), projectID,
		middleware.UserFromContext(r.Context()), body.input())
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

func (h *Handler) listCriteria(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListCriteria(r.Context(), chi.URLParam(r, "itemId"))
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"criteria": list})
}

func (h *Handler) addCriterion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text string `json:"text"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck

	c, err := h.svc.AddCriterion(r.Context(), chi.URLParam(r, "itemId"),
		middleware.ProjectFromContext(r.Context()), body.Text)
	if err != nil {
		apperr.BadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) updateCriterion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text    *string `json:"text"`
		SpecRef *string `json:"specRef"`
		Met     *bool   `json:"met"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.UpdateCriterion(r.Context(), chi.URLParam(r, "criterionId"),
		middleware.ProjectFromContext(r.Context()), body.Text, body.SpecRef, body.Met); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteCriterion(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteCriterion(r.Context(), chi.URLParam(r, "criterionId"),
		middleware.ProjectFromContext(r.Context())); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) rate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Impact  int `json:"impact"`
		Urgency int `json:"urgency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}

	item, err := h.svc.Rate(r.Context(), chi.URLParam(r, "itemId"),
		middleware.ProjectFromContext(r.Context()), middleware.UserFromContext(r.Context()),
		body.Impact, body.Urgency)
	if err != nil {
		apperr.BadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) listQuestions(w http.ResponseWriter, r *http.Request) {
	questions, err := h.svc.Questions(r.Context(), middleware.ProjectFromContext(r.Context()),
		chi.URLParam(r, "itemId"))
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"questions": questions})
}

func (h *Handler) answer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		QuestionID string `json:"questionId"`
		OptionID   string `json:"optionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}

	item, err := h.svc.Answer(r.Context(), chi.URLParam(r, "itemId"),
		middleware.ProjectFromContext(r.Context()), middleware.UserFromContext(r.Context()),
		body.QuestionID, body.OptionID)
	if err != nil {
		apperr.BadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) getMatrix(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "change"
	}
	cells, err := h.svc.Grid(r.Context(), middleware.ProjectFromContext(r.Context()), kind)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"kind": kind, "cells": cells})
}

func (h *Handler) setMatrixCell(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind     string `json:"kind"`
		Impact   int    `json:"impact"`
		Urgency  int    `json:"urgency"`
		Priority string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Kind == "" {
		body.Kind = "change"
	}
	if err := h.svc.SetCell(r.Context(), middleware.ProjectFromContext(r.Context()),
		body.Kind, body.Impact, body.Urgency, body.Priority); err != nil {
		apperr.BadRequest(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listActivity(w http.ResponseWriter, r *http.Request) {
	events, err := h.svc.ListActivity(r.Context(), chi.URLParam(r, "itemId"))
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"activity": events})
}

func (h *Handler) listComments(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListComments(r.Context(), chi.URLParam(r, "itemId"))
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"comments": list})
}

func (h *Handler) addComment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Body string `json:"body"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck

	c, err := h.svc.AddComment(r.Context(), chi.URLParam(r, "itemId"),
		middleware.ProjectFromContext(r.Context()), middleware.UserFromContext(r.Context()), body.Body)
	if err != nil {
		apperr.BadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteComment(r.Context(), chi.URLParam(r, "commentId"),
		middleware.ProjectFromContext(r.Context())); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listMilestones(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListMilestones(r.Context(), middleware.ProjectFromContext(r.Context()))
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(list), "milestones": list})
}

func (h *Handler) createMilestone(w http.ResponseWriter, r *http.Request) {
	var body milestoneBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	m, err := h.svc.CreateMilestone(r.Context(), middleware.ProjectFromContext(r.Context()),
		body.Name, body.Description, body.target())
	if err != nil {
		apperr.BadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (h *Handler) updateMilestone(w http.ResponseWriter, r *http.Request) {
	var body milestoneBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.UpdateMilestone(r.Context(), chi.URLParam(r, "milestoneId"),
		middleware.ProjectFromContext(r.Context()), body.Name, body.Description,
		body.target(), body.Completed); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteMilestone(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteMilestone(r.Context(), chi.URLParam(r, "milestoneId"),
		middleware.ProjectFromContext(r.Context())); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// milestoneBody takes the target date as a plain date string, which is what a
// date means here: a day somebody is aiming at, not an instant.
type milestoneBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TargetDate  string `json:"targetDate"`
	Completed   *bool  `json:"completed"`
}

func (b milestoneBody) target() *time.Time {
	if strings.TrimSpace(b.TargetDate) == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", b.TargetDate)
	if err != nil {
		return nil
	}
	return &t
}

func (h *Handler) meta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"statuses":   Statuses,
		"priorities": Priorities,
		"kinds":      Kinds,
	})
}

// itemBody is the wire shape. Pointers throughout, because a field that was
// not mentioned and a field that was cleared are different requests, and a
// PATCH that cannot tell them apart erases whatever it was not told about.
type itemBody struct {
	Title       *string   `json:"title"`
	Body        *string   `json:"body"`
	Status      *string   `json:"status"`
	Priority    *string   `json:"priority"`
	AssigneeID  *string   `json:"assigneeId"`
	ParentID    *string   `json:"parentId"`
	Labels      *[]string `json:"labels"`
	Rank        *int64    `json:"rank"`
	Kind        *string   `json:"kind"`
	MilestoneID *string   `json:"milestoneId"`
	TargetDate  *string   `json:"targetDate"`
}

func (b itemBody) input() Input {
	return Input{
		Title: b.Title, Body: b.Body, Status: b.Status, Priority: b.Priority,
		AssigneeID: b.AssigneeID, ParentID: b.ParentID, Labels: b.Labels, Rank: b.Rank,
		Kind: b.Kind, MilestoneID: b.MilestoneID, TargetDate: parseDate(b.TargetDate),
	}
}

// parseDate reads a target date, which is a day somebody is aiming at rather
// than an instant.
func parseDate(v *string) *time.Time {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *v)
	if err != nil {
		return nil
	}
	return &t
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
