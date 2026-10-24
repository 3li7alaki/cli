package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAgentReferenceFindings(t *testing.T) {
	root := validRepo(t)
	write(t, root, "AGENTS.md", "# Rules\nRead `docs/renamed.md:17` and `cli/go.mod:3`.\n@docs/old.md\n")
	got := Run(root)
	if len(got) != 2 {
		t.Fatalf("want two stale references, got %v", got)
	}
	for i, want := range []struct {
		line int
		ref  string
		kind string
	}{
		{2, "docs/renamed.md:17", "backtick"},
		{3, "@docs/old.md", "@import"},
	} {
		f := got[i]
		if f.Path != "AGENTS.md" || !strings.Contains(f.Problem, fmt.Sprintf("line %d", want.line)) ||
			!strings.Contains(f.Problem, want.ref) || !strings.Contains(f.Problem, want.kind) ||
			!strings.Contains(f.Fix, "correct") || !strings.Contains(f.Fix, "remove") {
			t.Errorf("finding %d: %+v", i, f)
		}
	}
}

func TestAgentReferenceCandidates(t *testing.T) {
	root := t.TempDir()
	write(t, root, "cli/go.mod", "module demo\n")
	for _, tt := range []struct {
		ref    string
		broken bool
	}{
		{"docs/missing.md", true},
		{"docs/missing/", true},
		{"docs/missing.md:42", true},
		{"../missing.md", true},
		{"cli/go.mod", false},
		{"cli/go.mod:3", false},
		{"cli/:3", false},
		{"README.md", false},
		{"io/fs", false},
		{"cli/cmd/cli", false},
		{"folder.name/command", false},
		{"docs/a b.md", false},
		{"docs/a\tb.md", false},
		{"docs/a\u00a0b.md", false},
		{"docs/<name>.md", false},
		{"docs/name>.md", false},
		{"docs/*.md", false},
		{"$ROOT/file.md", false},
		{"docs/{name.md", false},
		{"docs/name}.md", false},
		{"https://example.com/file.md", false},
		{"~/docs/file.md", false},
		{"~person/docs/file.md", false},
		{"--dir=docs/file.md", false},
	} {
		t.Run(tt.ref, func(t *testing.T) {
			write(t, root, "AGENTS.md", "Read `"+tt.ref+"`.\n")
			got := ruleAgentRefs(root)
			if (len(got) == 1) != tt.broken || len(got) > 1 {
				t.Fatalf("broken = %v, got %v", tt.broken, got)
			}
		})
	}
}

func TestAgentReferenceFencesAndLines(t *testing.T) {
	root := t.TempDir()
	write(t, root, "AGENTS.md", strings.Join([]string{
		"# Rules",
		"  ```text `docs/fence.md`",
		"@docs/inside.md",
		"`docs/also-inside.md`",
		"  ````",
		"Read `docs/first.md` and `docs/second.md:9`, then `docs/unclosed.md",
		"```",
		"@docs/unclosed-fence.md",
	}, "\r\n"))
	got := ruleAgentRefs(root)
	if len(got) != 2 {
		t.Fatalf("want two references outside fences, got %v", got)
	}
	for i, ref := range []string{"docs/first.md", "docs/second.md:9"} {
		if !strings.Contains(got[i].Problem, "line 6") || !strings.Contains(got[i].Problem, ref) {
			t.Errorf("finding %d: %+v", i, got[i])
		}
	}
}

func TestAgentReferenceResolution(t *testing.T) {
	root, home, outside := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	write(t, home, "rules.md", "rules\n")
	write(t, outside, "absolute.md", "rules\n")
	write(t, root, "local", "rules\n")
	abs := filepath.Join(outside, "absolute.md")
	missing := filepath.Join(outside, "missing.md")
	write(t, root, "AGENTS.md", strings.Join([]string{
		"@~/rules.md",
		"@" + abs,
		"`" + abs + ":7`",
		"  @local  ",
		"@",
		"@docs/ignored.md extra",
		"@docs/ignored\u2003.md",
		"@~/missing.md",
		"`" + missing + "`",
		"@" + missing,
	}, "\n"))
	got := ruleAgentRefs(root)
	if len(got) != 3 {
		t.Fatalf("want three unresolved references, got %v", got)
	}
	for i, want := range []string{"@~/missing.md", missing, "@" + missing} {
		if !strings.Contains(got[i].Problem, want) {
			t.Errorf("finding %d: %+v", i, got[i])
		}
	}
	t.Setenv("HOME", "")
	write(t, root, "AGENTS.md", "@~/rules.md\n")
	if got := ruleAgentRefs(root); len(got) != 1 {
		t.Fatalf("missing home directory must leave a broken reference: %v", got)
	}
}

func TestAgentReferenceReadScope(t *testing.T) {
	root := t.TempDir()
	write(t, root, "docs/AGENTS.md", "@missing.md\n")
	write(t, root, "CLAUDE.md", "@missing.md\n")
	if got := ruleAgentRefs(root); got != nil {
		t.Fatalf("missing root AGENTS.md: %v", got)
	}
	if err := os.Mkdir(filepath.Join(root, "AGENTS.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ruleAgentRefs(root); got != nil {
		t.Fatalf("unreadable root AGENTS.md: %v", got)
	}
	if err := os.Remove(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	write(t, root, "AGENTS.md", "Rules without references.\n")
	if got := ruleAgentRefs(root); got != nil {
		t.Fatalf("references in other files must be ignored: %v", got)
	}
}

func TestAgentReferenceLongLine(t *testing.T) {
	root := t.TempDir()
	write(t, root, "AGENTS.md", strings.Repeat("x", 70_000)+"\n`docs/missing.md`\n")
	got := ruleAgentRefs(root)
	if len(got) != 1 || !strings.Contains(got[0].Problem, "line 2") {
		t.Fatalf("reference after a long line: %v", got)
	}
}
