package crypto

import (
	"bytes"
	"strings"
	"testing"
)

func testKey32() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return k
}

func TestSealOpenBytesRoundTrip(t *testing.T) {
	key := testKey32()
	plaintext := []byte("hello world")

	sealed, err := SealBytes(key, plaintext)
	if err != nil {
		t.Fatalf("SealBytes: %v", err)
	}
	if bytes.Contains(sealed, plaintext) {
		t.Fatalf("sealed payload should not contain plaintext bytes")
	}

	opened, err := OpenBytes(key, sealed)
	if err != nil {
		t.Fatalf("OpenBytes: %v", err)
	}
	if !bytes.Equal(opened, plaintext) {
		t.Fatalf("got %q, want %q", opened, plaintext)
	}
}

func TestSealBytesRejectsBadKeyLength(t *testing.T) {
	if _, err := SealBytes(make([]byte, 10), []byte("x")); err == nil {
		t.Fatalf("expected error for invalid key length")
	}
}

func TestOpenBytesFailsWithWrongKey(t *testing.T) {
	key1 := testKey32()
	key2 := testKey32()
	key2[0] ^= 0xFF

	sealed, err := SealBytes(key1, []byte("secret"))
	if err != nil {
		t.Fatalf("SealBytes: %v", err)
	}
	if _, err := OpenBytes(key2, sealed); err == nil {
		t.Fatalf("expected decryption to fail with wrong key")
	}
}

func TestSealOpenTokenRoundTrip(t *testing.T) {
	key := testKey32()
	plaintext := []byte(`{"foo":"bar"}`)

	token, err := SealToken("fe", 3, key, plaintext)
	if err != nil {
		t.Fatalf("SealToken: %v", err)
	}
	if !strings.HasPrefix(token, "fe3:") {
		t.Fatalf("unexpected token shape: %s", token)
	}

	resolve := func(v int) ([]byte, error) {
		if v != 3 {
			t.Fatalf("resolveKey called with unexpected version %d", v)
		}
		return key, nil
	}
	opened, version, err := OpenToken("fe", resolve, token)
	if err != nil {
		t.Fatalf("OpenToken: %v", err)
	}
	if version != 3 {
		t.Fatalf("got version %d, want 3", version)
	}
	if !bytes.Equal(opened, plaintext) {
		t.Fatalf("got %q, want %q", opened, plaintext)
	}
}

func TestOpenTokenRejectsWrongPrefix(t *testing.T) {
	key := testKey32()
	token, err := SealToken("dek", 1, key, []byte("x"))
	if err != nil {
		t.Fatalf("SealToken: %v", err)
	}
	_, _, err = OpenToken("fe", func(int) ([]byte, error) { return key, nil }, token)
	if err == nil {
		t.Fatalf("expected error opening a token with the wrong family prefix")
	}
}

func TestParseToken(t *testing.T) {
	prefix, version, body, ok := ParseToken("cv12:YWJj")
	if !ok || prefix != "cv" || version != 12 || body != "YWJj" {
		t.Fatalf("got (%q, %d, %q, %v)", prefix, version, body, ok)
	}

	if _, _, _, ok := ParseToken("not-a-token"); ok {
		t.Fatalf("expected ok=false for a non-token string")
	}
	if _, _, _, ok := ParseToken(""); ok {
		t.Fatalf("expected ok=false for an empty string")
	}
}
