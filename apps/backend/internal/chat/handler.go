package chat

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	mw "github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/model"
)

// Handler handles HTTP requests for chat.
type Handler struct {
	svc *Service
}

// NewHandler creates a new chat Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// Routes returns the chat router. Every route requires an authenticated end
// user session — chat is inherently between users, so (unlike most other
// services) a server API key alone cannot act as a participant. Admin/
// moderation endpoints for server SDKs are a later milestone.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Devices
	r.Post("/devices", h.registerDevice)
	r.Get("/devices", h.listDevices)
	r.Delete("/devices/{deviceId}", h.revokeDevice)
	r.Post("/devices/{deviceId}/prekeys", h.topUpPrekeys)
	r.Get("/devices/{deviceId}/prekey-bundle", h.getPrekeyBundle)
	r.Get("/users/{userId}/devices", h.listUserDevices)

	// Conversations
	r.Post("/conversations", h.createConversation)
	r.Get("/conversations", h.listConversations)
	r.Get("/conversations/{conversationId}", h.getConversation)

	// Messages
	r.Post("/conversations/{conversationId}/messages", h.sendMessage)
	r.Get("/conversations/{conversationId}/messages", h.listMessages)
	r.Post("/messages/{messageId}/ack", h.ackMessage)

	return r
}

// requireUser extracts the caller's user id, requiring a real end-user
// session rather than a server API key — chat participants are always
// people, and a server SDK acts through the (separate, admin/moderation
// only) relay-role surface, not these endpoints.
func requireUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	if mw.IsAPIKey(r.Context()) {
		apperr.Write(w, http.StatusForbidden, "chat_requires_user_session",
			"This endpoint requires an authenticated user session, not a server API key.")
		return "", false
	}
	userID := mw.UserFromContext(r.Context())
	if userID == "" {
		apperr.Unauthorized(w)
		return "", false
	}
	return userID, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apperr.NotFound(w, "chat_resource")
	case errors.Is(err, ErrForbidden):
		apperr.Write(w, http.StatusForbidden, "chat_forbidden", "You do not have access to this chat resource.")
	default:
		apperr.BadRequest(w, err.Error())
	}
}

// ── Devices ──────────────────────────────────────────────────────────────────

func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		DeviceID        string               `json:"deviceId"`
		Name            string               `json:"name"`
		RegistrationID  int                  `json:"registrationId"`
		IdentityKey     string               `json:"identityKey"`
		SignedPrekeyID  int                  `json:"signedPrekeyId"`
		SignedPrekey    string               `json:"signedPrekey"`
		SignedPrekeySig string               `json:"signedPrekeySig"`
		PushToken       string               `json:"pushToken"`
		PushProvider    string               `json:"pushProvider"`
		OneTimePrekeys  []OneTimePrekeyInput `json:"oneTimePrekeys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	device, err := h.svc.RegisterDevice(r.Context(), projectID, userID, RegisterDeviceInput{
		DeviceID: body.DeviceID, Name: body.Name, RegistrationID: body.RegistrationID,
		IdentityKey: body.IdentityKey, SignedPrekeyID: body.SignedPrekeyID,
		SignedPrekey: body.SignedPrekey, SignedPrekeySig: body.SignedPrekeySig,
		PushToken: body.PushToken, PushProvider: body.PushProvider, OneTimePrekeys: body.OneTimePrekeys,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, device)
}

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	devices, err := h.svc.ListDevices(r.Context(), projectID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"devices": devices, "total": len(devices)})
}

func (h *Handler) revokeDevice(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	deviceID := chi.URLParam(r, "deviceId")
	if err := h.svc.RevokeDevice(r.Context(), projectID, userID, deviceID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) topUpPrekeys(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	deviceID := chi.URLParam(r, "deviceId")
	var body struct {
		OneTimePrekeys []OneTimePrekeyInput `json:"oneTimePrekeys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.TopUpPrekeys(r.Context(), projectID, userID, deviceID, body.OneTimePrekeys); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getPrekeyBundle(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	if _, ok := requireUser(w, r); !ok {
		return
	}
	deviceID := chi.URLParam(r, "deviceId")
	bundle, err := h.svc.GetPrekeyBundle(r.Context(), projectID, deviceID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bundle)
}

func (h *Handler) listUserDevices(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	if _, ok := requireUser(w, r); !ok {
		return
	}
	targetUserID := chi.URLParam(r, "userId")
	devices, err := h.svc.ListUserDevices(r.Context(), projectID, targetUserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"devices": devices, "total": len(devices)})
}

// ── Conversations ────────────────────────────────────────────────────────────

func (h *Handler) createConversation(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		UserID string `json:"userId"`
		Title  string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	conv, err := h.svc.CreateConversation(r.Context(), projectID, userID, body.UserID, body.Title)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

func (h *Handler) listConversations(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	convs, err := h.svc.ListConversations(r.Context(), projectID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"conversations": convs, "total": len(convs)})
}

func (h *Handler) getConversation(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	conversationID := chi.URLParam(r, "conversationId")
	conv, members, err := h.svc.GetConversation(r.Context(), projectID, userID, conversationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"conversation": conv, "members": members})
}

// ── Messages ─────────────────────────────────────────────────────────────────

func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	conversationID := chi.URLParam(r, "conversationId")
	var body struct {
		ClientMessageID string `json:"clientMessageId"`
		SenderDeviceID  string `json:"senderDeviceId"`
		EnvelopeType    string `json:"envelopeType"`
		Targets         []struct {
			DeviceID   string `json:"deviceId"`
			Ciphertext string `json:"ciphertext"`
		} `json:"targets"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	targets := make([]model.MessageTarget, len(body.Targets))
	for i, t := range body.Targets {
		targets[i] = model.MessageTarget{DeviceID: t.DeviceID, Ciphertext: t.Ciphertext}
	}
	msg, err := h.svc.SendMessage(r.Context(), projectID, userID, conversationID, SendMessageInput{
		ClientMessageID: body.ClientMessageID, SenderDeviceID: body.SenderDeviceID,
		EnvelopeType: body.EnvelopeType, Targets: targets,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	conversationID := chi.URLParam(r, "conversationId")
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		apperr.BadRequest(w, "device_id is required")
		return
	}
	afterSeq := queryInt64(r, "after_seq", 0)
	limit := queryInt(r, "limit", 100)

	messages, err := h.svc.ListMessages(r.Context(), projectID, userID, conversationID, deviceID, afterSeq, limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"messages": messages, "total": len(messages)})
}

func (h *Handler) ackMessage(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	messageID := chi.URLParam(r, "messageId")
	var body struct {
		DeviceID string `json:"deviceId"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.AckMessage(r.Context(), projectID, userID, messageID, body.DeviceID, body.Status); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func queryInt64(r *http.Request, key string, def int64) int64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		return def
	}
	return n
}
