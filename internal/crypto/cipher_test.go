package crypto

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	ciphertext, nonce, err := c.Encrypt("s3cr3t-router-password")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := c.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != "s3cr3t-router-password" {
		t.Errorf("Decrypt() = %q, want %q", got, "s3cr3t-router-password")
	}
}

func TestNewCipherRejectsWrongKeySize(t *testing.T) {
	_, err := NewCipher(make([]byte, 16))
	if !errors.Is(err, ErrInvalidKeySize) {
		t.Errorf("NewCipher(16 bytes) error = %v, want ErrInvalidKeySize", err)
	}
}

func TestDecryptFailsWithWrongKey(t *testing.T) {
	key1 := bytes.Repeat([]byte{0x01}, KeySize)
	key2 := bytes.Repeat([]byte{0x02}, KeySize)

	c1, err := NewCipher(key1)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	c2, err := NewCipher(key2)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	ciphertext, nonce, err := c1.Encrypt("hunter2")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if _, err := c2.Decrypt(ciphertext, nonce); err == nil {
		t.Error("Decrypt() with wrong key succeeded, want error")
	}
}

func TestDecryptFailsOnTamperedCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x03}, KeySize)
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	ciphertext, nonce, err := c.Encrypt("hunter2")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	ciphertext[0] ^= 0xFF // flip a bit

	if _, err := c.Decrypt(ciphertext, nonce); err == nil {
		t.Error("Decrypt() of tampered ciphertext succeeded, want error")
	}
}

func TestEncryptProducesDistinctNoncesAndCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x04}, KeySize)
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	ciphertext1, nonce1, err := c.Encrypt("same-plaintext")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	ciphertext2, nonce2, err := c.Encrypt("same-plaintext")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if bytes.Equal(nonce1, nonce2) {
		t.Error("two Encrypt calls produced the same nonce")
	}
	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("two Encrypt calls of the same plaintext produced the same ciphertext")
	}
}

func TestNewCipherFromBase64Key(t *testing.T) {
	generated, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	c, err := NewCipherFromBase64Key(generated)
	if err != nil {
		t.Fatalf("NewCipherFromBase64Key: %v", err)
	}

	ciphertext, nonce, err := c.Encrypt("round-trip-check")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	got, err := c.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != "round-trip-check" {
		t.Errorf("Decrypt() = %q, want %q", got, "round-trip-check")
	}
}

func TestNewCipherFromBase64KeyRejectsInvalidBase64(t *testing.T) {
	_, err := NewCipherFromBase64Key("not-valid-base64!!!")
	if err == nil {
		t.Error("NewCipherFromBase64Key(invalid base64) error = nil, want error")
	}
}

func TestGenerateKeyProducesUniqueKeys(t *testing.T) {
	a, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	b, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if a == b {
		t.Error("two GenerateKey calls returned the same key")
	}
}
