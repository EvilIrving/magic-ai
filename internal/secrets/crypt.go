package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const pbkdfIter = 100000

// Encrypt encrypts plaintext with passphrase (AES-GCM + PBKDF2).
func Encrypt(plaintext []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	key := pbkdf2.Key([]byte(passphrase), salt, pbkdfIter, 32, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := gcm.Seal(nonce, nonce, plaintext, nil)
	// salt || ciphertext
	return append(salt, out...), nil
}

// Decrypt reverses Encrypt.
func Decrypt(blob []byte, passphrase string) ([]byte, error) {
	if len(blob) < 16 {
		return nil, fmt.Errorf("ciphertext too short")
	}
	salt, rest := blob[:16], blob[16:]
	key := pbkdf2.Key([]byte(passphrase), salt, pbkdfIter, 32, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(rest) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := rest[:gcm.NonceSize()], rest[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}
