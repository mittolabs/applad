package transfer

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// fakeStore is an in-memory progressStore for orchestrator tests.
type fakeStore struct {
	mu        sync.Mutex
	status    string
	counts    map[string]GroupCount
	resources []string // "grp/type/sourceID=status"
	finished  string
}

func newFakeStore() *fakeStore { return &fakeStore{status: "pending", counts: map[string]GroupCount{}} }

func (f *fakeStore) MarkRunning(_ context.Context, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = "running"
	return nil
}
func (f *fakeStore) SetCounts(_ context.Context, _ string, counts map[string]GroupCount) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counts = map[string]GroupCount{}
	for k, v := range counts {
		f.counts[k] = v
	}
	return nil
}
func (f *fakeStore) RecordResource(_ context.Context, _ string, grp Group, rt, sourceID, _, status, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resources = append(f.resources, string(grp)+"/"+rt+"/"+sourceID+"="+status)
	return nil
}
func (f *fakeStore) Status(_ context.Context, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status, nil
}
func (f *fakeStore) Finish(_ context.Context, _ string, status, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.finished = status
	f.status = status
	return nil
}
func (f *fakeStore) cancel() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = "cancelled"
}

// fakeSource emits a fixed set of resources in dependency order.
type fakeSource struct {
	resources []Resource
	reportErr error
}

func (s *fakeSource) Name() string { return "fake" }
func (s *fakeSource) Close() error { return nil }
func (s *fakeSource) Report(_ context.Context, groups []Group) (map[Group]int, error) {
	if s.reportErr != nil {
		return nil, s.reportErr
	}
	m := map[Group]int{}
	for _, r := range s.resources {
		m[r.Group()]++
	}
	return m, nil
}
func (s *fakeSource) Export(ctx context.Context, _ []Group, emit Emit) error {
	for _, r := range s.resources {
		if err := emit(ctx, []Resource{r}); err != nil {
			return err
		}
	}
	return nil
}

// fakeDest records imports and can fail selected kinds or cancel mid-run.
type fakeDest struct {
	imported    []string
	failKind    string
	cancelAfter int // when >0, cancel the store after this many imports
	store       *fakeStore
}

func (d *fakeDest) Name() string { return "fake" }
func (d *fakeDest) Close() error { return nil }
func (d *fakeDest) Import(_ context.Context, r Resource) Result {
	d.imported = append(d.imported, r.Kind()+":"+r.SourceID())
	if d.cancelAfter > 0 && len(d.imported) >= d.cancelAfter && d.store != nil {
		d.store.cancel()
	}
	if r.Kind() == d.failKind {
		return Result{Status: StatusError, Message: "boom"}
	}
	return Result{DestID: "dest-" + r.SourceID(), Status: StatusDone}
}

func sampleResources() []Resource {
	return []Resource{
		User{ID: "u1", Email: "a@x.io", PasswordHash: "h", PasswordAlgo: "bcrypt"},
		Database{ID: "db1", Name: "Main"},
		Table{DatabaseID: "db1", ID: "t1", Name: "Posts"},
		Column{DatabaseID: "db1", TableID: "t1", Key: "title", Type: "string"},
		Row{DatabaseID: "db1", TableID: "t1", ID: "r1", Data: map[string]any{"title": "hi"}},
		Row{DatabaseID: "db1", TableID: "t1", ID: "r2", Data: map[string]any{"title": "yo"}},
	}
}

func TestTransferHappyPath(t *testing.T) {
	store := newFakeStore()
	src := &fakeSource{resources: sampleResources()}
	dst := &fakeDest{}

	err := NewTransfer(store, src, dst, "m1", []Group{GroupAuth, GroupDatabases}).Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if store.finished != "completed" {
		t.Fatalf("expected completed, got %q", store.finished)
	}
	if len(dst.imported) != 6 {
		t.Fatalf("expected 6 imports, got %d (%v)", len(dst.imported), dst.imported)
	}
	// Parent-before-child order: database imported before its table before its rows.
	if dst.imported[1] != "database:db1" || dst.imported[2] != "table:db1/t1" {
		t.Fatalf("dependency order wrong: %v", dst.imported)
	}
	// Counts: auth 1 done, databases 4 done (db, table, column, 2 rows = 5). Verify rows tallied.
	if store.counts["auth"].Done != 1 {
		t.Fatalf("auth done = %d, want 1", store.counts["auth"].Done)
	}
	if store.counts["databases"].Done != 5 {
		t.Fatalf("databases done = %d, want 5", store.counts["databases"].Done)
	}
	// Successful rows are tallied but not individually persisted as resource rows.
	for _, r := range store.resources {
		if r == "databases/row/t1/r1=done" {
			t.Fatal("successful row should not be persisted as a resource row")
		}
	}
}

