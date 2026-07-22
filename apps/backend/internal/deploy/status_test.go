package deploy

import "testing"

// A site with one failed build was labelled Active because the console had no
// status to read and defaulted to the happy one. These are the states that
// distinction has to survive.
func TestTargetStatus(t *testing.T) {
	cases := []struct {
		name         string
		lastRelease  string
		everDeployed bool
		want         string
	}{
		{"nothing has ever run", "", false, "never_deployed"},
		{"first build failed", "failed", false, "failed"},
		{"first build running", "building", false, "building"},
		{"first build queued", "pending", false, "building"},
		{"deployed and serving", "success", true, "active"},
		// A failed build does not take down what is already serving.
		{"live site, newest build failed", "failed", true, "active"},
		// Nor does a build in progress.
		{"live site, rebuilding", "building", true, "deploying"},
		{"cancelled with nothing live", "cancelled", false, "failed"},
	}
	for _, c := range cases {
		if got := targetStatus(c.lastRelease, c.everDeployed); got != c.want {
			t.Errorf("%s: targetStatus(%q, %v) = %q, want %q",
				c.name, c.lastRelease, c.everDeployed, got, c.want)
		}
	}
}
