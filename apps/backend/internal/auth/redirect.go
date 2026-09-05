package auth

import (
	"net/url"
	"path"
	"strings"
)

/*
 * Where an OAuth sign-in is allowed to send somebody.
 *
 * This runs before the user is authenticated, on a URL an attacker supplies, so
 * it is one of the more attractive open-redirect targets in the system. Two
 * things are allowed and nothing else:
 *
 *   1. A relative path. It can only land on this API's own origin.
 *   2. An absolute URL that matches one the project registered against a
 *      platform — which is what lets a native app be handed its session on
 *      funnier://auth or an app link.
 *
 * "Matches" is deliberately narrow. Prefix-matching the whole string is how
 * these go wrong: funnier://auth would admit funnier://auth.evil.com, and
 * https://app.example/cb would admit https://app.example/cbx.
 */

// safeRedirectTarget resolves a caller-supplied redirect against the URIs a
// project registered. It returns a URL that is safe to send a browser to, or
// "/" when the candidate is neither relative nor registered.
func safeRedirectTarget(raw string, registered []string) string {
	if raw == "" {
		return "/"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "/"
	}
	// A relative path is same-origin by construction. "//evil.com" parses with
	// an empty scheme but a non-empty host, so the host check catches it.
	if u.Scheme == "" && u.Host == "" && u.Opaque == "" {
		return raw
	}
	for _, candidate := range registered {
		if redirectMatches(u, candidate) {
			return raw
		}
	}
	return "/"
}

// redirectMatches reports whether target is covered by the registered URI.
func redirectMatches(target *url.URL, registered string) bool {
	allowed, err := url.Parse(strings.TrimSpace(registered))
	if err != nil || allowed.Scheme == "" {
		return false
	}
	if !strings.EqualFold(target.Scheme, allowed.Scheme) {
		return false
	}
	// Host comparison includes the port: a different port is a different
	// listener, and on localhost frequently a different application.
	if !strings.EqualFold(target.Host, allowed.Host) {
		return false
	}
	// Credentials in a redirect are never something a registration asked for,
	// and are a classic way to make a URL read as one host while resolving to
	// another.
	if target.User != nil {
		return false
	}
	// funnier://auth parses with an empty path; anything under it is fine.
	if allowed.Path == "" || allowed.Path == "/" {
		return true
	}
	return pathCovers(allowed.Path, target.Path)
}

// pathCovers reports whether allowed is target, or a parent of it at a segment
// boundary. "/cb" covers "/cb" and "/cb/done" but not "/cbx".
func pathCovers(allowed, target string) bool {
	// Cleaning collapses "." and ".." so a path cannot climb out of what was
	// registered by spelling it differently.
	allowed = path.Clean("/" + strings.TrimPrefix(allowed, "/"))
	target = path.Clean("/" + strings.TrimPrefix(target, "/"))
	if allowed == target {
		return true
	}
	return strings.HasPrefix(target, strings.TrimSuffix(allowed, "/")+"/")
}

// isAbsoluteRedirect reports whether a resolved target leaves this origin, and
// so needs the session handed to it explicitly — a cookie will not follow a
// custom scheme or another host.
func isAbsoluteRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Scheme != "" || u.Host != ""
}

// withParam appends a query parameter to a redirect, whatever it already has.
func withParam(raw, key, value string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}

// linkCallbackURL resolves the URL an emailed sign-in, verification or reset
// link should point at.
//
// These append a single-use credential to a URL the *caller* supplies and then
// email the result to an address the caller also names. Unvalidated, that is a
// way to have somebody else's token delivered somewhere of your choosing: ask
// for a password reset for their address with a URL of yours, and the link they
// receive hands you the token when they click it.
//
// So the same registry that governs an OAuth redirect governs these. An empty
// URL is fine and means "send the bare token"; anything else has to be a target
// the project registered.
func linkCallbackURL(raw string, registered []string) (string, bool) {
	if raw == "" {
		return "", true
	}
	// A relative path is meaningless in an email — there is no origin to
	// resolve it against — so unlike a redirect it is not quietly accepted.
	if !isAbsoluteRedirect(raw) {
		return "", false
	}
	if safeRedirectTarget(raw, registered) == raw {
		return raw, true
	}
	return "", false
}
