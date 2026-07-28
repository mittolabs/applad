package deploy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PreviewPath is where a site's screenshot is kept. The API serves it and the
// builds worker writes it, so it lives on the volume they share.
func PreviewPath(targetID string) string {
	base := os.Getenv("STORAGE_PATH")
	if base == "" {
		base = "/var/applad/storage"
	}
	return filepath.Join(base, "site-previews", targetID+".png")
}

// ErrSubdomainTaken is returned when a name resolves to an address another
// site already answers on. Callers turn it into a conflict rather than a
// server error, since the person naming the site can fix it.
var ErrSubdomainTaken = errors.New("subdomain already taken")

/*
 * The address a deployed app answers on.
 *
 * A subdomain is global: <sub>.applad.dev routes to whichever container claims
 * it. Deriving one from the target's name at deploy time and never storing it
 * meant two targets called the same thing resolved to one address, and the
 * later deploy quietly took the earlier one's traffic — across projects as
 * well as within them. It is stored and unique now, and claimed when a target
 * is created rather than discovered when one is deployed.
 */

// Subdomain reduces a name or domain to a DNS-safe label:
// "The Range" becomes "the-range", "the-range.applad.dev" becomes "the-range".
func Subdomain(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	if i := strings.Index(v, "."); i > 0 {
		v = v[:i] // keep only the first label of a full domain
	}

	var b strings.Builder
	lastDash := false
	for _, r := range v {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// TLSAuthorize reports whether a hostname is allowed to obtain a certificate on
// demand. The edge asks this before issuing, so a certificate is only ever minted
// for a name a deployed app actually answers on: a claimed subdomain under the
// deploy domain, or a registered custom domain. Without this gate a wildcard DNS
// record would let anyone request `anything.applad.dev` and burn through the CA's
// rate limit; with it, the set of issuable names is exactly the set of live apps.
func (s *Service) TLSAuthorize(ctx context.Context, host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	var one int
	if err := s.db.QueryRowContext(ctx,
		"SELECT 1 FROM custom_domains WHERE lower(domain) = $1", host).Scan(&one); err == nil {
		return true
	}
	suffix := "." + strings.ToLower(s.deployDomainOr())
	if !strings.HasSuffix(host, suffix) {
		return false
	}
	sub := strings.TrimSuffix(host, suffix)
	if sub == "" || strings.Contains(sub, ".") {
		return false // exactly one label under the deploy domain
	}
	if err := s.db.QueryRowContext(ctx,
		"SELECT 1 FROM deploy_targets WHERE subdomain = $1", sub).Scan(&one); err == nil {
		return true
	}
	return false
}

// ClaimSubdomain reserves a name's subdomain for a target, refusing one that
// another target already answers on.
//
// Rejecting is deliberate rather than quietly suffixing: somebody naming a
// second site "The Range" has almost certainly forgotten the first, and a
// silently different address is a worse surprise than being told.
func (s *Service) ClaimSubdomain(ctx context.Context, targetID, name string) (string, error) {
	sub := Subdomain(name)
	if sub == "" {
		return "", nil // nothing to claim; the deploy will not be routable
	}

	var owner string
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM deploy_targets WHERE subdomain = $1 AND id <> $2", sub, targetID).Scan(&owner)
	if err == nil {
		return "", fmt.Errorf("%w: %s is already used by another site — choose a different name",
			ErrSubdomainTaken, sub)
	}

	return sub, nil
}
