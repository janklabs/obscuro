package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	SaltSize   = 16
	KeySize    = 32 // AES-256
	NonceSize  = 12 // GCM standard nonce
	ArgonTime  = 3
	ArgonMem   = 64 * 1024 // 64 MB
	ArgonLanes = 4

	verificationPlaintext = "obscuro-verify"
)

// GenerateSalt returns a random 16-byte salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}
	return salt, nil
}

// DeriveKey derives a 256-bit key from a password and salt using Argon2id.
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, ArgonTime, ArgonMem, ArgonLanes, KeySize)
}

// Encrypt encrypts plaintext with AES-256-GCM. Returns base64(nonce || ciphertext).
func Encrypt(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil) // prepends nonce
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64(nonce || ciphertext) string with AES-256-GCM.
func Decrypt(key []byte, encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}
	if len(data) < NonceSize {
		return nil, errors.New("ciphertext too short")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	nonce := data[:NonceSize]
	ciphertext := data[NonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}
	return plaintext, nil
}

// CreateVerificationToken encrypts a known string so we can later verify the password.
func CreateVerificationToken(key []byte) (string, error) {
	return Encrypt(key, []byte(verificationPlaintext))
}

// VerifyKey checks that the given key can decrypt the verification token.
func VerifyKey(key []byte, token string) bool {
	plaintext, err := Decrypt(key, token)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(plaintext, []byte(verificationPlaintext)) == 1
}
