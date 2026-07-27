package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

var (
	ErrMasterKeyNotSet = errors.New("MASTER_KEY is not set")
	ErrInvalidKey      = errors.New("invalid master key length for AES-256")
	ErrCipher          = errors.New("encryption/decryption error")
)

func getMasterKey() ([]byte, error) {
	keyStr := os.Getenv("MASTER_KEY")
	if keyStr == "" {
		return nil, ErrMasterKeyNotSet
	}

	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, ErrInvalidKey
	}

	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	return key, nil
}

// Encrypt encrypts plain text using AES-GCM
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := getMasterKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts cipher text using AES-GCM
func Decrypt(ciphertextHex string) (string, error) {
	if ciphertextHex == "" {
		return "", nil
	}

	key, err := getMasterKey()
	if err != nil {
		return "", err
	}

	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", ErrCipher
	}

	nonce, ciphertextBytes := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintextBytes, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintextBytes), nil
}
