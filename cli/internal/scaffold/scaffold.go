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
	"CLAUDE.md",
	"cli/internal/base/base.go",
}

// Sync rewrites the canonical files into an existing repo and reports the ones it changed.
func Sync(root, name string) ([]string, error) {
	var written []string
	for _, rel := range SyncFiles {
		data, err := Canonical(rel, name)
		if err != nil {
			return written, err
		}
		target := filepath.Join(root, filepath.FromSlash(rel))
		if old, err := os.ReadFile(target); err == nil && bytes.Equal(old, data) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return written, err
		}
		written = append(written, rel)
	}
	return written, nil
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
