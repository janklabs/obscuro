package crypto

import (
	"encoding/base64"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	key := DeriveKey("testpassword", salt)
	plaintext := []byte("hello world")

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	salt, _ := GenerateSalt()
	key1 := DeriveKey("password1", salt)
	key2 := DeriveKey("password2", salt)

	encrypted, err := Encrypt(key1, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(key2, encrypted)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong key")
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)

	e1, _ := Encrypt(key, []byte("same"))
	e2, _ := Encrypt(key, []byte("same"))

	if e1 == e2 {
		t.Fatal("identical plaintexts should produce different ciphertexts due to random nonce")
	}
}

func TestVerificationToken(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("mypassword", salt)

	token, err := CreateVerificationToken(key)
	if err != nil {
		t.Fatal(err)
	}

	if !VerifyKey(key, token) {
		t.Fatal("expected verification to succeed with correct key")
	}
}

func TestVerificationTokenWrongKey(t *testing.T) {
	salt, _ := GenerateSalt()
	key1 := DeriveKey("correct", salt)
	key2 := DeriveKey("wrong", salt)

	token, _ := CreateVerificationToken(key1)

	if VerifyKey(key2, token) {
		t.Fatal("expected verification to fail with wrong key")
	}
}

func TestGenerateSaltUniqueness(t *testing.T) {
	s1, _ := GenerateSalt()
	s2, _ := GenerateSalt()

	if string(s1) == string(s2) {
		t.Fatal("two salts should not be identical")
	}
}

func TestDeriveKeyDifferentPasswords(t *testing.T) {
	salt, _ := GenerateSalt()
	key1 := DeriveKey("password1", salt)
	key2 := DeriveKey("password2", salt)
	if string(key1) == string(key2) {
		t.Fatal("different passwords should produce different keys")
	}
}

func TestEncryptOutputSize(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)
	plaintext := []byte("test message")

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatal(err)
	}

	expectedSize := NonceSize + len(plaintext) + 16
	if len(decoded) != expectedSize {
		t.Fatalf("expected decoded size %d, got %d", expectedSize, len(decoded))
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)

	_, err := Decrypt(key, "not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error decrypting invalid base64")
	}
}

func TestDecryptTooShort(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)

	shortData := base64.StdEncoding.EncodeToString(make([]byte, NonceSize-1))
	_, err := Decrypt(key, shortData)
	if err == nil {
		t.Fatal("expected error for ciphertext too short")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)
	plaintext := []byte("secret message")

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) > NonceSize {
		data[NonceSize] ^= 0xFF
	}

	tampered := base64.StdEncoding.EncodeToString(data)
	_, err = Decrypt(key, tampered)
	if err == nil {
		t.Fatal("expected error decrypting tampered ciphertext")
	}
}

func TestVerifyKeyMalformedEmpty(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)

	result := VerifyKey(key, "")
	if result {
		t.Fatal("expected VerifyKey to return false for empty token")
	}
}

func TestVerifyKeyMalformedInvalidBase64(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)

	result := VerifyKey(key, "!!!notbase64")
	if result {
		t.Fatal("expected VerifyKey to return false for invalid base64")
	}
}

func TestVerifyKeyMalformedTooShort(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)

	shortToken := base64.StdEncoding.EncodeToString(make([]byte, NonceSize-1))
	result := VerifyKey(key, shortToken)
	if result {
		t.Fatal("expected VerifyKey to return false for token too short")
	}
}

func TestVerifyKeyTamperedToken(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)

	token, err := CreateVerificationToken(key)
	if err != nil {
		t.Fatal(err)
	}

	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) > NonceSize {
		data[NonceSize] ^= 0xFF
	}

	tampered := base64.StdEncoding.EncodeToString(data)
	result := VerifyKey(key, tampered)
	if result {
		t.Fatal("expected VerifyKey to return false for tampered token")
	}
}

func TestBinaryRoundtrip(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)

	plaintext := []byte{0, 1, 2, 255, 0}

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatal(err)
	}

	if len(decrypted) != len(plaintext) {
		t.Fatalf("expected %d bytes, got %d", len(plaintext), len(decrypted))
	}

	for i, b := range plaintext {
		if decrypted[i] != b {
			t.Fatalf("byte mismatch at index %d: expected %d, got %d", i, b, decrypted[i])
		}
	}
}

func TestEncryptWithInvalidKeySize(t *testing.T) {
	invalidKey := []byte("tooshort")
	_, err := Encrypt(invalidKey, []byte("test"))
	if err == nil {
		t.Fatal("expected error with invalid key size")
	}
}

func TestDecryptWithInvalidKeySize(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("password", salt)
	plaintext := []byte("test")
	encrypted, _ := Encrypt(key, plaintext)

	invalidKey := []byte("tooshort")
	_, err := Decrypt(invalidKey, encrypted)
	if err == nil {
		t.Fatal("expected error with invalid key size")
	}
}
