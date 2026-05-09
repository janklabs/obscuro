# internal/crypto/

Argon2id KDF + AES-256-GCM. 96 LOC + 91 LOC test. Security-critical — change with care.

## Constants (do not alter without migration)

| Const        | Value  | Why                                       |
| ------------ | ------ | ----------------------------------------- |
| `SaltSize`   | 16 B   | Per-vault, stored b64 in `config.json`    |
| `KeySize`    | 32 B   | AES-256                                   |
| `NonceSize`  | 12 B   | GCM standard; **prepended to ciphertext** |
| `ArgonTime`  | 3      | Argon2id iterations                       |
| `ArgonMem`   | 64 MiB | Argon2id memory cost                      |
| `ArgonLanes` | 4      | Argon2id parallelism                      |

Changing any of these breaks decryption of existing vaults. Add a versioned config field first.

## Wire format

`Encrypt` returns `base64.StdEncoding(nonce ‖ ciphertext_with_tag)`. `Decrypt` reverses it. The nonce is generated fresh per call from `crypto/rand`; **never reuse a nonce with the same key**.

## Verification token

`CreateVerificationToken(key)` encrypts the constant `"obscuro-verify"`. Stored in `config.json` and checked by `VerifyKey` (uses `gcm.Open` failure as the signal — constant-time per Go stdlib). This is how a wrong password is detected before touching `secrets.json`.

## Anti-patterns

- **Do not** call `Encrypt` with a nonce you supply — it's generated internally for a reason.
- **Do not** swap to `crypto/sha256`-based KDFs; password-derived keys must be Argon2id.
- **Do not** log or return the derived key; pass `[]byte` and let it go out of scope.
- **Do not** use `bytes.Equal` for token comparison anywhere downstream — `subtle.ConstantTimeCompare` only (already imported).
