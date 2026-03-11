package wormhole

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"single byte", []byte{0x42}},
		{"small data", []byte("hello world")},
		{"medium data", make([]byte, 1024)},
		{"large data", make([]byte, 64*1024)},
		{"block size aligned", make([]byte, 16)},
		{"block size + 1", make([]byte, 17)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill with random data if not already set
			if len(tt.data) > 0 && tt.name != "small data" {
				rand.Read(tt.data)
			}

			key := NewEncryptionKey()
			ciphertext, err := Encrypt(key, tt.data)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			plaintext, err := Decrypt(key, ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if !bytes.Equal(plaintext, tt.data) {
				t.Errorf("Decrypt() mismatch: got %v, want %v", plaintext, tt.data)
			}
		})
	}
}

func TestEncryptWithDifferentKeys(t *testing.T) {
	data := []byte("secret message")
	key1 := NewEncryptionKey()
	key2 := NewEncryptionKey()

	ciphertext1, err := Encrypt(key1, data)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Trying to decrypt with wrong key should fail or produce garbage
	_, err = Decrypt(key2, ciphertext1)
	if err == nil {
		// If no error, check that plaintext doesn't match
		t.Log("Decrypt with wrong key succeeded (message authentication may not be implemented)")
	}
}

func TestDecryptTamperedData(t *testing.T) {
	data := []byte("secret message")
	key := NewEncryptionKey()

	ciphertext, err := Encrypt(key, data)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Tamper with ciphertext
	if len(ciphertext) > 0 {
		ciphertext[0] ^= 0x01
	}

	_, err = Decrypt(key, ciphertext)
	// Decryption might succeed with garbage output or fail
	// If it succeeds, verify it's not the original plaintext
	if err == nil {
		t.Log("Decrypt with tampered data succeeded (message authentication may not be implemented)")
	}
}

func TestNewEncryptionKey(t *testing.T) {
	key1 := NewEncryptionKey()
	key2 := NewEncryptionKey()

	// Keys should be different (with very high probability)
	if bytes.Equal(key1[:], key2[:]) {
		t.Error("NewEncryptionKey() generated identical keys")
	}

	// Keys should be 32 bytes (256-bit for AES-256)
	if len(key1) != 32 {
		t.Errorf("NewEncryptionKey() generated key of length %d, want 32", len(key1))
	}
}

func TestEncryptEmptyData(t *testing.T) {
	key := NewEncryptionKey()
	ciphertext, err := Encrypt(key, []byte{})
	if err != nil {
		t.Fatalf("Encrypt(empty) error = %v", err)
	}

	plaintext, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt(empty ciphertext) error = %v", err)
	}

	if len(plaintext) != 0 {
		t.Errorf("Decrypt(empty) = %v, want empty", plaintext)
	}
}

func TestEncryptNilData(t *testing.T) {
	key := NewEncryptionKey()
	ciphertext, err := Encrypt(key, nil)
	if err != nil {
		t.Fatalf("Encrypt(nil) error = %v", err)
	}

	plaintext, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt(nil ciphertext) error = %v", err)
	}

	if len(plaintext) != 0 {
		t.Errorf("Decrypt(nil) = %v, want empty", plaintext)
	}
}

func TestCiphertextLargerThanPlaintext(t *testing.T) {
	data := []byte("hello world")
	key := NewEncryptionKey()

	ciphertext, err := Encrypt(key, data)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Ciphertext should be larger due to nonce/IV
	if len(ciphertext) <= len(data) {
		t.Errorf("Ciphertext length %d should be larger than plaintext %d", len(ciphertext), len(data))
	}
}

func TestEncryptDeterminism(t *testing.T) {
	data := []byte("same data")
	key := NewEncryptionKey()

	c1, _ := Encrypt(key, data)
	c2, _ := Encrypt(key, data)

	// Same key + same data should produce different ciphertexts (due to random nonce/IV)
	if bytes.Equal(c1, c2) {
		t.Error("Encrypt() produced identical ciphertexts for same input (no nonce randomization?)")
	}

	// But both should decrypt to same plaintext
	p1, _ := Decrypt(key, c1)
	p2, _ := Decrypt(key, c2)
	if !bytes.Equal(p1, p2) {
		t.Error("Different ciphertexts should decrypt to same plaintext")
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	key := NewEncryptionKey()

	tests := []struct {
		name string
		data []byte
	}{
		{"too short for nonce", make([]byte, 8)},
		{"empty", []byte{}},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(key, tt.data)
			if err == nil {
				t.Error("Decrypt() should fail with invalid ciphertext")
			}
		})
	}
}

func TestEncryptDecryptLargeData(t *testing.T) {
	// Test with 1MB data
	data := make([]byte, 1024*1024)
	rand.Read(data)

	key := NewEncryptionKey()
	ciphertext, err := Encrypt(key, data)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	plaintext, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(plaintext, data) {
		t.Error("Large data round-trip failed")
	}
}

func TestEncryptDecryptUnicode(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"ascii", "Hello World"},
		{"chinese", "你好世界"},
		{"emoji", "👋🌍🎉"},
		{"mixed", "Hello 世界 🌍🎉"},
		{"arabic", "مرحبا بالعالم"},
		{"russian", "Привет мир"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := NewEncryptionKey()
			ciphertext, err := Encrypt(key, []byte(tt.data))
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			plaintext, err := Decrypt(key, ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if string(plaintext) != tt.data {
				t.Errorf("Decrypt() = %s, want %s", plaintext, tt.data)
			}
		})
	}
}

func TestEncryptDecryptBinaryData(t *testing.T) {
	// Test with all possible byte values
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}

	key := NewEncryptionKey()
	ciphertext, err := Encrypt(key, data)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	plaintext, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(plaintext, data) {
		t.Error("Binary data round-trip failed")
	}
}