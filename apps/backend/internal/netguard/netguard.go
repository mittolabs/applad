// Package netguard provides an http.Client for fetching URLs an attacker may
// control (workflow http_request nodes, webhook deliveries). Its dialer
// refuses any destination that is not publicly routable, which is what stands
// between a user-supplied URL and cloud metadata (169.254.169.254) or the
// compose-internal services (postgres, redis, the API itself).
package netguard

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"
)

const maxRedirects = 5

// allowPrivate reports the egress policy. Self-hosters whose workflows
// genuinely need to call internal services opt out with
// ALLOW_PRIVATE_EGRESS=true; the default is closed. It is read per-call (an env
// lookup is negligible next to a network dial) so an operator toggling the
// policy — and tests exercising both modes — see the current value.
func allowPrivate() bool {
	return os.Getenv("ALLOW_PRIVATE_EGRESS") == "true"
}

// Client returns an http.Client whose dials are checked AFTER DNS resolution,
// so a hostname that resolves to a private address is refused at connect time
// — which also covers DNS rebinding. Redirects are capped and every hop dials
// through the same guard.
func Client(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: transport(),
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("netguard: stopped after %d redirects", maxRedirects)
			}
			return nil
		},
	}
}

// transport is shared so guarded clients pool connections.
var transport = sync.OnceValue(func() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			return CheckAddr(address)
		},
	}).DialContext
	return t
})

// CheckAddr rejects a resolved host:port whose IP is loopback, link-local
// (169.254/16 — cloud metadata — and fe80::/10), private (RFC1918, fc00::/7),
// CGNAT (100.64/10), multicast or unspecified. It runs on the address being
// dialled, not the URL, so no name or redirect can smuggle one past it.
func CheckAddr(address string) error {
	if allowPrivate() {
		return nil
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("netguard: %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("netguard: %q is not an IP address", host)
	}
	if !isPublic(ip) {
		return fmt.Errorf("netguard: %s is not a public address (set ALLOW_PRIVATE_EGRESS=true to permit internal calls)", ip)
	}
	return nil
}

func isPublic(ip net.IP) bool {
	// IsPrivate covers RFC1918 (10/8, 172.16/12, 192.168/16) and IPv6 ULA
	// (fc00::/7); IsLinkLocalUnicast covers 169.254/16 and fe80::/10.
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1]&0xc0 == 64 { // CGNAT 100.64/10
		return false
	}
	return true
}
