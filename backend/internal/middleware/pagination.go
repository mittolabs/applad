package middleware

import (
	"net/http"
	"strconv"
)

const (
	DefaultLimit = 25
	MaxLimit     = 100
)

// Pagination holds parsed pagination parameters.
type Pagination struct {
	Limit  int
	Offset int
}

// ParsePagination extracts limit and offset from query parameters
// with defaults and bounds.
func ParsePagination(r *http.Request) Pagination {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if offset < 0 {
		offset = 0
	}

	return Pagination{Limit: limit, Offset: offset}
}
