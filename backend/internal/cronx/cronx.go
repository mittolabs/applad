// Package cronx parses the cron expressions used to schedule workflows,
// functions and deploy targets.
//
// It exists so that every place accepting a schedule validates it the same
// way. The previous hand-rolled matcher understood only "*", "*/n" and bare
// integers: ranges, lists and names silently evaluated to false, so a job
// scheduled for "0 9 * * 1-5" never ran and never reported why.
package cronx

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// parser accepts standard 5-field expressions plus descriptors like @daily.
// Seconds are deliberately not supported: the scheduler ticks once a minute,
// so a seconds field would promise precision it cannot keep.
var parser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// Schedule computes successive run times for an expression.
type Schedule = cron.Schedule

// Parse validates a cron expression and returns its schedule.
//
// A timezone may be given inline, as "CRON_TZ=Africa/Nairobi 0 9 * * *".
// Without one the expression is interpreted in UTC, which is rarely what
// somebody typing "0 9 * * *" means.
func Parse(expr string) (Schedule, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("schedule is empty")
	}
	s, err := parser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule %q: %w", expr, err)
	}
	return s, nil
}

// Validate reports whether an expression is usable, for API input checking.
// An empty expression is valid and means "not scheduled".
func Validate(expr string) error {
	if strings.TrimSpace(expr) == "" {
		return nil
	}
	_, err := Parse(expr)
	return err
}

// Next returns the first run time strictly after `after`.
func Next(expr string, after time.Time) (time.Time, error) {
	s, err := Parse(expr)
	if err != nil {
		return time.Time{}, err
	}
	return s.Next(after), nil
}

// Describe renders an expression for display, naming the timezone it runs in.
func Describe(expr string) string {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "Not scheduled"
	}
	if tz, rest, ok := strings.Cut(expr, " "); ok && strings.HasPrefix(tz, "CRON_TZ=") {
		return fmt.Sprintf("%s (%s)", rest, strings.TrimPrefix(tz, "CRON_TZ="))
	}
	return expr + " (UTC)"
}
