package transfer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mittolabs/applad/internal/netguard"
)

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func TestWireRoundTrip(t *testing.T) {
	in := []Resource{
		User{ID: "u1", Email: "a@x.io", PasswordHash: "h", PasswordAlgo: "bcrypt", PasswordParams: map[string]any{"salt": "s"}},
		Database{ID: "db1", Name: "Main"},
		Row{DatabaseID: "db1", TableID: "t1", ID: "r1", Data: map[string]any{"title": "hi"}},
	}
	for _, r := range in {
		wr := wireResource{Kind: r.Kind(), Data: mustJSON(r)}
		got, err := decodeWireResource(wr)
		if err != nil {
			t.Fatalf("decode %s: %v", r.Kind(), err)
		}
		if got.Kind() != r.Kind() || got.SourceID() != r.SourceID() {
			t.Fatalf("round-trip mismatch for %s: got kind=%s id=%s", r.Kind(), got.Kind(), got.SourceID())
		}
	}
}

func TestRemoteAppladExport(t *testing.T) {
	t.Setenv("ALLOW_PRIVATE_EGRESS", "true") // allow the loopback test server

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Applad-Project") != "p1" || r.Header.Get("X-Applad-Key") != "k1" {
			t.Errorf("missing/incorrect auth headers")
		}
		if r.URL.Query().Get("report") == "1" {
			_ = json.NewEncoder(w).Encode(map[string]any{"counts": map[string]int{"auth": 2}})
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		enc := json.NewEncoder(w)
		_ = enc.Encode(wireResource{Kind: "user", Data: mustJSON(User{ID: "u1", Email: "a@x.io", PasswordHash: "h", PasswordAlgo: "bcrypt"})})
		_ = enc.Encode(wireResource{Kind: "database", Data: mustJSON(Database{ID: "db1", Name: "Main"})})
	}))
	defer srv.Close()

	src := &remoteAppladSource{endpoint: srv.URL, projectID: "p1", apiKey: "k1", http: netguard.Client(0)}

	// Report
	counts, err := src.Report(context.Background(), []Group{GroupAuth})
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if counts[GroupAuth] != 2 {
		t.Fatalf("report auth = %d, want 2", counts[GroupAuth])
	}

	// Export
	var got []Resource
	err = src.Export(context.Background(), []Group{GroupAuth, GroupDatabases}, func(_ context.Context, rs []Resource) error {
		got = append(got, rs...)
		return nil
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(got))
	}
	u, ok := got[0].(User)
	if !ok || u.Email != "a@x.io" || u.PasswordAlgo != "bcrypt" {
		t.Fatalf("user decoded wrong: %+v", got[0])
	}
}

func TestRemoteAppladExportErrorLine(t *testing.T) {
	t.Setenv("ALLOW_PRIVATE_EGRESS", "true")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		enc := json.NewEncoder(w)
		_ = enc.Encode(wireResource{Kind: "user", Data: mustJSON(User{ID: "u1"})})
		_ = enc.Encode(wireResource{Kind: "error", Data: mustJSON(map[string]string{"message": "read failed"})})
	}))
	defer srv.Close()

	src := &remoteAppladSource{endpoint: srv.URL, projectID: "p1", apiKey: "k1", http: netguard.Client(0)}
	err := src.Export(context.Background(), nil, func(_ context.Context, _ []Resource) error { return nil })
	if err == nil {
		t.Fatal("expected an error from the terminal error line")
	}
}
