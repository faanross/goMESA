package common

import (
	"bytes"
	"testing"
)

func TestXOREncrypt(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		key      byte
		expected []byte
	}{
		{
			name:     "Empty data",
			data:     []byte{},
			key:      0x2E, // '.'
			expected: []byte{},
		},
		{
			name:     "Single byte",
			data:     []byte{0x41}, // 'A'
			key:      0x2E,         // '.'
			expected: []byte{0x6F}, // 'A' XOR '.'
		},
		{
			name:     "Multiple bytes",
			data:     []byte("Hello, World!"),
			key:      0x2E, // '.'
			expected: []byte{0x76, 0x66, 0x7E, 0x7E, 0x7D, 0x5A, 0x2E, 0x58, 0x7D, 0x73, 0x7E, 0x66, 0x43},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := XOREncrypt(tc.data, tc.key)
			if !bytes.Equal(result, tc.expected) {
				t.Errorf("XOREncrypt(%v, %v) = %v, expected %v", tc.data, tc.key, result, tc.expected)
			}

			// Test that decryption reverses encryption
			decrypted := XORDecrypt(result, tc.key)
			if !bytes.Equal(decrypted, tc.data) {
				t.Errorf("XORDecrypt(XOREncrypt(%v, %v), %v) = %v, expected %v",
					tc.data, tc.key, tc.key, decrypted, tc.data)
			}
		})
	}
}

func TestAESEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		key  []byte
	}{
		{
			name: "Empty data",
			data: []byte{},
			key:  make([]byte, 32),
		},
		{
			name: "Short text",
			data: []byte("Hello, World!"),
			key:  bytes.Repeat([]byte("key"), 8)[:32],
		},
		{
			name: "Longer text",
			data: []byte("This is a longer message that will be encrypted with AES-256-GCM."),
			key:  bytes.Repeat([]byte("secure-key"), 4)[:32],
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt the data
			encrypted, err := AESEncrypt(tc.data, tc.key)
			if err != nil {
				t.Fatalf("AESEncrypt failed: %v", err)
			}

			// Decrypt the data
			decrypted, err := AESDecrypt(encrypted, tc.key)
			if err != nil {
				t.Fatalf("AESDecrypt failed: %v", err)
			}

			// Check if the decrypted data matches the original
			if !bytes.Equal(decrypted, tc.data) {
				t.Errorf("AESDecrypt(AESEncrypt(%v, %v), %v) = %v, expected %v",
					tc.data, tc.key, tc.key, decrypted, tc.data)
			}
		})
	}
}

func TestGenerateAESKey(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey failed: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("GenerateAESKey() returned key of length %d, expected 32", len(key))
	}

	// Generate a second key to ensure they're different
	key2, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("Second GenerateAESKey failed: %v", err)
	}

	if bytes.Equal(key, key2) {
		t.Errorf("Generated keys should be different, but both are %v", key)
	}
}

func TestDerive32ByteKey(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "Empty password",
			password: "",
		},
		{
			name:     "Short password",
			password: "secret",
		},
		{
			name:     "Long password",
			password: "this is a very long password that exceeds 32 bytes in length",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key := Derive32ByteKey(tc.password)

			if len(key) != 32 {
				t.Errorf("Derive32ByteKey(%q) returned key of length %d, expected 32",
					tc.password, len(key))
			}

			// Same password should give same key
			key2 := Derive32ByteKey(tc.password)
			if !bytes.Equal(key, key2) {
				t.Errorf("Derive32ByteKey(%q) returned different keys: %v and %v",
					tc.password, key, key2)
			}

			// Different password should give different key
			if tc.password != "" {
				differentKey := Derive32ByteKey(tc.password + "different")
				if bytes.Equal(key, differentKey) {
					t.Errorf("Different passwords should produce different keys")
				}
			}
		})
	}
}
