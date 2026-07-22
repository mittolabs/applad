package workflows

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// The http_request node dials through netguard, which refuses loopback —
	// exactly where httptest servers live. These tests exercise the node, not
	// the egress policy (netguard has its own tests), so relax it here.
	os.Setenv("ALLOW_PRIVATE_EGRESS", "true") //nolint:errcheck
	os.Exit(m.Run())
}
