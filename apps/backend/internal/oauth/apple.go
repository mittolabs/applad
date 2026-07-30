package oauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// applyAuxConfig folds provider-specific auxiliary fields (stored in
// ProjectOAuthConfig.Extra) into a resolved Provider. Providers with no aux
// fields are left untouched.
func applyAuxConfig(p *Provider, providerName string, cfg *ProjectOAuthConfig) {
	switch providerName {
	case "microsoft":
		// Wire the configured Azure tenant into the authorize/token endpoints.
		// The definition hardcodes /common/, which breaks single-tenant apps;
		// an empty tenant keeps the /common/ multi-tenant behaviour.
		tenant := strings.TrimSpace(cfg.Extra["tenantId"])
		if tenant == "" {
			tenant = "common"
		}
		p.AuthURL = strings.Replace(p.AuthURL, "/common/", "/"+tenant+"/", 1)
		p.TokenURL = strings.Replace(p.TokenURL, "/common/", "/"+tenant+"/", 1)
	case "apple":
		// Apple's client secret is a short-lived ES256 JWT signed from the
		// account's .p8 key (stored encrypted in ClientSecret) using the key id
		// and team id. Defer it to exchange time so it is always fresh.
		serviceID := cfg.ClientID
		keyID := strings.TrimSpace(cfg.Extra["keyId"])
		teamID := strings.TrimSpace(cfg.Extra["teamId"])
		p8 := cfg.ClientSecret
		if serviceID != "" && keyID != "" && teamID != "" && p8 != "" {
			p.ClientSecretFn = func(ctx context.Context) (string, error) {
				return appleClientSecret(serviceID, teamID, keyID, p8, time.Now())
			}
		}
	}
}

// appleClientSecret builds the ES256 JWT Apple requires as the OAuth client
// secret. Apple issues no static secret: the relying party signs a short JWT
// with the .p8 elliptic-curve private key from its developer account.
//
//	header: {alg:ES256, kid:keyID, typ:JWT}
//	claims: {iss:teamID, iat, exp, aud:https://appleid.apple.com, sub:serviceID}
//
// The ES256 signature is R||S, each big-endian padded to 32 bytes (P-256), then
// base64url-encoded — the JWS form Apple expects.
func appleClientSecret(serviceID, teamID, keyID, p8 string, now time.Time) (string, error) {
	key, err := parseP8ECKey(p8)
	if err != nil {
		return "", err
	}

	header := map[string]string{"alg": "ES256", "kid": keyID, "typ": "JWT"}
	claims := map[string]interface{}{
		"iss": teamID,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aud": "https://appleid.apple.com",
		"sub": serviceID,
	}

	hb, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	cb, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	signingInput := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(cb)
	digest := sha256.Sum256([]byte(signingInput))

	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		return "", fmt.Errorf("oauth: apple: sign: %w", err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// parseP8ECKey parses an Apple .p8 PEM private key. Apple ships PKCS#8, but the
// SEC1 ("EC PRIVATE KEY") form is accepted too.
func parseP8ECKey(p8 string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(p8))
	if block == nil {
		return nil, fmt.Errorf("oauth: apple: invalid PEM private key")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		ec, ok := k.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("oauth: apple: p8 is not an EC key")
		}
		return ec, nil
	}
	ec, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("oauth: apple: parse p8: %w", err)
	}
	return ec, nil
}
