package apperr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrite(t *testing.T) {
	w := httptest.NewRecorder()
	Write(w, http.StatusBadRequest, "test_error", "something went wrong")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var e Error
	if err := json.NewDecoder(w.Body).Decode(&e); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if e.Type != "test_error" {
		t.Fatalf("expected type 'test_error', got %s", e.Type)
	}
	if e.Message != "something went wrong" {
		t.Fatalf("expected message 'something went wrong', got %s", e.Message)
	}
	if e.Code != 400 {
		t.Fatalf("expected code 400, got %d", e.Code)
	}
	if e.Version != "1.0.0" {
		t.Fatalf("expected version '1.0.0', got %s", e.Version)
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "user")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var e Error
	json.NewDecoder(w.Body).Decode(&e)
	if e.Type != "user_not_found" {
		t.Fatalf("expected type 'user_not_found', got %s", e.Type)
	}
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	Unauthorized(w)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "missing field")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var e Error
	json.NewDecoder(w.Body).Decode(&e)
	if e.Message != "missing field" {
		t.Fatalf("expected 'missing field', got %s", e.Message)
	}
}

func TestInternal(t *testing.T) {
	w := httptest.NewRecorder()
	Internal(w, nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	var e Error
	json.NewDecoder(w.Body).Decode(&e)
	// Should not leak internal error details
	if e.Message != "Internal server error" {
		t.Fatalf("expected generic message, got %s", e.Message)
	}
}

func TestConflict(t *testing.T) {
	w := httptest.NewRecorder()
	Conflict(w, "already exists")
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}
