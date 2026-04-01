package apperr

import (
	"encoding/json"
	"net/http"
)

// Error is the Appwrite-compatible error response shape.
type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

// Write writes a JSON error response with the given HTTP status code.
func Write(w http.ResponseWriter, code int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(Error{
		Message: message,
		Code:    code,
		Type:    errType,
		Version: "1.0.0",
	})
}

// NotFound writes a 404 error.
func NotFound(w http.ResponseWriter, resource string) {
	Write(w, http.StatusNotFound, resource+"_not_found", resource+" not found")
}

// Unauthorized writes a 401 error.
func Unauthorized(w http.ResponseWriter) {
	Write(w, http.StatusUnauthorized, "general_unauthorized_scope", "Missing or invalid credentials. Please check the endpoint, headers or params according to the documentation.")
}

// BadRequest writes a 400 error.
func BadRequest(w http.ResponseWriter, message string) {
	Write(w, http.StatusBadRequest, "general_argument_invalid", message)
}

// Internal writes a 500 error.
func Internal(w http.ResponseWriter, _ error) {
	Write(w, http.StatusInternalServerError, "general_server_error", "Internal server error")
}

// Conflict writes a 409 error.
func Conflict(w http.ResponseWriter, message string) {
	Write(w, http.StatusConflict, "general_query_invalid", message)
}
