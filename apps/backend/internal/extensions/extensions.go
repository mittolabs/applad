// Package extensions is the mount point for modules compiled into a build.
//
// It is a FIRST-PARTY, COMPILE-TIME seam. There is no dynamic loading, no
// sandboxing and no stable public plugin API: a module is a Go package included
// in the build, which registers itself from init(). A default build registers
// nothing and behaves exactly as if this package did not exist.
package extensions

import (
	"context"
	"database/sql"
	"sync"

	"github.com/go-chi/chi/v5"
)

// Migration is a schema change owned by a module. Versions are recorded with an
// "ee:" prefix so module numbering can never collide with core's.
type Migration struct {
	Version string
	SQL     string
}

// Deps is what core hands a module when mounting it. Modules receive their
// dependencies rather than reaching for globals, so registration order and
// startup sequencing stay core's business.
type Deps struct {
	DB *sql.DB
}

// Module is one unit registered into a build.
type Module struct {
	// Name identifies the module in logs.
	Name string
	// Setup runs once at startup with core's dependencies, before routes are
	// mounted. This is where a module registers the providers core asks
	// questions of: entitlements, policy. A failure is logged, not fatal, for
	// the same reason those default to permissive.
	Setup func(d Deps) error
	// Routes mounts the module's HTTP surface. Mounted under /v1 alongside core.
	Routes func(r chi.Router, d Deps)
	// Migrations run after core's, in order.
	Migrations []Migration
}

var (
	mu      sync.RWMutex
	modules []Module
)

// Add registers a module. Called from a module's init(), so merely including
// the package in a build is what activates it.
func Add(m Module) {
	mu.Lock()
	defer mu.Unlock()
	modules = append(modules, m)
}

// All returns the registered modules.
func All() []Module {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Module, len(modules))
	copy(out, modules)
	return out
}

// Migrations returns every module migration, in registration order.
func Migrations() []Migration {
	var out []Migration
	for _, m := range All() {
		out = append(out, m.Migrations...)
	}
	return out
}

// UsageReporter is how a module reads consumption WITHOUT touching core's
// tables. Modules are compiled from another repository and are not present in
// core's CI, so letting them query core's schema directly would make every
// schema change a silent break. Core owns the queries; modules get this.
type UsageReporter interface {
	CountProjects(ctx context.Context, orgID string) (int64, error)
	CountMembers(ctx context.Context, orgID string) (int64, error)
	StorageBytes(ctx context.Context, projectID string) (int64, error)
}

var (
	usageMu sync.RWMutex
	usage   UsageReporter
)

// SetUsageReporter installs core's implementation at startup.
func SetUsageReporter(r UsageReporter) {
	usageMu.Lock()
	defer usageMu.Unlock()
	usage = r
}

// Usage returns the reporter, or nil when core has not registered one.
func Usage() UsageReporter {
	usageMu.RLock()
	defer usageMu.RUnlock()
	return usage
}
