package uid

import (
	"testing"
)

func TestNew_Unique(t *testing.T) {
	id := New("unique()")
	if id == "" {
		t.Fatal("expected non-empty id")
	}
	if len(id) != 32 {
		t.Fatalf("expected 32-char hex id, got %d chars: %s", len(id), id)
	}
}

func TestNew_Empty(t *testing.T) {
	id := New("")
	if id == "" {
		t.Fatal("expected non-empty id for empty hint")
	}
	if len(id) != 32 {
		t.Fatalf("expected 32-char hex id, got %d chars: %s", len(id), id)
	}
}

func TestNew_CustomHint(t *testing.T) {
	id := New("my-custom-id")
	if id != "my-custom-id" {
		t.Fatalf("expected 'my-custom-id', got %s", id)
	}
}

func TestNew_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := New("unique()")
		if seen[id] {
			t.Fatalf("duplicate id generated: %s", id)
		}
		seen[id] = true
	}
}

func TestRandomHex(t *testing.T) {
	hex := RandomHex(16)
	if len(hex) != 32 { // 16 bytes = 32 hex chars
		t.Fatalf("expected 32 hex chars, got %d: %s", len(hex), hex)
	}

	// Uniqueness
	hex2 := RandomHex(16)
	if hex == hex2 {
		t.Fatal("two calls to RandomHex returned same value")
	}
}
