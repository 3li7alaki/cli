package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cli/internal/scaffold"
)

func TestRuleAgentFiles(t *testing.T) {
	root := validRepo(t)
	if got := ruleAgentFiles(root); len(got) != 0 {
		t.Fatalf("pass: %v", got)
	}
	write(t, root, "CLAUDE.md", "wrong\n")
	if got := ruleAgentFiles(root); len(got) != 1 {
		t.Fatalf("fail: %v", got)
	}
}

func TestRuleGoMod(t *testing.T) {
	root := validRepo(t)
	if got := ruleGoMod(root); len(got) != 0 {
		t.Fatalf("pass: %v", got)
	}
	write(t, root, "cli/go.mod", "module demo\n\ngo 1.26\n\nrequire example.com/x v1.0.0\n")
	if got := ruleGoMod(root); len(got) != 1 {
		t.Fatalf("fail: %v", got)
	}
}

func TestRuleWorkflows(t *testing.T) {
	root := validRepo(t)
	if got := ruleWorkflows(root); len(got) != 0 {
		t.Fatalf("pass: %v", got)
	}
	write(t, root, ".github/workflows/ci.yml", "name: wrong\n")
	if got := ruleWorkflows(root); len(got) != 1 {
		t.Fatalf("fail: %v", got)
	}
}

func TestRuleDashes(t *testing.T) {
	root := validRepo(t)
	if got := ruleDashes(root); len(got) != 0 {
		t.Fatalf("pass: %v", got)
	}
	write(t, root, "bad.txt", "bad "+string(rune(0x2014))+" dash\n")
	got := ruleDashes(root)
	if len(got) != 1 || !strings.Contains(got[0].Problem, "line 1") {
		t.Fatalf("fail: %v", got)
	}
}

func TestRuleVersionTarget(t *testing.T) {
	root := validRepo(t)
	if got := ruleVersionTarget(root); len(got) != 0 {
		t.Fatalf("pass: %v", got)
	}
	b, _ := os.ReadFile(filepath.Join(root, ".github/workflows/release.yml"))
	write(t, root, ".github/workflows/release.yml", strings.Replace(string(b), "demo/internal/version.Version", "demo/internal/missing.Version", 1))
	if got := ruleVersionTarget(root); len(got) != 1 {
		t.Fatalf("fail: %v", got)
	}
}

func validRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "AGENTS.md", "rules\n")
	write(t, root, "CLAUDE.md", "@AGENTS.md\n")
	write(t, root, "cli/go.mod", "module demo\n\ngo 1.26\n")
	write(t, root, "cli/internal/version/version.go", "package version\n")
	for _, name := range []string{"ci.yml", "release.yml"} {
		b, err := scaffold.Canonical(".github/workflows/"+name, "demo")
		if err != nil {
			t.Fatal(err)
		}
		write(t, root, ".github/workflows/"+name, string(b))
	}
	return root
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
