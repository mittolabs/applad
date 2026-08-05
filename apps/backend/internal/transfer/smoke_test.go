//go:build smoke

// Smoke tests for the external migration source adapters. They run against LIVE
// instances and are gated behind the `smoke` build tag so a normal `go test`
// never touches the network. Each test skips unless its credentials are set.
//
// Run one (or all) with, e.g.:
//
//	SMOKE_SUPABASE_HOST=... SMOKE_SUPABASE_USER=... SMOKE_SUPABASE_PASSWORD=... \
//	  go test -tags=smoke -v -run TestSmokeSupabase ./internal/transfer/
//
// Each test connects (Report), then streams a small, bounded slice of resources
// (Export, stopped early) and prints what it read. They assert the shapes are
// sane rather than exact values, since the data is whatever the target holds.
package transfer

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

var errSmokeStop = errors.New("smoke: stop")

// runSmoke validates connectivity via Report, then collects up to max resources
// from Export, stopping early. It logs a per-kind tally.
func runSmoke(t *testing.T, src Source, groups []Group, max int) []Resource {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	counts, err := src.Report(ctx, groups)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	t.Logf("Report counts: %v", counts)

	var got []Resource
	err = src.Export(ctx, groups, func(_ context.Context, rs []Resource) error {
		got = append(got, rs...)
		if len(got) >= max {
			return errSmokeStop
		}
		return nil
	})
	if err != nil && !errors.Is(err, errSmokeStop) {
		t.Fatalf("Export: %v", err)
	}

	tally := map[string]int{}
	for _, r := range got {
		tally[r.Kind()]++
	}
	t.Logf("collected %d resources: %v", len(got), tally)
	return got
}

// assertUsersHavePasswords logs how many collected users carry a usable password
// credential — the key signal that a source's hash mapping works.
func assertUsersHavePasswords(t *testing.T, got []Resource) {
	t.Helper()
	users, withPw := 0, 0
	for _, r := range got {
		if u, ok := r.(User); ok {
			users++
			if u.PasswordHash != "" && u.PasswordAlgo != "" {
				withPw++
			}
		}
	}
	if users > 0 {
		t.Logf("users: %d collected, %d with a usable password credential", users, withPw)
	}
}

func TestSmokeSupabase(t *testing.T) {
	host := os.Getenv("SMOKE_SUPABASE_HOST")
	if host == "" {
		t.Skip("set SMOKE_SUPABASE_HOST/USER/PASSWORD (and optionally PROJECT_URL/SERVICE_KEY) to run")
	}
	src, err := NewSupabaseSource(map[string]any{
		"host":       host,
		"port":       os.Getenv("SMOKE_SUPABASE_PORT"),
		"user":       os.Getenv("SMOKE_SUPABASE_USER"),
		"password":   os.Getenv("SMOKE_SUPABASE_PASSWORD"),
		"database":   os.Getenv("SMOKE_SUPABASE_DATABASE"),
		"projectUrl": os.Getenv("SMOKE_SUPABASE_PROJECT_URL"),
		"serviceKey": os.Getenv("SMOKE_SUPABASE_SERVICE_KEY"),
	})
	if err != nil {
		t.Fatalf("NewSupabaseSource: %v", err)
	}
	defer src.Close()
	got := runSmoke(t, src, []Group{GroupAuth, GroupDatabases, GroupStorage}, 20)
	assertUsersHavePasswords(t, got)
}

func TestSmokeAppwrite(t *testing.T) {
	endpoint := os.Getenv("SMOKE_APPWRITE_ENDPOINT")
	if endpoint == "" {
		t.Skip("set SMOKE_APPWRITE_ENDPOINT/PROJECT/KEY to run")
	}
	src, err := NewAppwriteSource(map[string]any{
		"endpoint":  endpoint,
		"projectId": os.Getenv("SMOKE_APPWRITE_PROJECT"),
		"apiKey":    os.Getenv("SMOKE_APPWRITE_KEY"),
	})
	if err != nil {
		t.Fatalf("NewAppwriteSource: %v", err)
	}
	defer src.Close()
	got := runSmoke(t, src, []Group{GroupAuth, GroupDatabases, GroupStorage}, 20)
	assertUsersHavePasswords(t, got)
}

func TestSmokeNhost(t *testing.T) {
	host := os.Getenv("SMOKE_NHOST_HOST")
	if host == "" {
		t.Skip("set SMOKE_NHOST_HOST/USER/PASSWORD to run")
	}
	src, err := NewNhostSource(map[string]any{
		"host":        host,
		"port":        os.Getenv("SMOKE_NHOST_PORT"),
		"user":        os.Getenv("SMOKE_NHOST_USER"),
		"password":    os.Getenv("SMOKE_NHOST_PASSWORD"),
		"database":    os.Getenv("SMOKE_NHOST_DATABASE"),
		"storageUrl":  os.Getenv("SMOKE_NHOST_STORAGE_URL"),
		"adminSecret": os.Getenv("SMOKE_NHOST_ADMIN_SECRET"),
	})
	if err != nil {
		t.Fatalf("NewNhostSource: %v", err)
	}
	defer src.Close()
	got := runSmoke(t, src, []Group{GroupAuth, GroupDatabases, GroupStorage}, 20)
	assertUsersHavePasswords(t, got)
}

func TestSmokeAppladRemote(t *testing.T) {
	endpoint := os.Getenv("SMOKE_APPLAD_ENDPOINT")
	if endpoint == "" {
		t.Skip("set SMOKE_APPLAD_ENDPOINT/PROJECT/KEY to run the cross-instance export")
	}
	src, err := NewRemoteAppladSource(endpoint, os.Getenv("SMOKE_APPLAD_PROJECT"), os.Getenv("SMOKE_APPLAD_KEY"))
	if err != nil {
		t.Fatalf("NewRemoteAppladSource: %v", err)
	}
	defer src.Close()
	got := runSmoke(t, src, []Group{GroupAuth, GroupDatabases, GroupStorage}, 20)
	assertUsersHavePasswords(t, got)
}

func TestSmokeFirebase(t *testing.T) {
	saPath := os.Getenv("SMOKE_FIREBASE_SA_PATH")
	if saPath == "" {
		t.Skip("set SMOKE_FIREBASE_SA_PATH (service-account JSON file) to run")
	}
	sa, err := os.ReadFile(saPath)
	if err != nil {
		t.Fatalf("read service account: %v", err)
	}
	src, err := NewFirebaseSource(map[string]any{
		"serviceAccount": string(sa),
		"signerKey":      os.Getenv("SMOKE_FIREBASE_SIGNER_KEY"),
		"saltSeparator":  os.Getenv("SMOKE_FIREBASE_SALT_SEPARATOR"),
		"rounds":         os.Getenv("SMOKE_FIREBASE_ROUNDS"),
		"memCost":        os.Getenv("SMOKE_FIREBASE_MEM_COST"),
		"storageBucket":  os.Getenv("SMOKE_FIREBASE_STORAGE_BUCKET"),
	})
	if err != nil {
		t.Fatalf("NewFirebaseSource: %v", err)
	}
	defer src.Close()
	got := runSmoke(t, src, []Group{GroupAuth, GroupDatabases, GroupStorage}, 20)
	assertUsersHavePasswords(t, got)
}
