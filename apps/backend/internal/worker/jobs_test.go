package worker

import (
	"errors"
	"testing"
)

func TestDecideDispatch(t *testing.T) {
	transportErr := errors.New("connection refused")

	cases := []struct {
		name        string
		status      int
		err         error
		attempts    int
		maxAttempts int
		want        dispatchAction
	}{
		// A 2xx acks regardless of how many attempts remain.
		{"200 acks", 200, nil, 1, 3, dispatchAck},
		{"201 acks", 201, nil, 1, 3, dispatchAck},
		{"299 acks", 299, nil, 3, 3, dispatchAck},

		// Non-2xx responses retry while attempts remain, then fail.
		{"500 with attempts left retries", 500, nil, 1, 3, dispatchRetry},
		{"500 second try retries", 500, nil, 2, 3, dispatchRetry},
		{"500 last attempt fails", 500, nil, 3, 3, dispatchFail},
		{"404 retries", 404, nil, 1, 3, dispatchRetry},
		{"3xx is not success", 301, nil, 1, 3, dispatchRetry},

		// A transport error (netguard block, timeout, refused) is a failure too.
		{"transport error retries", 0, transportErr, 1, 3, dispatchRetry},
		{"transport error on last attempt fails", 0, transportErr, 2, 2, dispatchFail},

		// A single-attempt queue fails on the first non-2xx.
		{"single attempt fails immediately", 503, nil, 1, 1, dispatchFail},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decideDispatch(c.status, c.err, c.attempts, c.maxAttempts)
			if got != c.want {
				t.Errorf("decideDispatch(%d, %v, attempts=%d, max=%d) = %d, want %d",
					c.status, c.err, c.attempts, c.maxAttempts, got, c.want)
			}
		})
	}
}

func TestDispatchError(t *testing.T) {
	if msg := dispatchError(0, errors.New("boom")); msg != "boom" {
		t.Errorf("transport error message = %q, want %q", msg, "boom")
	}
	if msg := dispatchError(500, nil); msg != "worker returned status 500" {
		t.Errorf("status message = %q, want %q", msg, "worker returned status 500")
	}
}
