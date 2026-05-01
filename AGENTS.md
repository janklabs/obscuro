# AGENTS.md

## What this is

Go CLI tool that encrypts secrets (Argon2id + AES-256-GCM) and stores them in `.obscuro/`. Primary use case: Helm post-renderer that replaces `__KEY__` placeholders via stdin/stdout.

## Build

```bash
go build -ldflags "-X github.com/janklabs/obscuro/internal/version.Version=$(git describe --tags --always)" -o obscuro .
```

The `-ldflags` version injection is required — without it `obscuro version` prints empty.

## Test

```bash
go test ./...
```

Tests need `git` available — each test creates a temp dir and runs `git init` because `store.RepoRoot()` walks up to find the repo root.

## CI checks (all must pass)

1. `gofmt -l .` — no unformatted files
2. `go build ./...`
3. `go test ./...`

Run `gofmt -w .` before committing.

## Release

Automated via `go-semantic-release` on main after CI passes. Use [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, etc.) for version bumps.

## Structure

```
main.go          → entrypoint, calls cmd.Execute()
cmd/             → cobra commands (one file per command + cmd_test.go)
internal/crypto/ → Argon2id key derivation + AES-256-GCM encrypt/decrypt
internal/store/  → .obscuro/ file I/O (config.json, secrets.json), repo root detection
internal/keychain/ → OS keychain integration (macOS Keychain, Linux Secret Service)
internal/version/  → version string (injected at build time)
```

## Conventions

- CLI uses `github.com/spf13/cobra`
- `cmd.Stdout` is an `io.Writer` abstraction (not raw `os.Stdout`) so tests can capture output
- Password resolution order: `--password` flag → OS keychain → `OBSCURO_PASSWORD` env var → interactive prompt
- `.obscuro/` directory is always at the git repo root (found via `store.RepoRoot()`)
