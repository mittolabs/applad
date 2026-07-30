package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvPairs(t *testing.T) {
	got, err := parseEnvPairs([]string{"A=1", "B=hello=world", "C="})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["A"] != "1" || got["B"] != "hello=world" || got["C"] != "" {
		t.Fatalf("unexpected map: %#v", got)
	}
	if _, err := parseEnvPairs([]string{"noequals"}); err == nil {
		t.Fatal("expected error for missing '='")
	}
	if _, err := parseEnvPairs([]string{"=v"}); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestReadFunctionSourceFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "handler.js")
	if err := os.WriteFile(file, []byte("export default () => 1"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readFunctionSource(file, "node20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "export default () => 1" {
		t.Fatalf("unexpected source: %q", got)
	}
}

func TestReadFunctionSourceDirRuntimeFilename(t *testing.T) {
	dir := t.TempDir()
	// A node runtime resolves to index.js even when other files are present.
	if err := os.WriteFile(filepath.Join(dir, "index.js"), []byte("main"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readFunctionSource(dir, "node20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "main" {
		t.Fatalf("expected index.js content, got %q", got)
	}
}

func TestReadFunctionSourceDirSingleFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "only.rb"), []byte("puts 1"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readFunctionSource(dir, "ruby")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "puts 1" {
		t.Fatalf("unexpected source: %q", got)
	}
}

func TestReadFunctionSourceDirAmbiguous(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.rb"), []byte("1"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.rb"), []byte("2"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readFunctionSource(dir, "ruby"); err == nil {
		t.Fatal("expected error for ambiguous directory")
	}
}
