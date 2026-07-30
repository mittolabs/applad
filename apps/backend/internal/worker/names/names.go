// Package names is the single source of truth for the set of Applad background
// worker names. It has no dependencies so both the worker processes (which
// publish a status:worker:<name> heartbeat under these names) and the API's
// status service (which checks that every one of them is alive) can import it
// without dragging the worker package's heavy runtime dependencies into the API
// binary.
package names

// All is the canonical, ordered list of the 12 background workers that a
// complete Applad install runs. Each worker publishes a Redis heartbeat keyed
// by its name; the status service treats the set below as the expected roster.
var All = []string{
	"builds",
	"certificates",
	"cron",
	"databases",
	"deletes",
	"executions",
	"jobs",
	"mails",
	"messaging",
	"migrations",
	"usage",
	"webhooks",
}
