package runtime

import (
	"strings"
	"testing"
)

// hostConfigOf pulls the HostConfig sub-map out of a Docker create body.
func hostConfigOf(t *testing.T, body map[string]interface{}) map[string]interface{} {
	t.Helper()
	hc, ok := body["HostConfig"].(map[string]interface{})
	if !ok {
		t.Fatalf("body has no HostConfig map: %#v", body["HostConfig"])
	}
	return hc
}

// Deployed apps run arbitrary customer code, so their container must carry the
// same baseline hardening a function gets: all capabilities dropped, no
// privilege escalation, and a process cap so a fork bomb cannot exhaust the
// host. This is the regression guard for the missing hardening.
func TestDeployContainerBodyIsHardened(t *testing.T) {
	body := deployContainerBody(
		"applad-site-demo", "img:1", "3000",
		[]string{"PORT=3000"},
		map[string]string{"applad.managed": "true"},
		"applad_default",
	)
	hc := hostConfigOf(t, body)

	capDrop, ok := hc["CapDrop"].([]string)
	if !ok || len(capDrop) != 1 || capDrop[0] != "ALL" {
		t.Errorf("CapDrop = %#v, want [ALL]", hc["CapDrop"])
	}

	secOpt, ok := hc["SecurityOpt"].([]string)
	if !ok || !contains(secOpt, "no-new-privileges") {
		t.Errorf("SecurityOpt = %#v, want to contain no-new-privileges", hc["SecurityOpt"])
	}

	pids, ok := hc["PidsLimit"].(int64)
	if !ok || pids <= 0 {
		t.Errorf("PidsLimit = %#v, want a positive cap", hc["PidsLimit"])
	}
}

// A deployed web app is routed by container name on the deploy network through
// the edge proxy, so it must join that network. It must NOT publish its port on
// the host (that exposed it without the proxy in front).
func TestDeployContainerBodyStaysRoutable(t *testing.T) {
	body := deployContainerBody(
		"applad-site-demo", "img:1", "8080", nil, nil, "applad_default",
	)
	hc := hostConfigOf(t, body)

	if _, present := hc["PublishAllPorts"]; present {
		t.Errorf("PublishAllPorts should not be set (routing is by network name): %#v", hc["PublishAllPorts"])
	}
	if hc["NetworkMode"] != "applad_default" {
		t.Errorf("NetworkMode = %#v, want applad_default", hc["NetworkMode"])
	}
	// The app is never given a writable-blocking read-only rootfs: web apps
	// write temp/cache, and the deploy container has no writable tmpfs to
	// compensate the way a function does.
	if _, present := hc["ReadonlyRootfs"]; present {
		t.Errorf("ReadonlyRootfs should not be set for a deployed app: %#v", hc["ReadonlyRootfs"])
	}
	exposed, ok := body["ExposedPorts"].(map[string]interface{})
	if !ok {
		t.Fatalf("ExposedPorts missing: %#v", body["ExposedPorts"])
	}
	if _, ok := exposed["8080/tcp"]; !ok {
		t.Errorf("ExposedPorts = %#v, want 8080/tcp", exposed)
	}
}

// The browser sits on the shared deploy network, reachable by every
// deployed container. The forwarder in front of DevTools now demands a
// per-session token, and that token is delivered to the browser as an env var.
// Without it in the body, the running browser would accept an empty token and
// the gate would be off.
func TestBrowserBodyCarriesToken(t *testing.T) {
	const token = "deadbeefcafe"
	body := browserBody("applad-browser:1", token)

	env, ok := body["Env"].([]string)
	if !ok || !contains(env, "APPLAD_DEVTOOLS_TOKEN="+token) {
		t.Errorf("Env = %#v, want APPLAD_DEVTOOLS_TOKEN=%s", body["Env"], token)
	}

	hc := hostConfigOf(t, body)
	secOpt, ok := hc["SecurityOpt"].([]string)
	if !ok || !contains(secOpt, "no-new-privileges") {
		t.Errorf("SecurityOpt = %#v, want no-new-privileges", hc["SecurityOpt"])
	}
}

// The token must reach the CDP client, which authenticates simply by dialing
// the returned URL, so the token has to ride in the WebSocket URL's query.
func TestWithDevToolsTokenEmbedsToken(t *testing.T) {
	got := withDevToolsToken("ws://applad-browser-x:9222/devtools/page/AB12", "tok")
	if !strings.Contains(got, devToolsTokenParam+"=tok") {
		t.Errorf("withDevToolsToken did not embed token: %q", got)
	}

	// A URL that already has a query keeps it and appends with &.
	got = withDevToolsToken("ws://h:9222/p?x=1", "tok")
	if !strings.Contains(got, "x=1") || !strings.Contains(got, "&"+devToolsTokenParam+"=tok") {
		t.Errorf("withDevToolsToken clobbered existing query: %q", got)
	}

	// No token, no change.
	if got := withDevToolsToken("ws://h/p", ""); got != "ws://h/p" {
		t.Errorf("empty token changed URL: %q", got)
	}
}

// Two sessions must not share a secret.
func TestNewDevToolsTokenIsUnique(t *testing.T) {
	a, b := newDevToolsToken(), newDevToolsToken()
	if a == "" || b == "" {
		t.Fatal("token generation returned empty")
	}
	if a == b {
		t.Errorf("tokens are not unique: %q", a)
	}
	if len(a) < 32 {
		t.Errorf("token too short to resist guessing: %q", a)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
