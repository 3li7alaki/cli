package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSubstitutesContentAndPaths(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "repo")
	if err := New("demo", dir, false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "cli/cmd/demo/main.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "__TOOL__") || !strings.Contains(string(b), `"demo/internal/version"`) {
		t.Fatalf("placeholder not replaced in %s", path)
	}
}

func TestNewRefusesNonEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := New("demo", dir, false); err == nil {
		t.Fatal("expected refusal")
	}
}
