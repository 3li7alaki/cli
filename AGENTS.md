# cli

Scaffolds new Go CLIs from the canonical layout, and lints existing ones back onto it.

    cli new <name> [--dir <path>] [--force] [--json]
    cli check [path] [--json]

`new` is the cheap half. `check` is the load-bearing half: a template only stops drift at
creation time, and every drift observed so far happened afterwards.

## One copy of the canonical files

`cli/internal/scaffold/template/` is the only copy that may exist. It is the tree
`new` writes and the tree `check` compares against, reached through
`scaffold.Canonical(rel, name)`. `check` holds no workflows of its own.

Never add a second copy, not under `check`, not at the repo root, not as a GitHub
template repo. Three copies existed once and that is the exact failure this tool sells
itself as preventing. `go:embed` cannot reach a parent directory, which is why the
single copy lives under `scaffold` rather than at the root.

`cli/go.mod` inside the template is stored as `go.mod.tmpl` because an embedded
`go.mod` would create a nested module boundary. `New` strips the suffix when writing.

## Rules

- No third-party dependencies. `cli/go.mod` must stay free of `require`. Stdlib `flag`
  plus a switch on `os.Args` is the whole command layer, and it is enough.
- Every command takes `--json`. Human output is never the integration protocol.
- A new `check` rule needs a real bug behind it. The five that exist each catch one that
  was observed in a real repo, not one that seemed plausible.
- Releases roll: every push to main republishes the single `latest` release.
- Never write an em dash or an en dash, anywhere in this repo. `check` fails on one.
- This repo is scaffolded by itself. `cli check .` must pass here too.

## Gates

    cd cli && gofmt -l . && go vet ./... && go build ./... && go test ./... -count=1
    go run ./cmd/cli check ..
