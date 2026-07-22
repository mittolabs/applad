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
	Limit       int
	Offset      int
	CursorAfter string // row/resource ID for cursor-based pagination
	OrderAttr   string
	OrderType   string // ASC or DESC
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

	return Pagination{
		Limit:       limit,
		Offset:      offset,
		CursorAfter: r.URL.Query().Get("cursorAfter"),
		OrderAttr:   r.URL.Query().Get("orderAttr"),
		OrderType:   r.URL.Query().Get("orderType"),
	}
}
