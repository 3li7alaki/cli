package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cli/internal/check"
	"cli/internal/scaffold"
	"cli/internal/version"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: cli new <name> [--dir <path>] [--force] [--json] | cli check [path] [--json]")
		return 2
	}
	if args[0] == "--version" || args[0] == "version" {
		fmt.Fprintln(stdout, version.Version)
		return 0
	}
	switch args[0] {
	case "new":
		return runNew(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return 2
	}
}

func runNew(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "", "destination directory")
	force := fs.Bool("force", false, "write into a non-empty directory")
	asJSON := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(flagsFirst(args, map[string]bool{"--dir": true})); err != nil || fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: cli new <name> [--dir <path>] [--force] [--json]")
		return 2
	}
	name := fs.Arg(0)
	if *dir == "" {
		*dir = name
	}
	if err := scaffold.New(name, *dir, *force); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		_ = json.NewEncoder(stdout).Encode(map[string]string{"name": name, "dir": *dir})
	} else {
		fmt.Fprintf(stdout, "created %s in %s\n", name, *dir)
	}
	return 0
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(flagsFirst(args, nil)); err != nil || fs.NArg() > 1 {
		fmt.Fprintln(stderr, "usage: cli check [path] [--json]")
		return 2
	}
	root := "."
	if fs.NArg() == 1 {
		root = fs.Arg(0)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	findings := check.Run(root)
	if *asJSON {
		_ = json.NewEncoder(stdout).Encode(findings)
	} else {
		for _, f := range findings {
			fmt.Fprintf(stdout, "%s: %s. Fix: %s\n", f.Path, f.Problem, f.Fix)
		}
		if len(findings) == 0 {
			fmt.Fprintln(stdout, "clean")
		}
	}
	if len(findings) > 0 {
		return 1
	}
	return 0
}

func flagsFirst(args []string, values map[string]bool) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		if len(args[i]) > 0 && args[i][0] == '-' {
			flags = append(flags, args[i])
			if values[args[i]] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return append(flags, positional...)
}
