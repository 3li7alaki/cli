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
	if written, removed, err := Sync(dir, "demo"); err != nil || len(written)+len(removed) != 0 {
		t.Fatalf("fresh scaffold should already be canonical: %v %v %v", written, removed, err)
	}
	drifted := filepath.Join(dir, "cli/internal/base/base.go")
	if err := os.WriteFile(drifted, []byte("package base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	written, _, err := Sync(dir, "demo")
	if err != nil || len(written) != 1 || written[0] != "cli/internal/base/base.go" {
		t.Fatalf("want only base.go rewritten, got %v %v", written, err)
	}
	b, err := os.ReadFile(drifted)
	if err != nil || !strings.Contains(string(b), "func Command(") {
		t.Fatalf("base.go not restored: %v", err)
	}
}

func TestSyncRemovesRetiredPointerButNeverRealContent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "repo")
	if err := New("demo", dir, false); err != nil {
		t.Fatal(err)
	}
	pointer := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(pointer, []byte("@AGENTS.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, removed, err := Sync(dir, "demo"); err != nil || len(removed) != 1 || removed[0] != "CLAUDE.md" {
		t.Fatalf("want the pointer removed, got %v %v", removed, err)
	}
	if _, err := os.Stat(pointer); !os.IsNotExist(err) {
		t.Fatalf("pointer still on disk: %v", err)
	}
	if err := os.WriteFile(pointer, []byte("real rules\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, removed, err := Sync(dir, "demo"); err != nil || len(removed) != 0 {
		t.Fatalf("a CLAUDE.md with content must never be deleted: %v %v", removed, err)
	}
}

func TestSyncRefusesRepoWithoutModule(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Sync(dir, "__TOOL__"); err == nil {
		t.Fatal("expected refusal on a repo with no cli/go.mod")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("refused sync still wrote files: %v", entries)
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
