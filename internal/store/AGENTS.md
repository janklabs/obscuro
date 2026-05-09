# internal/store/

On-disk layout + git-repo-root resolution. 202 LOC + 131 LOC test.

## Files written under `<repoRoot>/.obscuro/` (mode 0700)

| File           | Schema                                                           |
| -------------- | ---------------------------------------------------------------- |
| `config.json`  | `{ "salt": <b64>, "verification_token": <b64(nonce‖ct)> }`       |
| `secrets.json` | `{ "<NAME>": <b64(nonce‖ct)>, ... }` — flat map, sorted on write |

Both are written atomically (temp file + `os.Rename`). Mode is enforced on the directory; individual files inherit umask — callers must not write here directly.

## RepoRoot

`RepoRoot()` shells out to `git rev-parse --show-toplevel`, caches via `sync.Once`. Failure returns the _exact_ error string `"not inside a git repository — obscuro must be run within a git repo"` — `cmd/` matches user-facing wording on this. **Tests must call `ResetRoot()` after `os.Chdir`** or they will see another test's repo.

## Conventions

- All paths resolve through `Dir()` (which calls `RepoRoot()`); never join paths against CWD.
- Sort secret names on write so JSON diffs are stable.
- `LoadSecrets` returns `map[string]string` of base64 ciphertexts — decryption is the caller's job.
- `Init` is idempotent in spirit but will overwrite `config.json` if called twice; `cmd/init.go` checks `IsInitialized()` first.

## Anti-patterns

- **Do not** add fields to `Config` without bumping a version field first — old binaries must still read it.
- **Do not** export raw paths; expose them through `Dir()`/helpers so the cache stays authoritative.
- **Do not** use `ioutil.*` (deprecated); `os` + `encoding/json` only.
- **Do not** widen `0o700` directory perms — secrets file lives here.
