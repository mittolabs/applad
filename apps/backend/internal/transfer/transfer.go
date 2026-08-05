package transfer

import (
	"context"
	"errors"
	"time"
)

// errCancelled is returned internally when a job is cancelled mid-run.
var errCancelled = errors.New("transfer: cancelled")

// progressStore is the slice of the Store the orchestrator needs. Defining it as
// an interface keeps Transfer unit-testable with an in-memory fake. *Store
// satisfies it.
type progressStore interface {
	MarkRunning(ctx context.Context, id string) error
	SetCounts(ctx context.Context, id string, counts map[string]GroupCount) error
	RecordResource(ctx context.Context, migrationID string, grp Group, resourceType, sourceID, destID, status, message string) error
	Status(ctx context.Context, id string) (string, error)
	Finish(ctx context.Context, id, status, errMsg string) error
}

// Transfer orchestrates one migration: it drives a Source to export resources
// and hands each to a Destination, recording progress against the Store. It is
// created per job by the worker.
type Transfer struct {
	src    Source
	dst    Destination
	store  progressStore
	id     string
	groups []Group

	counts    map[string]GroupCount
	processed int
	lastFlush time.Time
}

func NewTransfer(store progressStore, src Source, dst Destination, migrationID string, groups []Group) *Transfer {
	return &Transfer{
		src:    src,
		dst:    dst,
		store:  store,
		id:     migrationID,
		groups: groups,
		counts: map[string]GroupCount{},
	}
}

// Run executes the migration end to end. It is safe to re-run for the same
// migration ID: the Destination imports idempotently, so a resumed job converges
// rather than duplicating. A user cancellation ends the job as "cancelled"; any
// other export/import error ends it as "failed"; context cancellation (worker
// shutdown) leaves it "running" so it can resume.
func (t *Transfer) Run(ctx context.Context) error {
	if err := t.store.MarkRunning(ctx, t.id); err != nil {
		return err
	}

	// Pre-flight: record per-group totals for the progress UI.
	if totals, err := t.src.Report(ctx, t.groups); err == nil {
		for g, n := range totals {
			c := t.counts[string(g)]
			c.Total = n
			t.counts[string(g)] = c
		}
		_ = t.store.SetCounts(ctx, t.id, t.counts)
	}

	emit := func(ctx context.Context, res []Resource) error {
		if cancelled, _ := t.isCancelled(ctx); cancelled {
			return errCancelled
		}
		for _, r := range res {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			result := t.dst.Import(ctx, r)
			t.record(ctx, r, result)
		}
		t.maybeFlush(ctx, false)
		return nil
	}

	exportErr := t.src.Export(ctx, t.groups, emit)
	t.maybeFlush(ctx, true) // always flush final counts

	switch {
	case errors.Is(exportErr, errCancelled):
		return t.store.Finish(ctx, t.id, "cancelled", "")
	case ctx.Err() != nil:
		// Worker shutdown: leave the job running so a later worker resumes it.
		return ctx.Err()
	case exportErr != nil:
		return t.store.Finish(ctx, t.id, "failed", exportErr.Error())
	default:
		return t.store.Finish(ctx, t.id, "completed", "")
	}
}

// record updates the in-memory tally and persists a per-resource row. To keep
// data_migration_resources bounded on large imports, successful bulk rows are
// tallied but not individually persisted; every failure/warning/skip and every
// non-row resource is persisted for inspection and idempotent resume.
func (t *Transfer) record(ctx context.Context, r Resource, res Result) {
	g := string(r.Group())
	c := t.counts[g]
	switch res.Status {
	case StatusDone:
		c.Done++
	case StatusWarning:
		c.Warning++
	case StatusError:
		c.Error++
	case StatusSkip:
		c.Skip++
	}
	t.counts[g] = c
	t.processed++

	if r.Kind() == "row" && res.Status == StatusDone {
		return
	}
	_ = t.store.RecordResource(ctx, t.id, r.Group(), r.Kind(), r.SourceID(), res.DestID, res.Status, res.Message)
}

// maybeFlush persists counts periodically (and always when force is set) so the
// console poll sees live progress without a DB write per resource.
func (t *Transfer) maybeFlush(ctx context.Context, force bool) {
	if force || time.Since(t.lastFlush) > 2*time.Second {
		_ = t.store.SetCounts(ctx, t.id, t.counts)
		t.lastFlush = time.Now()
	}
}

func (t *Transfer) isCancelled(ctx context.Context) (bool, error) {
	st, err := t.store.Status(ctx, t.id)
	if err != nil {
		return false, err
	}
	return st == "cancelled", nil
}
