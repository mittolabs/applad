package credentials

import (
	"os"
	"testing"
)

func TestEncryptDecryptSecret_RoundTrip(t *testing.T) {
	os.Setenv("CREDENTIALS_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	plaintext := "an-oauth-client-secret"
	token, err := EncryptSecret(plaintext)
	if err != nil {
		t.Fatalf("EncryptSecret: %v", err)
	}
	if token == plaintext {
		t.Fatal("token must not equal the plaintext")
	}
	if !IsEncryptedSecret(token) {
		t.Fatal("token should be recognised as encrypted")
	}

	got, err := DecryptSecret(token)
	if err != nil {
		t.Fatalf("DecryptSecret: %v", err)
	}
	if got != plaintext {
		t.Fatalf("round-trip = %q, want %q", got, plaintext)
	}
}

// A value without the token marker is treated as legacy plaintext and returned
// verbatim, so a store that predates encryption keeps working.
func TestDecryptSecret_LegacyPlaintextPassthrough(t *testing.T) {
	os.Setenv("CREDENTIALS_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	legacy := "plain-old-secret"
	if IsEncryptedSecret(legacy) {
		t.Fatal("plaintext should not look encrypted")
	}
	got, err := DecryptSecret(legacy)
	if err != nil {
		t.Fatalf("DecryptSecret: %v", err)
	}
	if got != legacy {
		t.Fatalf("passthrough = %q, want %q", got, legacy)
	}
}

func TestEncryptSecret_Nondeterministic(t *testing.T) {
	os.Setenv("CREDENTIALS_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	a, _ := EncryptSecret("same")
	b, _ := EncryptSecret("same")
	if a == b {
		t.Fatal("expected a fresh nonce per encryption, got identical tokens")
	}
}
