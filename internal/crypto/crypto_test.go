package crypto

import (
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
