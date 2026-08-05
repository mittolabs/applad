package transfer

import "context"

// Source reads a platform and emits normalized resources. Implementations:
// source_applad.go, and (later) source_appwrite/supabase/nhost/firebase.
type Source interface {
	// Name identifies the source type (applad, appwrite, supabase, ...).
	Name() string
	// Report validates credentials/connectivity and returns the count of
	// available resources per requested group, for the pre-flight summary.
	Report(ctx context.Context, groups []Group) (map[Group]int, error)
	// Export pulls the requested groups and hands resources to emit in
	// dependency order. It returns when everything has been emitted or ctx is
	// cancelled. Export must not import anything itself.
	Export(ctx context.Context, groups []Group, emit Emit) error
	// Close releases any connections held by the source.
	Close() error
}

// Destination writes resources into an Applad project. The only implementation
// is dest_applad.go; every migration targets an Applad instance.
type Destination interface {
	Name() string
	// Import writes one resource and reports the outcome. It must be idempotent
	// so a resumed migration does not duplicate data.
	Import(ctx context.Context, res Resource) Result
	Close() error
}
