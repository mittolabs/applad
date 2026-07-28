package runtime

import (
	"strings"
	"testing"
)

func TestLineWriter_SplitsAndFlushes(t *testing.T) {
	var got []string
	w := &lineWriter{fn: func(s string) { got = append(got, s) }}

	// Writes may split mid-line; only complete lines fire until Flush.
	w.Write([]byte("Step 1/5\nStep 2"))
	if len(got) != 1 || got[0] != "Step 1/5" {
		t.Fatalf("after partial write, got %v", got)
	}
	w.Write([]byte("/5\r\n"))
	if len(got) != 2 || got[1] != "Step 2/5" {
		t.Fatalf("expected CR trimmed and line completed, got %v", got)
	}
	w.Write([]byte("tail-without-newline"))
	if len(got) != 2 {
		t.Fatalf("no newline should not fire yet, got %v", got)
	}
	w.Flush()
	if len(got) != 3 || got[2] != "tail-without-newline" {
		t.Fatalf("flush should emit the remainder, got %v", got)
	}
	// Blank lines are dropped.
	got = nil
	w.Write([]byte("\n   \n"))
	if len(got) != 0 {
		t.Fatalf("blank lines should be dropped, got %v", strings.Join(got, "|"))
	}
}
