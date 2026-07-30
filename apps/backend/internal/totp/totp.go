// Package totp implements the small slice of RFC 6238 / RFC 4226 that Applad
// needs for time-based one-time passwords: generating a shared secret, deriving
// the 6-digit code for a time step, and validating a submitted code with a
// ±1-step window for clock skew.
//
// It exists so the console-admin MFA flow and the per-project user MFA flow can
// share one implementation of the crypto rather than each carrying its own copy.
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math/big"
	"net/url"
	"time"
)

// b32 is base32 without padding — the encoding authenticator apps expect for
// the otpauth secret.
var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewSecret returns a fresh 20-byte secret, base32-encoded (no padding), ready
// to hand to an authenticator app and to store for later validation.
func NewSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("totp: generate secret: %w", err)
	}
	return b32.EncodeToString(secret), nil
}

// NewRecoveryCodes returns n single-use 8-digit recovery codes, used when the
// authenticator device is unavailable.
func NewRecoveryCodes(n int) ([]string, error) {
	codes := make([]string, n)
	for i := range codes {
		buf := make([]byte, 4)
		if _, err := rand.Read(buf); err != nil {
			return nil, fmt.Errorf("totp: generate recovery code: %w", err)
		}
		v := new(big.Int).SetBytes(buf)
		codes[i] = fmt.Sprintf("%08d", v.Int64()%100000000)
	}
	return codes, nil
}

// Validate reports whether code matches the secret for the current time step or
// either adjacent step (±30s), which tolerates a small clock difference between
// the server and the authenticator.
func Validate(secretB32, code string) bool {
	secret, err := b32.DecodeString(secretB32)
	if err != nil {
		return false
	}
	step := time.Now().Unix() / 30
	for _, offset := range []int64{-1, 0, 1} {
		// Constant-time compare so validation time does not leak how close a
		// submitted code was to the expected one.
		if subtle.ConstantTimeCompare([]byte(Generate(secret, step+offset)), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

// Generate derives the 6-digit HOTP code for a raw secret and counter, per
// RFC 4226 dynamic truncation.
func Generate(secret []byte, counter int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", truncated%1000000)
}

// OTPAuthURL builds the otpauth:// URI an authenticator app scans, with the
// issuer and account label a user sees in their app.
func OTPAuthURL(issuer, account, secretB32 string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secretB32)
	q.Set("issuer", issuer)
	return "otpauth://totp/" + label + "?" + q.Encode()
}
