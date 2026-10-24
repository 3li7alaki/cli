package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuleAgentRefs(t *testing.T) {
	root := validRepo(t)
	if got := ruleAgentRefs(root); len(got) != 0 {
		t.Fatalf("an AGENTS.md with no references must pass: %v", got)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	write(t, home, "notes/global.md", "x\n")
	write(t, root, "AGENTS.md", strings.Join([]string{
		"# demo",
		"Edit `cli/go.mod` and `cli/internal/version/` freely.",
		"See `cli/go.mod:3`, `io/fs`, `cli new <name>`, `<repo>/AGENTS.md`, `https://example.com/a.md`, `docs/*.md`, `~/gone/x.md`.",
		"The guide is `docs/gone.md`.",
		"```",
		"cat `cli/fenced-missing.go`",
		"@docs/fenced-import.md",
		"```",
		"@cli/go.mod",
		"@~/notes/global.md",
		"@docs/rules.md",
		"",
	}, "\n"))

	got := ruleAgentRefs(root)
	if len(got) != 2 {
		t.Fatalf("want exactly the two broken references (line 4 and line 11), got %d: %v", len(got), got)
	}
	for _, want := range []struct{ line, ref string }{{"line 4", "docs/gone.md"}, {"line 11", "docs/rules.md"}} {
		ok := false
		for _, f := range got {
			if f.Path == "AGENTS.md" && strings.Contains(f.Problem, want.line) && strings.Contains(f.Problem, want.ref) && f.Fix != "" {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("no finding on AGENTS.md naming %s and %s: %v", want.line, want.ref, got)
		}
	}

	inRun := false
	for _, f := range Run(root) {
		if f.Path == "AGENTS.md" && strings.Contains(f.Problem, "docs/gone.md") {
			inRun = true
		}
	}
	if !inRun {
		t.Fatal("Run must include ruleAgentRefs")
	}

	write(t, root, "docs/gone.md", "x\n")
	write(t, root, "docs/rules.md", "x\n")
	if got := ruleAgentRefs(root); len(got) != 0 {
		t.Fatalf("every reference resolves now: %v", got)
	}

	if err := os.Remove(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if got := ruleAgentRefs(root); len(got) != 0 {
		t.Fatalf("a missing AGENTS.md is ruleAgentFiles' finding, not this rule's: %v", got)
	}
}
