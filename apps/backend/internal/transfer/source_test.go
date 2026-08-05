package transfer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mittolabs/applad/internal/auth"
)

func TestMapAppwriteHash(t *testing.T) {
	if a, _ := mapAppwriteHash("bcrypt", nil); a != auth.AlgoBcrypt {
		t.Errorf("bcrypt -> %q", a)
	}
	if a, _ := mapAppwriteHash("argon2", nil); a != auth.AlgoArgon2id {
		t.Errorf("argon2 -> %q", a)
	}
	if a, p := mapAppwriteHash("scryptMod", map[string]any{"signerKey": "k", "salt": "s"}); a != auth.AlgoScryptFirebase || p["signerKey"] != "k" {
		t.Errorf("scryptMod -> %q %v", a, p)
	}
	// Unknown algorithm is passed through under its own name so login fails
	// cleanly rather than being silently accepted.
	if a, _ := mapAppwriteHash("phpass", nil); a != "phpass" {
		t.Errorf("phpass -> %q", a)
	}
}

func TestDecodeFirestoreValue(t *testing.T) {
	cases := map[string]any{
		`{"stringValue":"hi"}`:  "hi",
		`{"integerValue":"42"}`: "42",
		`{"booleanValue":true}`: true,
		`{"nullValue":null}`:    nil,
		`{"doubleValue":1.5}`:   1.5,
	}
	for in, want := range cases {
		got := decodeFirestoreValue(json.RawMessage(in))
		if got != want {
			t.Errorf("decode %s = %v, want %v", in, got, want)
		}
	}
	// Nested map + array.
	m := decodeFirestoreValue(json.RawMessage(`{"mapValue":{"fields":{"a":{"stringValue":"x"}}}}`))
	mm, ok := m.(map[string]any)
	if !ok || mm["a"] != "x" {
		t.Errorf("map decode failed: %v", m)
	}
	arr := decodeFirestoreValue(json.RawMessage(`{"arrayValue":{"values":[{"integerValue":"1"},{"stringValue":"y"}]}}`))
	av, ok := arr.([]any)
	if !ok || len(av) != 2 || av[1] != "y" {
		t.Errorf("array decode failed: %v", arr)
	}
}

func TestBuildPGDSN(t *testing.T) {
	// Valid input produces an sslmode=require URL with escaped credentials.
	dsn, err := buildPGDSN("db.example.com", "6543", "user", "p@ss word", "postgres")
	if err != nil {
		t.Fatalf("valid dsn rejected: %v", err)
	}
	if !contains(dsn, "sslmode=require") || !contains(dsn, "p%40ss+word") {
		t.Fatalf("dsn not built safely: %s", dsn)
	}
	// Injection attempts in structural parts are rejected, so an attacker cannot
	// append DSN options (e.g. flip sslmode) via host/port/database.
	bad := [][3]string{
		{"host?sslmode=disable", "5432", "db"},
		{"host", "5432 sslmode=disable", "db"},
		{"host", "5432", "db?options=x"},
		{"h ost", "5432", "db"},
	}
	for _, c := range bad {
		if _, err := buildPGDSN(c[0], c[1], "u", "p", c[2]); err == nil {
			t.Errorf("expected rejection for host=%q port=%q db=%q", c[0], c[1], c[2])
		}
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (indexByte2(s, sub) >= 0) }
func indexByte2(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestStripSystemKeys(t *testing.T) {
	in := map[string]any{"$id": "1", "$permissions": []any{}, "title": "hello", "count": 3.0}
	out := stripSystemKeys(in)
	if _, ok := out["$id"]; ok {
		t.Error("$id not stripped")
	}
	if out["title"] != "hello" || out["count"] != 3.0 {
		t.Errorf("user fields lost: %v", out)
	}
}

func TestAppwriteExportUsers(t *testing.T) {
	// netguard refuses loopback by default; allow it for this in-process server.
	t.Setenv("ALLOW_PRIVATE_EGRESS", "true")

	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Appwrite-Project") == "" || r.Header.Get("X-Appwrite-Key") == "" {
			t.Errorf("missing appwrite auth headers")
		}
		w.Header().Set("Content-Type", "application/json")
		if page == 0 {
			page++
			_, _ = w.Write([]byte(`{"total":1,"users":[{"$id":"u1","email":"a@x.io","name":"A","emailVerification":true,"password":"$2a$hash","hash":"bcrypt"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"total":1,"users":[]}`))
	}))
	defer srv.Close()

	src := &appwriteSource{
		base: srv.URL,
		http: newHTTPJSON(map[string]string{"X-Appwrite-Project": "p", "X-Appwrite-Key": "k"}),
	}

	var got []Resource
	err := src.exportUsers(context.Background(), func(_ context.Context, rs []Resource) error {
		got = append(got, rs...)
		return nil
	})
	if err != nil {
		t.Fatalf("exportUsers: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 user, got %d", len(got))
	}
	u := got[0].(User)
	if u.Email != "a@x.io" || u.PasswordAlgo != auth.AlgoBcrypt || u.PasswordHash != "$2a$hash" || !u.EmailVerified {
		t.Fatalf("user mapped wrong: %+v", u)
	}
}
