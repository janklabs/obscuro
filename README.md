# Obscuro

[![CI](https://github.com/janklabs/obscuro/actions/workflows/ci.yml/badge.svg)](https://github.com/janklabs/obscuro/actions/workflows/ci.yml)

Safely store encrypted secrets in your repository. Obscuro encrypts values with a password-derived key (Argon2id + AES-256-GCM) and stores them in `.obscuro/secrets.json`. Secrets are injected into templates by replacing `__KEY__` placeholders via stdin/stdout — designed to work as a Helm post-renderer.

## Installation

### One-liner

```bash
curl -sSL https://raw.githubusercontent.com/janklabs/obscuro/main/install.sh | bash
```

This clones the repo, builds the binary, installs it to `~/.local/bin`, and optionally adds it to your `PATH`.

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
# or non-interactively
obscuro set API_KEY --password mypass --value my-secret
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

Upgrades to the latest release. Fetches the latest tag from GitHub, builds from source in a temp directory, and replaces the current binary. Requires Go to be installed.

```bash
obscuro upgrade
```

### Flags

| Flag | Short | Scope | Description |
|------|-------|-------|-------------|
| `--password` | `-p` | All commands | Master password (skips interactive prompt) |
| `--value` | | `set` only | Secret value (skips interactive prompt) |

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
