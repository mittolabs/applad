// Package crypto provides the single AES-256-GCM seal/open primitive used
// everywhere Applad encrypts a secret at rest: credential vault entries,
// storage files, and field-encrypted database columns. Centralizing it here
// means every caller gets the same nonce handling and the same
// self-describing token format, instead of each package hand-rolling its own
// AES-GCM boilerplate.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// SealBytes encrypts plaintext with AES-GCM under key (16, 24, or 32 bytes for
// AES-128/192/256 — new callers should use 32-byte AES-256 keys; shorter sizes
// are accepted only so existing legacy keys, e.g. a pre-existing
// STORAGE_ENCRYPTION_KEY, keep decrypting what they already encrypted),
// returning the raw binary payload: a random nonce prepended to the
// ciphertext+tag. Use this for bulk data (e.g. file contents) where a
// text-token encoding would add unwanted overhead.
func SealBytes(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// OpenBytes reverses SealBytes.
func OpenBytes(key, sealed []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(sealed) < ns {
		return nil, fmt.Errorf("crypto: ciphertext too short")
	}
	nonce, ct := sealed[:ns], sealed[ns:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt: %w", err)
	}
	return plaintext, nil
}

// SealToken encrypts plaintext under key and returns a self-describing,
// versioned, base64 token: "<prefix><version>:<base64(nonce||ciphertext)>".
// prefix distinguishes token families sharing this format (e.g. "cv" for
// credentials, "dek" for wrapped project keys, "fe" for field-encrypted row
// data) so tokens from different subsystems are never cross-decoded, and
// version lets the caller resolve which key produced this token even after
// that subsystem has since rotated to a newer one.
func SealToken(prefix string, version int, key, plaintext []byte) (string, error) {
	sealed, err := SealBytes(key, plaintext)
	if err != nil {
		return "", err
	}
	return prefix + strconv.Itoa(version) + ":" + base64.StdEncoding.EncodeToString(sealed), nil
}

// OpenToken reverses SealToken. resolveKey is called with the version parsed
// from the token and must return the key that produced it (e.g. by looking up
// a rotated-away key version).
func OpenToken(prefix string, resolveKey func(version int) ([]byte, error), token string) (plaintext []byte, version int, err error) {
	gotPrefix, v, body, ok := ParseToken(token)
	if !ok || gotPrefix != prefix {
		return nil, 0, fmt.Errorf("crypto: not a valid %q token", prefix)
	}
	key, err := resolveKey(v)
	if err != nil {
		return nil, 0, err
	}
	sealed, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, 0, fmt.Errorf("crypto: base64 decode: %w", err)
	}
	pt, err := OpenBytes(key, sealed)
	if err != nil {
		return nil, 0, err
	}
	return pt, v, nil
}

// ParseToken splits a token into (prefix, version, base64Body) without
// decrypting it, e.g. for validation/introspection or a legacy-plaintext
// fallback check ("does this value even look encrypted?").
func ParseToken(token string) (prefix string, version int, body string, ok bool) {
	// prefix is the leading run of non-digit characters, e.g. "cv" in "cv1:...".
	i := 0
	for i < len(token) && (token[i] < '0' || token[i] > '9') {
		i++
	}
	if i == 0 || i >= len(token) {
		return "", 0, "", false
	}
	prefix = token[:i]
	rest := token[i:]
	colon := strings.IndexByte(rest, ':')
	if colon <= 0 {
		return "", 0, "", false
	}
	v, err := strconv.Atoi(rest[:colon])
	if err != nil {
		return "", 0, "", false
	}
	return prefix, v, rest[colon+1:], true
}

func newGCM(key []byte) (cipher.AEAD, error) {
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("crypto: key must be 16, 24, or 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm: %w", err)
	}
	return gcm, nil
}
