package totp

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

func TestValidate_AcceptsCurrentCode(t *testing.T) {
	secretB32, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	secret, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secretB32)
	code := Generate(secret, time.Now().Unix()/30)
	if !Validate(secretB32, code) {
		t.Errorf("expected current code %q to validate", code)
	}
}

func TestValidate_RejectsWrongCode(t *testing.T) {
	secretB32, _ := NewSecret()
	if Validate(secretB32, "000000") {
		// 000000 could in theory be the real code; only fail if it is not.
		secret, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secretB32)
		if Generate(secret, time.Now().Unix()/30) != "000000" {
			t.Error("expected wrong code to be rejected")
		}
	}
}

func TestValidate_RejectsBadSecret(t *testing.T) {
	if Validate("!!!not-base32!!!", "123456") {
		t.Error("expected an invalid base32 secret to reject")
	}
}

func TestNewRecoveryCodes_CountAndShape(t *testing.T) {
	codes, err := NewRecoveryCodes(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 8 {
		t.Fatalf("expected 8 codes, got %d", len(codes))
	}
	for _, c := range codes {
		if len(c) != 8 {
			t.Errorf("expected 8-digit code, got %q", c)
		}
	}
}

func TestOTPAuthURL_HasIssuerAndSecret(t *testing.T) {
	uri := OTPAuthURL("Applad Console", "admin@test.com", "ABC234")
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Errorf("bad scheme: %q", uri)
	}
	if !strings.Contains(uri, "secret=ABC234") || !strings.Contains(uri, "issuer=Applad+Console") {
		t.Errorf("missing params: %q", uri)
	}
}
