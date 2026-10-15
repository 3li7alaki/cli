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
	write(t, root, "CLAUDE.md", "@AGENTS.md\n")
	if got := ruleAgentFiles(root); len(got) != 1 || !strings.Contains(got[0].Problem, "redundant") {
		t.Fatalf("a leftover pointer must be reported redundant: %v", got)
	}
	write(t, root, "CLAUDE.md", "real rules\n")
	if got := ruleAgentFiles(root); len(got) != 1 || !strings.Contains(got[0].Problem, "shadows") {
		t.Fatalf("a CLAUDE.md with content must be reported as shadowing AGENTS.md: %v", got)
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

func TestRuleBase(t *testing.T) {
	root := validRepo(t)
	if got := ruleBase(root); len(got) != 0 {
		t.Fatalf("pass: %v", got)
	}
	write(t, root, "cli/internal/base/base.go", "package base\n\nfunc Command(args []string) string { return \"\" }\n")
	if got := ruleBase(root); len(got) != 1 {
		t.Fatalf("a repo that quietly changed the shared contract must be reported: %v", got)
	}
}

func TestRuleMainTest(t *testing.T) {
	root := validRepo(t)
	if got := ruleMainTest(root); len(got) != 0 {
		t.Fatalf("pass: %v", got)
	}
	write(t, root, "cli/cmd/demo/main_test.go", "package main\n")
	if got := ruleMainTest(root); len(got) != 1 {
		t.Fatalf("a repo that quietly changed the argument contract test must be reported: %v", got)
	}
}

func TestRuleGitattributes(t *testing.T) {
	root := validRepo(t)
	if got := ruleGitattributes(root); len(got) != 0 {
		t.Fatalf("pass: %v", got)
	}
	if err := os.Remove(filepath.Join(root, ".gitattributes")); err != nil {
		t.Fatal(err)
	}
	if got := ruleGitattributes(root); len(got) != 1 {
		t.Fatalf("a repo that deleted the line-ending policy must be reported: %v", got)
	}
}

func validRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "AGENTS.md", "rules\n")
	write(t, root, "cli/go.mod", "module demo\n\ngo 1.26\n")
	gitattributes, err := scaffold.Canonical(".gitattributes", "demo")
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, ".gitattributes", string(gitattributes))
	write(t, root, "cli/internal/version/version.go", "package version\n")
	base, err := scaffold.Canonical("cli/internal/base/base.go", "demo")
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, "cli/internal/base/base.go", string(base))
	mainTest, err := scaffold.Canonical("cli/cmd/__TOOL__/main_test.go", "demo")
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, "cli/cmd/demo/main_test.go", string(mainTest))
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
