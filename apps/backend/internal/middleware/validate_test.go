package middleware

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"user+tag@sub.domain.com", true},
		{"a@b.co", true},
		{"", false},
		{"not-email", false},
		{"@domain.com", false},
		{"user@", false},
		{"user@.com", false},
		{strings.Repeat("a", 255) + "@b.com", false}, // too long
	}
	for _, tc := range tests {
		got := ValidateEmail(tc.email)
		if got != tc.want {
			t.Errorf("ValidateEmail(%q) = %v, want %v", tc.email, got, tc.want)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		pw   string
		want bool
	}{
		{"12345678", true},
		{strings.Repeat("a", 256), true},
		{"short", false},
		{"", false},
		{strings.Repeat("a", 257), false},
	}
	for _, tc := range tests {
		got := ValidatePassword(tc.pw)
		if got != tc.want {
			t.Errorf("ValidatePassword(len=%d) = %v, want %v", len(tc.pw), got, tc.want)
		}
	}
}

func TestSanitizeString(t *testing.T) {
	if s := SanitizeString("  hello  ", 100); s != "hello" {
		t.Fatalf("expected 'hello', got %q", s)
	}
	if s := SanitizeString("abcdef", 3); s != "abc" {
		t.Fatalf("expected 'abc', got %q", s)
	}
}
