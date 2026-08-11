# cli

Scaffolds new Go CLIs from one canonical layout, and checks existing ones still match it.

```sh
curl -fsSL https://raw.githubusercontent.com/3li7alaki/cli/main/install.sh | sh
```

## Use

```sh
cli new gauge              # write a new repo into ./gauge
cli new gauge --dir ~/dev/gauge
cli check                  # lint the repo in the current directory
cli check ~/dev/vision --json
```

`check` exits 1 with one line per finding, or 0 and `clean`. It reports:

- `AGENTS.md` missing, or `CLAUDE.md` that is not exactly `@AGENTS.md`
- a `require` directive in `cli/go.mod`, meaning a dependency crept in
- `.github/workflows/ci.yml` or `release.yml` drifted from canonical (the Go version is
  ignored, everything else is compared)
- an em dash or en dash anywhere in the repo, with the file and line
- release ldflags whose `-X <pkg>.Version` target does not exist, which Go accepts
  silently and which therefore ships a binary whose `--version` never updates

## Layout it produces

```
cli/cmd/<name>/main.go      the binary
cli/internal/version/       Version, stamped by release ldflags
install.sh                  no dependencies, no root, idempotent, --uninstall
.github/workflows/ci.yml    gofmt, vet, build, test
.github/workflows/release.yml   4 binaries, rolling `latest` release, no tags
AGENTS.md + CLAUDE.md       the pointer pair every harness reads
```

Stdlib only, in the tool and in what it scaffolds. No CLI framework.
