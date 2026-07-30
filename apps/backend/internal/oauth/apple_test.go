package oauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func genP8(t *testing.T) (string, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return string(pemBytes), key
}

// appleClientSecret produces a well-formed, correctly-signed ES256 JWT with the
// claims Apple requires.
func TestAppleClientSecret_SignedAndWellFormed(t *testing.T) {
	p8, key := genP8(t)
	now := time.Unix(1_700_000_000, 0)

	jwt, err := appleClientSecret("com.example.svc", "TEAMID1234", "KEYID5678", p8, now)
	if err != nil {
		t.Fatalf("appleClientSecret: %v", err)
	}

	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT segments, got %d", len(parts))
	}

	var header map[string]string
	hb, _ := base64.RawURLEncoding.DecodeString(parts[0])
	if err := json.Unmarshal(hb, &header); err != nil {
		t.Fatalf("header decode: %v", err)
	}
	if header["alg"] != "ES256" || header["kid"] != "KEYID5678" {
		t.Errorf("bad header: %+v", header)
	}

	var claims map[string]interface{}
	cb, _ := base64.RawURLEncoding.DecodeString(parts[1])
	if err := json.Unmarshal(cb, &claims); err != nil {
		t.Fatalf("claims decode: %v", err)
	}
	if claims["iss"] != "TEAMID1234" || claims["sub"] != "com.example.svc" {
		t.Errorf("bad claims: %+v", claims)
	}
	if claims["aud"] != "https://appleid.apple.com" {
		t.Errorf("aud = %v", claims["aud"])
	}
	if int64(claims["iat"].(float64)) != now.Unix() {
		t.Errorf("iat = %v, want %d", claims["iat"], now.Unix())
	}
	if int64(claims["exp"].(float64)) <= now.Unix() {
		t.Errorf("exp must be in the future of iat")
	}

	// Verify the signature against the public key.
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	if len(sig) != 64 {
		t.Fatalf("signature length = %d, want 64", len(sig))
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(&key.PublicKey, digest[:], r, s) {
		t.Fatal("signature does not verify against the public key")
	}
}

func TestAppleClientSecret_RejectsBadKey(t *testing.T) {
	if _, err := appleClientSecret("svc", "team", "key", "not a pem key", time.Now()); err == nil {
		t.Fatal("expected error for invalid p8")
	}
}

// applyAuxConfig wires Apple's ClientSecretFn from the stored p8 + extra fields,
// and ExchangeCode then invokes it (verified indirectly: the fn yields a JWT).
func TestApplyAuxConfig_AppleSecretFn(t *testing.T) {
	p8, _ := genP8(t)
	p := AllProviderDefinitions()["apple"].ToProvider("com.example.svc", p8)
	applyAuxConfig(p, "apple", &ProjectOAuthConfig{
		ClientID:     "com.example.svc",
		ClientSecret: p8,
		Extra:        map[string]string{"keyId": "KEYID5678", "teamId": "TEAMID1234"},
	})
	if p.ClientSecretFn == nil {
		t.Fatal("expected ClientSecretFn to be set for Apple")
	}
	secret, err := p.ClientSecretFn(context.Background())
	if err != nil {
		t.Fatalf("ClientSecretFn: %v", err)
	}
	if strings.Count(secret, ".") != 2 {
		t.Errorf("expected a JWT client secret, got %q", secret)
	}
}

// Without the aux fields, no client-secret function is wired (nothing to sign
// with), so Apple stays honestly unconfigured rather than silently broken.
func TestApplyAuxConfig_AppleMissingFieldsNoFn(t *testing.T) {
	p := AllProviderDefinitions()["apple"].ToProvider("com.example.svc", "")
	applyAuxConfig(p, "apple", &ProjectOAuthConfig{ClientID: "com.example.svc"})
	if p.ClientSecretFn != nil {
		t.Fatal("ClientSecretFn should be nil when key/team/p8 are absent")
	}
}
