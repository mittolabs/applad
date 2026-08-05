package transfer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mittolabs/applad/internal/netguard"
)

// errTooLarge is returned when a downloaded object exceeds the import size cap.
var errTooLarge = errors.New("transfer: object exceeds import size limit")

// External source adapters connect to hosts named in caller-supplied
// credentials, so every outbound connection (Postgres and HTTP) is guarded the
// same way workflow HTTP nodes are: the destination IP is checked AFTER DNS
// resolution, which refuses loopback/RFC1918/metadata and covers DNS rebinding.

// openGuardedPostgres opens an external Postgres pool whose dials are refused if
// they resolve to a non-public address. Used by the Supabase and NHost sources.
func openGuardedPostgres(dsn string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("transfer: parse postgres dsn: %w", err)
	}
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			return netguard.CheckAddr(address)
		},
	}
	cfg.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}
	return stdlib.OpenDB(*cfg), nil
}

// httpJSON is a small guarded JSON HTTP client for the REST-based sources
// (Appwrite, Firebase). It never follows a redirect to a private address and
// never logs the response body (which may carry secrets/hashes).
type httpJSON struct {
	client  *http.Client
	headers map[string]string
}

func newHTTPJSON(headers map[string]string) *httpJSON {
	return &httpJSON{client: netguard.Client(30 * time.Second), headers: headers}
}

func (h *httpJSON) getInto(ctx context.Context, url string, out any) error {
	return h.do(ctx, http.MethodGet, url, nil, out)
}

func (h *httpJSON) do(ctx context.Context, method, url string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read a bounded slice for the error, but do not surface full bodies.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("transfer: %s %s: status %d: %s", method, redact(url), resp.StatusCode, string(snippet))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// getBytes downloads raw content (file bytes) up to maxBytes, guarded.
func (h *httpJSON) getBytes(ctx context.Context, url string, maxBytes int64) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("transfer: download %s: status %d", redact(url), resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxBytes {
		return nil, "", errTooLarge
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// redact strips a query string (which may carry a token) from a URL for errors.
func redact(url string) string {
	if i := indexByte(url, '?'); i >= 0 {
		return url[:i] + "?…"
	}
	return url
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
