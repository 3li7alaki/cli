package check

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cli/internal/scaffold"
)

type Finding struct {
	Path    string `json:"path"`
	Problem string `json:"problem"`
	Fix     string `json:"fix"`
}

func Run(root string) []Finding {
	var out []Finding
	for _, rule := range []func(string) []Finding{ruleAgentFiles, ruleGoMod, ruleWorkflows, ruleBase, ruleMainTest, ruleGitattributes, ruleDashes, ruleVersionTarget} {
		out = append(out, rule(root)...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Problem < out[j].Problem
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func ruleAgentFiles(root string) []Finding {
	var out []Finding
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		out = append(out, finding("AGENTS.md", "file is missing", "create AGENTS.md"))
	}
	// Any of these makes Claude Code read it INSTEAD of AGENTS.md (code.claude.com/docs/en/memory).
	for _, rel := range []string{"CLAUDE.md", ".claude/CLAUDE.md", "CLAUDE.local.md"} {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		if strings.TrimSuffix(string(b), "\n") == "@AGENTS.md" {
			out = append(out, finding(rel, "redundant: Claude Code reads AGENTS.md natively", "delete it (cli sync does)"))
		} else {
			out = append(out, finding(rel, "shadows AGENTS.md: Claude Code reads this file instead", "move its content into AGENTS.md and delete it"))
		}
	}
	return out
}

func ruleGoMod(root string) []Finding {
	path := filepath.Join(root, "cli/go.mod")
	b, err := os.ReadFile(path)
	if err != nil {
		return []Finding{finding("cli/go.mod", "file is missing", "create a dependency-free Go module")}
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "require ") || strings.TrimSpace(line) == "require (" {
			return []Finding{finding("cli/go.mod", "contains a require directive", "remove external dependencies")}
		}
	}
	return nil
}

func ruleWorkflows(root string) []Finding {
	var out []Finding
	for _, name := range []string{"ci.yml", "release.yml"} {
		rel := filepath.ToSlash(filepath.Join(".github/workflows", name))
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			out = append(out, finding(rel, "workflow is missing", "copy the canonical workflow"))
			continue
		}
		expected, _ := scaffold.Canonical(rel, ModuleName(root))
		if normalizeWorkflow(actual) != normalizeWorkflow(expected) {
			out = append(out, finding(rel, diffSummary(actual, expected), "replace it with the canonical workflow"))
		}
	}
	return out
}

// The shared argument contract drifted three ways across four repos before this rule
// existed: one tool answered -v and not -h, another the reverse, a third neither.
func ruleBase(root string) []Finding {
	rel := "cli/internal/base/base.go"
	expected, err := scaffold.Canonical(rel, ModuleName(root))
	if err != nil {
		return nil
	}
	actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return []Finding{finding(rel, "shared argument contract is missing", "run cli sync")}
	}
	if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
		return []Finding{finding(rel, "shared argument contract differs from canonical", "run cli sync, and change the template if the difference is wanted")}
	}
	return nil
}

// A scaffolded repo must keep the argument contract test that exercises base.go's
// stdout/stderr contract. main.go is repo-owned prose, so this guards the test, not it.
func ruleMainTest(root string) []Finding {
	rel := "cli/cmd/__TOOL__/main_test.go"
	name := ModuleName(root)
	expected, err := scaffold.Canonical(rel, name)
	if err != nil {
		return nil
	}
	diskRel := strings.ReplaceAll(filepath.FromSlash(rel), "__TOOL__", name)
	actual, err := os.ReadFile(filepath.Join(root, diskRel))
	if err != nil {
		return []Finding{finding(diskRel, "argument contract test is missing", "run cli sync")}
	}
	if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
		return []Finding{finding(diskRel, "argument contract test differs from canonical", "run cli sync, and change the template if the difference is wanted")}
	}
	return nil
}

// Copying a canonical file between repos carries its line endings along, which is how a
// CRLF LICENSE reached three repos. The policy that prevents it is itself canonical.
func ruleGitattributes(root string) []Finding {
	rel := ".gitattributes"
	expected, err := scaffold.Canonical(rel, ModuleName(root))
	if err != nil {
		return nil
	}
	actual, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return []Finding{finding(rel, "line-ending policy is missing", "run cli sync")}
	}
	if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
		return []Finding{finding(rel, "line-ending policy differs from canonical", "run cli sync, and change the template if the difference is wanted")}
	}
	return nil
}

func ruleDashes(root string) []Finding {
	var out []Finding
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			if err == nil && entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil || bytes.IndexByte(b, 0) >= 0 {
			return nil
		}
		scanner := bufio.NewScanner(bytes.NewReader(b))
		line := 0
		for scanner.Scan() {
			line++
			if strings.ContainsAny(scanner.Text(), "\u2013\u2014") {
				rel, _ := filepath.Rel(root, path)
				out = append(out, finding(filepath.ToSlash(rel), fmt.Sprintf("forbidden dash at line %d", line), "replace it with punctuation or a hyphen"))
			}
		}
		return nil
	})
	return out
}

func ruleVersionTarget(root string) []Finding {
	rel := ".github/workflows/release.yml"
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	re := regexp.MustCompile(`-X\s+([^\s"]+)\.Version=`)
	matches := re.FindAllStringSubmatch(string(b), -1)
	if len(matches) == 0 {
		return []Finding{finding(rel, "release ldflags has no -X package.Version target", "target the shipped internal/version package")}
	}
	var out []Finding
	for _, match := range matches {
		pkg := match[1]
		module := ModuleName(root)
		relPkg := strings.TrimPrefix(pkg, module+"/")
		if relPkg == pkg || relPkg == "" {
			out = append(out, finding(rel, "ldflags Version target is outside the module", "target "+module+"/internal/version.Version"))
			continue
		}
		if info, err := os.Stat(filepath.Join(root, "cli", filepath.FromSlash(relPkg))); err != nil || !info.IsDir() {
			out = append(out, finding(rel, "ldflags Version package does not exist: "+pkg, "create the package or correct the target"))
		}
	}
	return out
}

func finding(path, problem, fix string) Finding { return Finding{path, problem, fix} }

func ModuleName(root string) string {
	b, _ := os.ReadFile(filepath.Join(root, "cli/go.mod"))
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	return "__TOOL__"
}

var goVersion = regexp.MustCompile(`go-version:\s*['\"]?[0-9.]+['\"]?`)

func normalizeWorkflow(b []byte) string {
	return goVersion.ReplaceAllString(strings.TrimSpace(string(b)), "go-version: '<GO>'")
}

func diffSummary(actual, expected []byte) string {
	a, e := strings.Split(normalizeWorkflow(actual), "\n"), strings.Split(normalizeWorkflow(expected), "\n")
	changed := 0
	max := len(a)
	if len(e) > max {
		max = len(e)
	}
	for i := 0; i < max; i++ {
		if i >= len(a) || i >= len(e) || a[i] != e[i] {
			changed++
		}
	}
	return fmt.Sprintf("workflow differs from canonical (%d differing lines)", changed)
}
