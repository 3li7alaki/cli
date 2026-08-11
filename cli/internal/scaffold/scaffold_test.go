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

func TestSyncRewritesDriftAndReportsOnlyWhatChanged(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "repo")
	if err := New("demo", dir, false); err != nil {
		t.Fatal(err)
	}
	if written, err := Sync(dir, "demo"); err != nil || len(written) != 0 {
		t.Fatalf("fresh scaffold should already be canonical: %v %v", written, err)
	}
	drifted := filepath.Join(dir, "cli/internal/base/base.go")
	if err := os.WriteFile(drifted, []byte("package base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	written, err := Sync(dir, "demo")
	if err != nil || len(written) != 1 || written[0] != "cli/internal/base/base.go" {
		t.Fatalf("want only base.go rewritten, got %v %v", written, err)
	}
	b, err := os.ReadFile(drifted)
	if err != nil || !strings.Contains(string(b), "func Command(") {
		t.Fatalf("base.go not restored: %v", err)
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
