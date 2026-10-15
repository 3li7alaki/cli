package scaffold

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:template
var files embed.FS

// Canonical returns a template file, with __TOOL__ replaced by name. Paths are
// relative to the scaffolded repo root, so ".github/workflows/ci.yml".
func Canonical(rel, name string) ([]byte, error) {
	b, err := files.ReadFile("template/" + rel)
	if err != nil {
		return nil, err
	}
	return []byte(strings.ReplaceAll(string(b), "__TOOL__", name)), nil
}

// SyncFiles are the files `cli check` enforces and `cli sync` reinstalls. Files a repo
// owns are deliberately absent, so sync stays safe to run on a repo full of real work:
// AGENTS.md is its own prose, main.go is its own commands, go.mod is its own module.
var SyncFiles = []string{
	".gitattributes",
	".github/workflows/ci.yml",
	".github/workflows/release.yml",
	"cli/cmd/__TOOL__/main_test.go",
	"cli/internal/base/base.go",
}

// Retired are files the template used to ship, keyed to the exact content it shipped.
// Sync deletes one only while it still holds that content, so a file the repo has since
// written real prose into is reported by check and never touched.
var Retired = map[string]string{
	// Claude Code reads AGENTS.md natively since v2.1.277 when no CLAUDE.md exists, so
	// the pointer is dead weight, and a CLAUDE.md beside AGENTS.md shadows it.
	"CLAUDE.md": "@AGENTS.md",
}

// Sync rewrites the canonical files into an existing repo, deletes retired ones, and
// reports what it changed.
func Sync(root, name string) (written, removed []string, err error) {
	// check.ModuleName falls back to the placeholder when cli/go.mod is absent. Syncing
	// then would plant a literal cli/cmd/__TOOL__/ tree into a repo that was never ours.
	if name == "" || strings.Contains(name, "__TOOL__") {
		return nil, nil, errors.New("cli/go.mod declares no module: sync only runs on a cli-scaffolded repo")
	}
	for rel, shipped := range Retired {
		target := filepath.Join(root, filepath.FromSlash(rel))
		old, readErr := os.ReadFile(target)
		if readErr != nil || strings.TrimSuffix(string(old), "\n") != shipped {
			continue
		}
		if err := os.Remove(target); err != nil {
			return written, removed, err
		}
		removed = append(removed, rel)
	}
	for _, rel := range SyncFiles {
		data, err := Canonical(rel, name)
		if err != nil {
			return written, removed, err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		for i := range parts {
			parts[i] = strings.ReplaceAll(parts[i], "__TOOL__", name)
		}
		target := filepath.Join(append([]string{root}, parts...)...)
		if old, err := os.ReadFile(target); err == nil && bytes.Equal(old, data) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return written, removed, err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return written, removed, err
		}
		written = append(written, strings.Join(parts, "/"))
	}
	return written, removed, nil
}

func New(name, dir string, force bool) error {
	if name == "" || strings.ContainsAny(name, `/\\`) {
		return errors.New("name must be a non-empty path component")
	}
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(entries) > 0 && !force {
		return fmt.Errorf("destination %s is not empty (use --force)", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(files, "template", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel("template", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		for i := range parts {
			parts[i] = strings.ReplaceAll(parts[i], "__TOOL__", name)
		}
		target := filepath.Join(append([]string{dir}, parts...)...)
		target = strings.TrimSuffix(target, ".tmpl")
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if filepath.Base(target) == ".git" {
			if _, err := os.Stat(target); err == nil {
				return nil
			}
		}
		data, err := files.ReadFile(path)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0o644)
		if filepath.Base(target) == "install.sh" {
			mode = 0o755
		}
		return os.WriteFile(target, []byte(strings.ReplaceAll(string(data), "__TOOL__", name)), mode)
	})
}
