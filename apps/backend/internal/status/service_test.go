package status

import (
	"testing"
	"time"

	workernames "github.com/mittolabs/applad/internal/worker/names"
)

// beatsFor returns a fresh heartbeat value (now) for every expected worker.
func beatsFor(now time.Time) []string {
	beats := make([]string, len(workernames.All))
	for i := range beats {
		beats[i] = now.Format(time.RFC3339)
	}
	return beats
}

func TestEvaluateWorkerHeartbeats_AllPresentOperational(t *testing.T) {
	now := time.Now().UTC()
	got := evaluateWorkerHeartbeats(workernames.All, beatsFor(now), now)
	if got.status != statusOperational {
		t.Fatalf("all workers fresh: want %q, got %q (%s)", statusOperational, got.status, got.errMsg)
	}
}

func TestEvaluateWorkerHeartbeats_OneMissingDegraded(t *testing.T) {
	now := time.Now().UTC()
	beats := beatsFor(now)
	beats[len(beats)-1] = "" // last worker's key is gone

	got := evaluateWorkerHeartbeats(workernames.All, beats, now)
	if got.status != statusDegraded {
		t.Fatalf("one worker missing: want %q, got %q", statusDegraded, got.status)
	}
	if want := workernames.All[len(workernames.All)-1]; got.errMsg == "" || !containsName(got.errMsg, want) {
		t.Fatalf("degraded errMsg should name the down worker %q, got %q", want, got.errMsg)
	}
}

func TestEvaluateWorkerHeartbeats_OneStaleDegraded(t *testing.T) {
	now := time.Now().UTC()
	beats := beatsFor(now)
	// A value present but older than the TTL is stale, not fresh.
	beats[0] = now.Add(-workerHeartbeatTTL - time.Minute).Format(time.RFC3339)

	got := evaluateWorkerHeartbeats(workernames.All, beats, now)
	if got.status != statusDegraded {
		t.Fatalf("one worker stale: want %q, got %q", statusDegraded, got.status)
	}
	if !containsName(got.errMsg, workernames.All[0]) {
		t.Fatalf("stale errMsg should name worker %q, got %q", workernames.All[0], got.errMsg)
	}
}

func TestEvaluateWorkerHeartbeats_NonePresentDown(t *testing.T) {
	now := time.Now().UTC()
	beats := make([]string, len(workernames.All)) // all empty

	got := evaluateWorkerHeartbeats(workernames.All, beats, now)
	if got.status != statusDown {
		t.Fatalf("no workers reporting: want %q, got %q", statusDown, got.status)
	}
}

func TestHeartbeatFresh(t *testing.T) {
	now := time.Now().UTC()
	cases := []struct {
		name string
		beat string
		want bool
	}{
		{"missing", "", false},
		{"fresh", now.Format(time.RFC3339), true},
		{"withinTTL", now.Add(-workerHeartbeatTTL + time.Second).Format(time.RFC3339), true},
		{"stale", now.Add(-workerHeartbeatTTL - time.Second).Format(time.RFC3339), false},
		{"unparseablePresent", "not-a-timestamp", true},
	}
	for _, c := range cases {
		if got := heartbeatFresh(c.beat, now); got != c.want {
			t.Errorf("%s: heartbeatFresh(%q) = %v, want %v", c.name, c.beat, got, c.want)
		}
	}
}

// containsName reports whether msg mentions the given worker name.
func containsName(msg, name string) bool {
	for i := 0; i+len(name) <= len(msg); i++ {
		if msg[i:i+len(name)] == name {
			return true
		}
	}
	return false
}