func TestTransferRecordsErrors(t *testing.T) {
	store := newFakeStore()
	src := &fakeSource{resources: sampleResources()}
	dst := &fakeDest{failKind: "row"}

	if err := NewTransfer(store, src, dst, "m1", nil).Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if store.counts["databases"].Error != 2 {
		t.Fatalf("expected 2 row errors, got %d", store.counts["databases"].Error)
	}
	// Failed rows ARE persisted for inspection.
	var persistedRowErr bool
	for _, r := range store.resources {
		if r == "databases/row/t1/r1=error" {
			persistedRowErr = true
		}
	}
	if !persistedRowErr {
		t.Fatal("failed row should be persisted as a resource row")
	}
}

func TestTransferCancellation(t *testing.T) {
	store := newFakeStore()
	src := &fakeSource{resources: sampleResources()}
	dst := &fakeDest{cancelAfter: 1, store: store} // cancel after the first import

	if err := NewTransfer(store, src, dst, "m1", nil).Run(context.Background()); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if store.finished != "cancelled" {
		t.Fatalf("expected cancelled, got %q", store.finished)
	}
	// The first resource imports; cancellation is observed before the next emit,
	// so not all six are imported.
	if len(dst.imported) >= len(sampleResources()) {
		t.Fatalf("expected cancellation to stop the run early, imported %d", len(dst.imported))
	}
}

func TestTransferExportError(t *testing.T) {
	store := newFakeStore()
	src := &fakeSource{resources: sampleResources(), reportErr: nil}
	dst := &fakeDest{}
	// Force an export error by wrapping the source.
	failing := &failingSource{fakeSource: src, at: 3}

	err := NewTransfer(store, failing, dst, "m1", nil).Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if store.finished != "failed" {
		t.Fatalf("expected failed, got %q", store.finished)
	}
}

type failingSource struct {
	*fakeSource
	at int
}

func (s *failingSource) Export(ctx context.Context, _ []Group, emit Emit) error {
	for i, r := range s.resources {
		if i == s.at {
			return errors.New("source blew up")
		}
		if err := emit(ctx, []Resource{r}); err != nil {
			return err
		}
	}
	return nil
}

func TestSanitizeID(t *testing.T) {
	cases := map[string]string{
		"abc":       "abc",
		"a-b-c":     "a_b_c",
		"a.b/c":     "a_b_c",
		"  spaced ": "spaced",
		"":          "",
		"---":       "",
	}
	for in, want := range cases {
		if got := sanitizeID(in); got != want {
			t.Errorf("sanitizeID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSafeID(t *testing.T) {
	// Clean, short IDs pass through unchanged and are stable.
	if got := safeID("u", "abc123"); got != "abc123" {
		t.Fatalf("clean id changed: %q", got)
	}
	// Stability: same input -> same output.
	if safeID("u", "a-b.c") != safeID("u", "a-b.c") {
		t.Fatal("safeID not stable for same input")
	}
	// Collision resistance: two distinct source IDs that sanitize to the same
	// base must NOT collapse to the same dest ID.
	if safeID("u", "a-b") == safeID("u", "a_b") {
		t.Fatal("distinct source IDs collided to same dest ID")
	}
	// Empty input still yields a non-empty id.
	if safeID("row", "") == "" {
		t.Fatal("safeID returned empty for empty input")
	}
}

func TestMapColumnType(t *testing.T) {
	cases := map[string]string{
		"varchar": "string", "int8": "integer", "bool": "boolean",
		"timestamptz": "datetime", "jsonb": "string", "weirdtype": "string",
	}
	for in, want := range cases {
		if got := mapColumnType(in); got != want {
			t.Errorf("mapColumnType(%q) = %q, want %q", in, got, want)
		}
	}
}
