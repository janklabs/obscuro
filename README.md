# Obscuro

[![CI](https://github.com/janklabs/obscuro/actions/workflows/ci.yml/badge.svg)](https://github.com/janklabs/obscuro/actions/workflows/ci.yml)

Safely store encrypted secrets in your repository. Obscuro encrypts values with a password-derived key (Argon2id + AES-256-GCM) and stores them in `.obscuro/secrets.json`. Secrets are injected into templates by replacing `__KEY__` placeholders via stdin/stdout — designed to work as a Helm post-renderer.

## Installation

### One-liner

```bash
curl -sSL https://raw.githubusercontent.com/janklabs/obscuro/main/install.sh | bash
```

This downloads the latest prebuilt binary for your OS/architecture from [GitHub Releases](https://github.com/janklabs/obscuro/releases), verifies its SHA-256 checksum, installs it to `~/.local/bin`, and optionally adds it to your `PATH`. No Go toolchain required.

Supported platforms: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64, arm64).

To install a specific version or to a custom directory:

```bash
curl -sSL https://raw.githubusercontent.com/janklabs/obscuro/main/install.sh | \
  OBSCURO_VERSION=v1.7.11 OBSCURO_INSTALL_DIR=/usr/local/bin bash
```

### Manual download

Grab the appropriate archive from [Releases](https://github.com/janklabs/obscuro/releases/latest), verify it against `checksums.txt`, then move it onto your `PATH`:

```bash
mv obscuro-v1.7.11-linux-amd64 ~/.local/bin/obscuro
chmod +x ~/.local/bin/obscuro
```

### From source

Requires [Go 1.21+](https://go.dev/dl).

```bash
git clone https://github.com/janklabs/obscuro.git
cd obscuro
go build -ldflags "-X github.com/janklabs/obscuro/internal/version.Version=$(git describe --tags --always)" -o obscuro .
mv obscuro ~/.local/bin/  # or anywhere on your PATH
```

## Quick start

```bash
# Initialize with a master password
obscuro init

# Store secrets
obscuro set API_KEY
obscuro set DB_PASS

# Retrieve a secret
obscuro get API_KEY

# List all secret names
obscuro list
```

## Usage

### `obscuro init`

Creates the `.obscuro/` directory and sets up encryption. Prompts for a master password (with confirmation).

### `obscuro set KEY`

Encrypts and stores a secret. Prompts for the value interactively.

```bash
obscuro set API_KEY
# or non-interactively (from files)
obscuro set API_KEY --password-file ./pw.txt --value-file ./secret.txt
```

### `obscuro get KEY`

Decrypts and prints a secret value to stdout.

```bash
obscuro get API_KEY --password mypass
```

### `obscuro list`

Lists all stored secret names (no password required).

### `obscuro inject`

Reads stdin, replaces all `__KEY__` placeholders with decrypted values, writes to stdout.

```bash
echo "token: __API_KEY__" | obscuro inject --password mypass
# Output: token: my-secret
```

### `obscuro version`

Prints the current version.

```bash
obscuro version
# Output: obscuro v1.2.0
```

### `obscuro upgrade`

Upgrades to the latest release. Downloads the matching prebuilt binary from GitHub, verifies its SHA-256 checksum, and atomically replaces the current binary. No Go toolchain required.

```bash
obscuro upgrade
```

### `obscuro auth`

Manage OS keychain password storage.

```bash
obscuro auth store    # Verify and store password in OS keychain
obscuro auth clear    # Remove password from keychain
obscuro auth status   # Check if keychain has a stored password
```

### Flags

| Flag | Short | Scope | Description |
|------|-------|-------|-------------|
| `--password` | `-p` | All commands | Master password (skips interactive prompt) |
| `--password-file` | | All commands | Read master password from file (use `-` for stdin) |
| `--value` | | `set` only | Secret value (skips interactive prompt) |
| `--value-file` | | `set` only | Read secret value from file (use `-` for stdin) |

> **Security note:** `--password` and `--value` pass secrets as command-line arguments, which are visible to other users on the system via `ps` or `/proc`. Prefer `--password-file`, `--value-file`, the OS keychain, or the `OBSCURO_PASSWORD` environment variable on shared systems.

## Password resolution

The master password is resolved in this order:

1. `--password` / `-p` flag
2. `--password-file` flag (reads from file, or stdin with `-`)
3. OS keychain (macOS Keychain, Linux Secret Service)
4. `OBSCURO_PASSWORD` environment variable
5. Interactive terminal prompt

During `obscuro init`, you'll be asked to store the password in the OS keychain. Once stored, all subsequent commands (including `inject` as a Helm post-renderer) authenticate automatically with no flags, env vars, or prompts needed.

For CI/headless environments where no keychain is available, use the environment variable:

```bash
export OBSCURO_PASSWORD=mypassword
helm install myrelease ./chart --post-renderer obscuro --post-renderer-args inject
```

## Helm post-renderer

Use `obscuro inject` as a Helm post-renderer to replace placeholders in rendered manifests with decrypted secrets.

In your templates, use placeholders:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
stringData:
  api-key: __API_KEY__
  db-password: __DB_PASS__
```

Then deploy with:

```bash
helm install myrelease ./chart --post-renderer obscuro --post-renderer-args inject
```

## What gets committed

Both files in `.obscuro/` are safe to commit:

- **`config.json`** — Argon2id salt and a verification token. The salt is not secret.
- **`secrets.json`** — All values are AES-256-GCM encrypted with random nonces.

## How it works

1. **`init`** generates a random salt and derives a 256-bit key from your password using Argon2id. A verification token (encrypted known string) is stored so future commands can check if the password is correct.
2. **`set`** derives the key, verifies it, then encrypts the value with AES-256-GCM using a random nonce. The nonce is prepended to the ciphertext and stored as base64.
3. **`get`** / **`inject`** derive the key, verify it, then decrypt.

Identical values produce different ciphertexts each time due to random nonces.
