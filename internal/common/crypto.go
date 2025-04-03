package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// XOREncrypt performs simple XOR encryption/decryption on data
// Using a single byte key, as per the original specification
func XOREncrypt(data []byte, key byte) []byte {
	encrypted := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		encrypted[i] = data[i] ^ key
	}
	return encrypted
}

// XORDecrypt performs simple XOR decryption on data
// Since XOR is symmetric, this is the same as XOREncrypt
func XORDecrypt(data []byte, key byte) []byte {
	return XOREncrypt(data, key)
}

// AESEncrypt encrypts data with AES-256-GCM
// This is an enhancement over the original implementation
func AESEncrypt(plaintext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESDecrypt decrypts data with AES-256-GCM
func AESDecrypt(ciphertext string, key []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(data) < aesGCM.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, cipherData := data[:aesGCM.NonceSize()], data[aesGCM.NonceSize():]
	plaintext, err := aesGCM.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// GenerateAESKey generates a 32-byte key for AES-256
func GenerateAESKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// Derive32ByteKey derives a 32-byte key from a password
func Derive32ByteKey(password string) []byte {
	// Simple key derivation for demonstration
	// In a production system, use a proper KDF like PBKDF2, Argon2, etc.
	key := make([]byte, 32)
	for i := 0; i < len(password); i++ {
		key[i%32] ^= password[i]
	}
	return key
}
