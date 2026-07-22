package uid

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
)

// New returns a unique ID. If hint is "unique()" or empty, generate one.
// A hint survives only if it is a valid ID: IDs end up in filesystem paths
// (storage) and URLs, so "." or "/" in a caller-supplied hint is traversal,
// not identity — such hints are discarded, not sanitised in place.
func New(hint string) string {
	if hint == "" || hint == "unique()" || !ValidID(hint) {
		return generate()
	}
	return hint
}

// ValidID reports whether s is safe to use as an ID: 1–128 chars of
// [A-Za-z0-9_-]. Callers that receive an ID off a URL rather than minting one
// (chunked uploads) use this to refuse anything path-shaped.
func ValidID(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}

func generate() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

// RandomHex returns n random hex-encoded bytes.
func RandomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}
