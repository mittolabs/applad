package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
)

// Password algorithm identifiers stored in users.password_algo. Applad only ever
// writes AlgoBcrypt; the rest exist so an account imported from another platform
// can be verified against the hash that platform produced, then re-hashed to
// bcrypt on the next successful sign-in (see passwordNeedsRehash).
const (
	AlgoBcrypt         = "bcrypt"
	AlgoArgon2id       = "argon2"          // PHC string, $argon2id$/$argon2i$
	AlgoScryptFirebase = "scrypt-firebase" // Firebase / Appwrite "ScryptModified"
	AlgoScrypt         = "scrypt"          // generic scrypt via params
	AlgoSHA256         = "sha256"
	AlgoSHA512         = "sha512"
	AlgoSHA1           = "sha1"
	AlgoMD5            = "md5"
	AlgoPlaintext      = "plaintext"
)

// Upper bounds on cost parameters. Because these parameters arrive with an
// imported credential (attacker-influenced) and are evaluated on the
// unauthenticated login path, an unbounded value would let one crafted account
// exhaust CPU/memory on every sign-in attempt. Anything above these is rejected
// as malformed rather than executed. The maxima sit well above every real-world
// configuration (Appwrite argon2 default m=65536; Firebase memCost=14).
const (
	maxArgonMemKiB     = 262144 // 256 MiB
	maxArgonTime       = 16
	maxArgonLanes      = 16
	maxScryptN         = 1 << 20
	maxScryptR         = 16
	maxScryptP         = 16
	maxFirebaseMemCost = 17 // N up to 2^17
	maxFirebaseRounds  = 16
)

// passwordNeedsRehash reports whether a successful verify against algo should be
// upgraded to the native bcrypt scheme. Everything that is not already bcrypt is
// rehashed, so imported credentials converge on bcrypt over time.
func passwordNeedsRehash(algo string) bool {
	return !strings.EqualFold(strings.TrimSpace(algo), AlgoBcrypt)
}

// verifyForeignPassword checks password against a stored hash produced by algo,
// using algo-specific params (decoded from users.password_params). It returns
// (matched, error); error is returned only for an unsupported algorithm or
// malformed stored data, never for an ordinary password mismatch (that is
// (false, nil)). An empty algo is treated as bcrypt for backward compatibility.
//
// A stored hash that is empty (whitespace only) is always a non-match: an
// imported account with no usable credential must never authenticate, and an
// empty-vs-empty comparison would otherwise accept any (or an empty) password.
func verifyForeignPassword(algo, hashStr string, params map[string]any, password string) (bool, error) {
	if strings.TrimSpace(hashStr) == "" {
		return false, nil
	}
	switch strings.ToLower(strings.TrimSpace(algo)) {
	case "", AlgoBcrypt:
		err := bcrypt.CompareHashAndPassword([]byte(hashStr), []byte(password))
		if err == nil {
			return true, nil
		}
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, err
	case AlgoArgon2id:
		return verifyArgon2(hashStr, password)
	case AlgoScryptFirebase:
		return verifyFirebaseScrypt(hashStr, params, password)
	case AlgoScrypt:
		return verifyScrypt(hashStr, params, password)
	case AlgoSHA256:
		return verifyDigest(sha256.New, hashStr, params, password), nil
	case AlgoSHA512:
		return verifyDigest(sha512.New, hashStr, params, password), nil
	case AlgoSHA1:
		return verifyDigest(sha1.New, hashStr, params, password), nil
	case AlgoMD5:
		return verifyDigest(md5.New, hashStr, params, password), nil
	case AlgoPlaintext:
		return subtle.ConstantTimeCompare([]byte(hashStr), []byte(password)) == 1, nil
	default:
		return false, fmt.Errorf("auth: unsupported password algorithm %q", algo)
	}
}

