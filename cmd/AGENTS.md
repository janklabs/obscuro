# cmd/

Cobra command layer. One file per verb. ~1300 LOC, 12 files.

## Commands

| File        | Verb                      | Notes                                                                 |
| ----------- | ------------------------- | --------------------------------------------------------------------- |
| root.go     | `obscuro`                 | Root + `authenticate()` + `getPassword()` + background update check   |
| init.go     | `init`                    | Creates `.obscuro/`, writes salt + verification token                 |
| set.go      | `set KEY`                 | Encrypt + store; supports `--value-file -` (stdin)                    |
| get.go      | `get KEY`                 | Decrypt one secret to `cmd.Stdout`                                    |
| list.go     | `list`                    | Names only, **no password required** (reads `secrets.json` keys)      |
| edit.go     | `edit KEY`                | Spawns `$EDITOR` on a tempfile, re-encrypts on close                  |
| remove.go   | `remove KEY`              | Deletes from secrets map                                              |
| inject.go   | `inject`                  | Reads stdin, replaces `__NAME__` placeholders, writes to `cmd.Stdout` |
| auth.go     | `auth store/clear/status` | Manage keychain entry (keyed by base64 salt)                          |
| upgrade.go  | `upgrade`                 | Self-update from GitHub Releases; verifies sha256, atomic rename      |
| version.go  | `version`                 | Prints `internal/version.Version`                                     |
| cmd_test.go | —                         | Integration tests; drives `rootCmd` with captured `Stdout`/`Stderr`   |

## Adding a command

1. New file `cmd/<verb>.go`, `package cmd`.
2. Declare `var <verb>Cmd = &cobra.Command{ Use: "...", RunE: ... }`.
3. Register in that file's `init()`: `rootCmd.AddCommand(<verb>Cmd)` (NOT in `root.go`).
4. Use `RunE` (not `Run`) so errors propagate; cobra exits with code 1 via `Execute()`.
5. Payload → `Stdout` (the package var). Human/diagnostic text → `os.Stderr` or `cmd.ErrOrStderr()`.
6. To get a key, call `authenticate()` — it handles config load, password prompt, verification token check.
7. Add a test in `cmd_test.go` using `setup(t)` + `execCmd(t, "verb", "args"...)`.

## Conventions specific to this package

- `Stdout io.Writer = os.Stdout` (root.go:25) is the **only** sink for command payload. Tests reassign it; never write payload to `os.Stdout` directly.
- Persistent flags `--password` / `--password-file` live on `rootCmd` and are read by `getPassword()`.
- Skip the update check by name in `PersistentPreRun` if your command shouldn't trigger it (currently exempt: `upgrade`, `version`).
- Long-running prompts read from `openTTY()` (a helper that opens `/dev/tty` directly) so they work even when stdin is piped (`obscuro inject` case).
- `secretValue` is a package-level var reset in `setup(t)` — if you add another package-level flag var, reset it there too or tests will leak state.

## Anti-patterns

- **Do not** call `os.Exit` inside a command — return an error from `RunE`.
- **Do not** print errors to `Stdout`; cobra prints `RunE` errors to stderr automatically.
- **Do not** make the update check or `fetchChangelog` blocking/fatal (upgrade.go:204).
- **Do not** add a command that decrypts without going through `authenticate()` — bypasses the verification-token gate.
- **Do not** read CWD-relative paths; everything goes through `store.Dir()`.
