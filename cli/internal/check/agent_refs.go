package check

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// The tafaol rename left 17 stale references in AGENTS.md.
func ruleAgentRefs(root string) []Finding {
	b, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		return nil
	}
	var out []Finding
	checkRef := func(kind, ref, target string, line int) {
		if !agentRefExists(root, target) {
			out = append(out, finding("AGENTS.md", fmt.Sprintf("line %d: broken %s reference: %s", line, kind, ref), "correct or remove the reference"))
		}
	}
	lineNumber := regexp.MustCompile(`:[0-9]+$`)
	fenced := false
	for i, line := range strings.Split(string(b), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if strings.HasPrefix(trimmed, "@") && len(trimmed) > 1 && strings.IndexFunc(trimmed, unicode.IsSpace) < 0 {
			checkRef("@import", trimmed, trimmed[1:], i+1)
		}
		spans := strings.Split(line, "`")
		for j := 1; j+1 < len(spans); j += 2 {
			ref := spans[j]
			target := lineNumber.ReplaceAllString(ref, "")
			slash := strings.LastIndex(target, "/")
			if slash < 0 || strings.IndexFunc(target, unicode.IsSpace) >= 0 ||
				strings.ContainsAny(target, "<>*${}") || strings.Contains(target, "://") ||
				strings.HasPrefix(target, "~") || strings.HasPrefix(target, "-") {
				continue
			}
			if !strings.HasSuffix(target, "/") && !strings.Contains(target[slash+1:], ".") {
				continue
			}
			checkRef("backticked", ref, target, i+1)
		}
	}
	return out
}

func agentRefExists(root, target string) bool {
	if strings.HasPrefix(target, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		target = filepath.Join(home, target[2:])
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	_, err := os.Stat(target)
	return err == nil
}
