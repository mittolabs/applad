package cronx

import (
	"testing"
	"time"
)

// The expressions the previous matcher silently ignored. Each of these is
// valid standard cron, and each used to evaluate to false forever — so a job
// scheduled with one simply never ran, with no error anywhere.
func TestParsesExpressionsTheOldMatcherDropped(t *testing.T) {
	// Wednesday 2026-07-22 09:30 UTC.
	now := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)

	tests := []struct {
		expr     string
		intent   string
		firesNow bool
	}{
		{"30 9 * * *", "every day at 09:30", true},
		{"30 9 * * 1-5", "weekdays at 09:30", true},
		{"30 9 * * MON-FRI", "weekdays, named", true},
		{"0,30 9 * * *", "twice an hour", true},
		{"30 9-17 * * *", "hourly through business hours", true},
		// The 22nd is outside days 1-7, so this correctly does not fire today;
		// it is checked against a date inside the range below.
		{"30 9 1-7 * *", "first week of the month", false},
		{"*/15 * * * *", "every 15 minutes", true},
		{"@daily", "descriptor", false}, // fires at midnight, not 09:30
		{"30 9 * * 6", "Saturdays only", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			s, err := Parse(tt.expr)
			if err != nil {
				t.Fatalf("%s (%s): %v", tt.expr, tt.intent, err)
			}
			// A schedule fires at `now` when the next run after the previous
			// minute is exactly now.
			fires := s.Next(now.Add(-time.Minute)).Equal(now)
			if fires != tt.firesNow {
				t.Errorf("%s (%s): fires at %s = %v, want %v",
					tt.expr, tt.intent, now.Format("Mon 15:04"), fires, tt.firesNow)
			}
		})
	}
}

// A day-of-month range fires on the days inside it and no others.
func TestDayOfMonthRange(t *testing.T) {
	s, err := Parse("30 9 1-7 * *")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		day   int
		fires bool
	}{{1, true}, {3, true}, {7, true}, {8, false}, {22, false}} {
		at := time.Date(2026, 7, tc.day, 9, 30, 0, 0, time.UTC)
		if got := s.Next(at.Add(-time.Minute)).Equal(at); got != tc.fires {
			t.Errorf("day %d fires = %v, want %v", tc.day, got, tc.fires)
		}
	}
}

// Day-of-month is 1-based, so */2 means the 1st, 3rd, 5th. The old matcher
// used value%step, which fired on the 2nd, 4th, 6th instead.
func TestDayOfMonthSteppingIsOneBased(t *testing.T) {
	s, err := Parse("0 0 */2 * *")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		day   int
		fires bool
	}{{1, true}, {2, false}, {3, true}, {4, false}} {
		at := time.Date(2026, 7, tc.day, 0, 0, 0, 0, time.UTC)
		got := s.Next(at.Add(-time.Minute)).Equal(at)
		if got != tc.fires {
			t.Errorf("day %d fires = %v, want %v", tc.day, got, tc.fires)
		}
	}
}

// "0 9 * * *" set by someone in Nairobi should mean 09:00 there, not 09:00 UTC.
func TestTimezoneIsHonoured(t *testing.T) {
	s, err := Parse("CRON_TZ=Africa/Nairobi 0 9 * * *")
	if err != nil {
		t.Fatal(err)
	}
	// Nairobi is UTC+3, so 09:00 local is 06:00 UTC.
	base := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	next := s.Next(base).UTC()
	if next.Hour() != 6 || next.Minute() != 0 {
		t.Errorf("next run = %s UTC, want 06:00 UTC (09:00 in Nairobi)", next.Format("15:04"))
	}
}

func TestValidateRejectsNonsenseAndAllowsEmpty(t *testing.T) {
	if err := Validate(""); err != nil {
		t.Errorf("empty schedule means 'not scheduled' and must validate: %v", err)
	}
	for _, bad := range []string{"not a cron", "0 9 * *", "99 * * * *", "0 9 * * FUNDAY"} {
		if err := Validate(bad); err == nil {
			t.Errorf("Validate(%q) accepted an unusable expression", bad)
		}
	}
}

func TestDescribeNamesTheTimezone(t *testing.T) {
	tests := map[string]string{
		"":                                 "Not scheduled",
		"0 9 * * *":                        "0 9 * * * (UTC)",
		"CRON_TZ=Africa/Nairobi 0 9 * * *": "0 9 * * * (Africa/Nairobi)",
	}
	for in, want := range tests {
		if got := Describe(in); got != want {
			t.Errorf("Describe(%q) = %q, want %q", in, got, want)
		}
	}
}
