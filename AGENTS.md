# AGENTS.md

**Generated:** 2026-05-09 · **Commit:** ba5b65a · **Branch:** main

## What this is

Go 1.25 CLI (`cobra`) that encrypts secrets (Argon2id + AES-256-GCM) into `.obscuro/secrets.json` at the **git repo root**. Primary use: Helm post-renderer replacing `__KEY__` placeholders via stdin/stdout (`obscuro inject`). Self-upgrades from GitHub Releases. Bundled Next.js docs site lives in `website/`.

## Structure

```
main.go              → 7-line entrypoint → cmd.Execute()
cmd/                 → cobra commands, one file per verb + cmd_test.go (see cmd/AGENTS.md)
internal/crypto/     → Argon2id KDF + AES-256-GCM (see internal/crypto/AGENTS.md)
internal/store/      → .obscuro/ I/O + git-repo-root detection (see internal/store/AGENTS.md)
internal/keychain/   → 28-line wrapper around zalando/go-keyring; key=salt, service="obscuro"
internal/version/    → 1 var, set at link time via -ldflags (default "dev")
.github/workflows/   → ci.yml (fmt/build/test) → release.yml (semantic-release + cross-compile) → website.yml
install.sh           → curl|sh installer; downloads release asset, verifies sha256, installs to ~/.local/bin
website/             → Next.js 16 + fumadocs docs site (its own AGENTS.md applies inside)
```

## Where to look

| Task                               | Location                                           |
| ---------------------------------- | -------------------------------------------------- |
| Add a new CLI command              | `cmd/<verb>.go` + register in its `init()`         |
| Change password resolution         | `cmd/root.go::getPassword`                         |
| Change on-disk format              | `internal/store/store.go` (`Config`, `Secrets`)    |
| Tune Argon2 params                 | `internal/crypto/crypto.go` constants              |
| Add a release asset / target       | `.github/workflows/release.yml` `targets=(...)`    |
| Modify update-check / self-upgrade | `cmd/upgrade.go` + `cmd/root.go::PersistentPreRun` |
| Docs content                       | `website/content/docs/*.mdx`                       |

## Build

```bash
go build -ldflags "-X github.com/janklabs/obscuro/internal/version.Version=$(git describe --tags --always)" -o obscuro .
```

The `-ldflags` injection is **required** — without it `obscuro version` prints empty AND `PersistentPreRun` skips the update check (treats `version=="dev"` as a dev build).

## Test

```bash
go test ./...
```

Every test calls `t.TempDir()` → `os.Chdir` → `git init` → `store.ResetRoot()`. **`git` must be on PATH** — `store.RepoRoot()` shells out to `git rev-parse --show-toplevel` and caches via `sync.Once`. Forgetting `ResetRoot()` between tests leaks state across cases.

## CI checks (all must pass)

1. `gofmt -l .` — must be empty (run `gofmt -w .` before commit)
2. `go build ./...`
3. `go test ./...`

Website is built by a separate workflow and is excluded from the release path-filter (changes under `website/` do not bump the CLI version).

## Release

`go-semantic-release` on `main` after CI passes. Conventional Commits drive version bumps (`feat:` minor, `fix:` patch, `feat!:` major). Release workflow then cross-compiles 6 targets (linux/darwin/windows × amd64/arm64) with `CGO_ENABLED=0`, generates `checksums.txt`, and uploads to the release. `install.sh` and `cmd/upgrade.go` both depend on this asset naming: `obscuro-${VERSION}-${GOOS}-${GOARCH}[.exe]`.

## Code map (top symbols)

| Symbol                       | File                         | Role                                                   |
| ---------------------------- | ---------------------------- | ------------------------------------------------------ |
| `cmd.Execute`                | cmd/root.go                  | Sole exported entrypoint from `main`                   |
| `cmd.Stdout` (`io.Writer`)   | cmd/root.go:25               | **Always write payload here, not `os.Stdout`** (tests) |
| `cmd.authenticate`           | cmd/root.go                  | Resolves key via the password order below              |
| `store.RepoRoot`             | internal/store/store.go:30   | `git rev-parse`-cached; **all paths derive from this** |
| `store.ResetRoot`            | internal/store/store.go:43   | Test-only; clears the `sync.Once`                      |
| `crypto.DeriveKey`           | internal/crypto/crypto.go:36 | Argon2id(t=3, mem=64MiB, lanes=4) → 32-byte key        |
| `crypto.Encrypt` / `Decrypt` | internal/crypto/crypto.go    | AES-256-GCM; output is `base64(nonce‖ct)`              |
| `crypto.VerifyKey`           | internal/crypto/crypto.go    | Decrypts the verification token to gate password       |

## Conventions

- Cobra: one command per file in `cmd/`, registered in that file's `init()` via `rootCmd.AddCommand`.
- Payload (the data the user asked for, e.g. `obscuro get`) → `cmd.Stdout`. Human messages → `os.Stderr`. Never mix.
- Password resolution order (do not reorder):
  `--password` flag → `--password-file` → OS keychain (keyed by salt) → `OBSCURO_PASSWORD` env → interactive TTY prompt.
- `.obscuro/` is always at the git repo root; never resolve relative to CWD.
- Keychain entries are keyed by the **base64 salt string** (not a fixed key), so re-`init` invalidates old entries automatically.
- Errors returned from `RunE` are surfaced by cobra; do not call `os.Exit` from commands.

## Anti-patterns (forbidden here)

- **Do not** commit plaintext secrets or decrypted `obscuro inject` output (see `website/content/docs/guides/{docker-compose,kubernetes}.mdx`).
- **Do not** make `fetchChangelog` or the update-check fatal — they must always degrade silently (`cmd/upgrade.go:204`, `cmd/root.go` 2 s timeout).
- **Do not** weaken Argon2 parameters or shrink `SaltSize`/`NonceSize` without a migration path — old vaults must still decrypt.
- **Do not** read raw `os.Stdout` in commands; use the `cmd.Stdout` indirection.
- **Do not** assume CWD == repo root; always go through `store.RepoRoot()` / `store.Dir()`.
- **Do not** add CGO. Release pipeline builds with `CGO_ENABLED=0` for static cross-platform binaries.

## Commands

```bash
gofmt -w .                         # required pre-commit
go test ./...                      # full test suite
go build -ldflags "-X github.com/janklabs/obscuro/internal/version.Version=$(git describe --tags --always)" -o obscuro .
OBSCURO_NO_UPDATE_CHECK=1 ./obscuro …   # silence the background self-update probe
pnpm --dir website dev             # docs site (Next.js 16 — see website/AGENTS.md)
```

## Notes

- `cmd/root.go` fires a goroutine in `PersistentPreRun` to check GitHub for a newer tag; result is consumed in `PersistentPostRun` with a 2 s timeout. Set `OBSCURO_NO_UPDATE_CHECK=1` in CI/scripts to skip.
- `inject` decrypts **all** secrets up-front (single password prompt) then does string replacement; placeholder format is exactly `__NAME__` (double underscore, no padding).
- Module path is `github.com/janklabs/obscuro` — keep `-ldflags -X` target in sync if it ever moves.
