// Package crypto encrypts router API credentials at rest with AES-256-GCM,
// per the app-level field encryption design in project memory: the master
// key comes from an env var, never the database, so a stolen DB file alone
// is not enough to recover stored credentials.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// KeySize is the required raw key length: AES-256 uses a 32-byte key.
const KeySize = 32

var ErrInvalidKeySize = fmt.Errorf("encryption key must be %d bytes", KeySize)

// Cipher encrypts and decrypts values with AES-256-GCM under one key.
type Cipher struct {
	gcm cipher.AEAD
}

// NewCipher builds a Cipher from a raw key. len(key) must be KeySize.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return &Cipher{gcm: gcm}, nil
}

// NewCipherFromBase64Key decodes a base64-encoded key, as read from an env
// var, and builds a Cipher.
func NewCipherFromBase64Key(encoded string) (*Cipher, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	return NewCipher(key)
}

// Encrypt returns ciphertext and the nonce used to produce it. Both must be
// stored; Decrypt needs the nonce back to reverse it.
func (c *Cipher) Encrypt(plaintext string) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext = c.gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return ciphertext, nonce, nil
}

// Decrypt reverses Encrypt. It fails if ciphertext, nonce, or the key don't
// match what Encrypt produced — GCM authenticates the data, so tampering or
// using the wrong key is detected rather than silently returning garbage.
func (c *Cipher) Decrypt(ciphertext, nonce []byte) (string, error) {
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("decrypt: authentication failed (wrong key, or data was tampered with)")
	}
	return string(plaintext), nil
}

// GenerateKey returns a new random key, base64-encoded, suitable for
// WDSPLIT_ENCRYPTION_KEY.
func GenerateKey() (string, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
