package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpAndVersionGoToStdout(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help", "-v", "--version", "version"} {
		var stdout, stderr bytes.Buffer
		if code := run([]string{arg}, &stdout, &stderr); code != 0 {
			t.Fatalf("%s: exit %d, stderr %q", arg, code, stderr.String())
		}
		if strings.TrimSpace(stdout.String()) == "" {
			t.Fatalf("%s: nothing on stdout", arg)
		}
	}
}

func TestNoArgsIsAUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}