// verifyArgon2 verifies a PHC-encoded Argon2 hash, e.g.
// $argon2id$v=19$m=65536,t=3,p=2$<b64salt>$<b64hash>. Both argon2id and argon2i
// are accepted; the cost parameters and salt come from the string itself and are
// bounded before the (expensive) KDF runs.
func verifyArgon2(phc, password string) (bool, error) {
	parts := strings.Split(phc, "$")
	// ["", variant, "v=19", "m=..,t=..,p=..", salt, hash]
	if len(parts) != 6 {
		return false, fmt.Errorf("auth: malformed argon2 hash")
	}
	variant := parts[1]
	if variant != "argon2id" && variant != "argon2i" {
		return false, fmt.Errorf("auth: unsupported argon2 variant %q", variant)
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("auth: malformed argon2 version")
	}
	var m, t, p uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, fmt.Errorf("auth: malformed argon2 params")
	}
	if m == 0 || t == 0 || p == 0 || m > maxArgonMemKiB || t > maxArgonTime || p > maxArgonLanes {
		return false, fmt.Errorf("auth: argon2 cost parameters out of range")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false, fmt.Errorf("auth: malformed argon2 salt")
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, fmt.Errorf("auth: malformed argon2 hash")
	}
	var got []byte
	if variant == "argon2id" {
		got = argon2.IDKey([]byte(password), salt, t, m, uint8(p), uint32(len(want)))
	} else {
		got = argon2.Key([]byte(password), salt, t, m, uint8(p), uint32(len(want)))
	}
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// verifyFirebaseScrypt implements Firebase's modified-scrypt password scheme
// (also emitted by Appwrite as "ScryptModified"). The stored hash is the
// base64 of AES-256-CTR(signerKey) keyed by the first 32 bytes of
// scrypt(password, salt||saltSeparator, N=2^memCost, r=rounds, p=1, 64).
//
// Verified against Firebase's own published test vector in password_test.go.
func verifyFirebaseScrypt(hashStr string, params map[string]any, password string) (bool, error) {
	salt, err := base64.StdEncoding.DecodeString(paramStr(params, "salt"))
	if err != nil {
		return false, fmt.Errorf("auth: firebase scrypt: bad salt")
	}
	saltSep, err := base64.StdEncoding.DecodeString(paramStr(params, "saltSeparator"))
	if err != nil {
		return false, fmt.Errorf("auth: firebase scrypt: bad saltSeparator")
	}
	signerKey, err := base64.StdEncoding.DecodeString(paramStr(params, "signerKey"))
	if err != nil || len(signerKey) == 0 {
		// An empty signer key would produce an empty ciphertext that matches an
		// empty stored hash for ANY password — reject it as malformed.
		return false, fmt.Errorf("auth: firebase scrypt: bad signerKey")
	}
	rounds := paramInt(params, "rounds", 8)
	memCost := paramInt(params, "memCost", 14)
	if rounds <= 0 || rounds > maxFirebaseRounds || memCost <= 0 || memCost > maxFirebaseMemCost {
		return false, fmt.Errorf("auth: firebase scrypt: cost params out of range")
	}

	// salt || saltSeparator, on a fresh backing array so we never mutate salt.
	saltFull := make([]byte, 0, len(salt)+len(saltSep))
	saltFull = append(saltFull, salt...)
	saltFull = append(saltFull, saltSep...)

	dk, err := scrypt.Key([]byte(password), saltFull, 1<<memCost, rounds, 1, 64)
	if err != nil {
		return false, fmt.Errorf("auth: firebase scrypt: %w", err)
	}
	block, err := aes.NewCipher(dk[:32])
	if err != nil {
		return false, err
	}
	iv := make([]byte, aes.BlockSize) // zero IV, per the Firebase scheme
	out := make([]byte, len(signerKey))
	cipher.NewCTR(block, iv).XORKeyStream(out, signerKey)
	got := base64.StdEncoding.EncodeToString(out)
	return subtle.ConstantTimeCompare([]byte(got), []byte(strings.TrimSpace(hashStr))) == 1, nil
}

// verifyScrypt verifies a generic scrypt hash. params carry N, r, p, keyLen (ints)
// and salt (base64); the stored hash is compared base64. Cost parameters are
// bounded before the KDF runs.
func verifyScrypt(hashStr string, params map[string]any, password string) (bool, error) {
	salt, err := base64.StdEncoding.DecodeString(paramStr(params, "salt"))
	if err != nil || len(salt) == 0 {
		return false, fmt.Errorf("auth: scrypt: bad salt")
	}
	n := paramInt(params, "N", 16384)
	r := paramInt(params, "r", 8)
	p := paramInt(params, "p", 1)
	keyLen := paramInt(params, "keyLen", 32)
	if n < 2 || n > maxScryptN || r < 1 || r > maxScryptR || p < 1 || p > maxScryptP || keyLen < 16 || keyLen > 128 {
		return false, fmt.Errorf("auth: scrypt: cost params out of range")
	}
	dk, err := scrypt.Key([]byte(password), salt, n, r, p, keyLen)
	if err != nil {
		return false, fmt.Errorf("auth: scrypt: %w", err)
	}
	got := base64.StdEncoding.EncodeToString(dk)
	return subtle.ConstantTimeCompare([]byte(got), []byte(strings.TrimSpace(hashStr))) == 1, nil
}

// verifyDigest verifies a plain (optionally salted) hash digest. params:
//   - salt:     optional salt string
//   - order:    "password", "password+salt" (default), or "salt+password"
//   - encoding: "hex" (default) or "base64"
//
// Hex comparison is case-insensitive; base64 comparison is case-sensitive
// (base64 is a case-significant alphabet, so folding it would admit collisions).
func verifyDigest(newHash func() hash.Hash, hashStr string, params map[string]any, password string) bool {
	salt := paramStr(params, "salt")
	order := paramStr(params, "order")
	if order == "" {
		order = "password+salt"
	}
	var input string
	switch order {
	case "password":
		input = password
	case "salt+password":
		input = salt + password
	default: // password+salt
		input = password + salt
	}
	h := newHash()
	h.Write([]byte(input))
	sum := h.Sum(nil)

	stored := strings.TrimSpace(hashStr)
	var got string
	if strings.EqualFold(paramStr(params, "encoding"), "base64") {
		got = base64.StdEncoding.EncodeToString(sum)
	} else {
		got = hex.EncodeToString(sum)
		got = strings.ToLower(got)
		stored = strings.ToLower(stored)
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(stored)) == 1
}

func paramStr(p map[string]any, k string) string {
	if p == nil {
		return ""
	}
	if v, ok := p[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func paramInt(p map[string]any, k string, def int) int {
	if p == nil {
		return def
	}
	switch v := p[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
