package uid

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
)

// New returns a unique ID. If hint is "unique()" or empty, generate one.
// Otherwise sanitise and return hint.
func New(hint string) string {
	if hint == "" || hint == "unique()" {
		return generate()
	}
	return hint
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
