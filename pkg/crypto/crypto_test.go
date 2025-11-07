package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	plaintext := []byte("Hello, World! This is a secret message.")
	passphrase := "my-secret-passphrase"

	// Encrypt
	ciphertext, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Verify ciphertext is different from plaintext
	if bytes.Equal(ciphertext, plaintext) {
		t.Error("Ciphertext should be different from plaintext")
	}

	// Verify ciphertext has expected minimum size
	minSize := saltSize + nonceSize + 16 // salt + nonce + tag
	if len(ciphertext) < minSize {
		t.Errorf("Ciphertext too short: got %d, want at least %d", len(ciphertext), minSize)
	}

	// Decrypt
	decrypted, err := Decrypt(ciphertext, passphrase)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	// Verify roundtrip
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted text doesn't match original.\nGot: %s\nWant: %s", decrypted, plaintext)
	}
}

func TestEncryptDecryptEmpty(t *testing.T) {
	plaintext := []byte("")
	passphrase := "test-passphrase"

	ciphertext, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, passphrase)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Empty plaintext roundtrip failed")
	}
}

func TestEncryptDecryptLargeData(t *testing.T) {
	// Test with 1MB of data
	plaintext := make([]byte, 1024*1024)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}
	passphrase := "test-passphrase"

	ciphertext, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, passphrase)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Large data roundtrip failed")
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	plaintext := []byte("Secret message")
	correctPass := "correct-passphrase"
	wrongPass := "wrong-passphrase"

	ciphertext, err := Encrypt(plaintext, correctPass)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Try to decrypt with wrong passphrase
	_, err = Decrypt(ciphertext, wrongPass)
	if err == nil {
		t.Error("Decrypt should fail with wrong passphrase")
	}
}

func TestDecryptCorruptedData(t *testing.T) {
	plaintext := []byte("Secret message")
	passphrase := "test-passphrase"

	ciphertext, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Corrupt the ciphertext (flip a byte in the middle)
	corrupted := make([]byte, len(ciphertext))
	copy(corrupted, ciphertext)
	corrupted[len(corrupted)/2] ^= 0xFF

	// Try to decrypt corrupted data
	_, err = Decrypt(corrupted, passphrase)
	if err == nil {
		t.Error("Decrypt should fail with corrupted data")
	}
}

func TestDecryptTooShort(t *testing.T) {
	// Ciphertext too short to contain salt + nonce + tag
	shortData := make([]byte, 10)
	passphrase := "test-passphrase"

	_, err := Decrypt(shortData, passphrase)
	if err == nil {
		t.Error("Decrypt should fail with too-short ciphertext")
	}
}

func TestEncryptEmptyPassphrase(t *testing.T) {
	plaintext := []byte("Secret message")

	_, err := Encrypt(plaintext, "")
	if err == nil {
		t.Error("Encrypt should fail with empty passphrase")
	}
}

func TestDecryptEmptyPassphrase(t *testing.T) {
	ciphertext := make([]byte, 100)

	_, err := Decrypt(ciphertext, "")
	if err == nil {
		t.Error("Decrypt should fail with empty passphrase")
	}
}

func TestEncryptDifferentNonces(t *testing.T) {
	// Encrypting same plaintext twice should produce different ciphertext
	// (due to random nonces)
	plaintext := []byte("Same message")
	passphrase := "test-passphrase"

	ciphertext1, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("First encrypt failed: %v", err)
	}

	ciphertext2, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Second encrypt failed: %v", err)
	}

	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("Two encryptions of same plaintext should produce different ciphertext")
	}

	// Both should decrypt correctly
	decrypted1, err := Decrypt(ciphertext1, passphrase)
	if err != nil {
		t.Fatalf("First decrypt failed: %v", err)
	}

	decrypted2, err := Decrypt(ciphertext2, passphrase)
	if err != nil {
		t.Fatalf("Second decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted1, plaintext) || !bytes.Equal(decrypted2, plaintext) {
		t.Error("Decrypted text doesn't match original")
	}
}

func TestEncryptFileDecryptFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tempDir, "source.txt")
	plaintext := []byte("This is a test file with secret content.\n")
	if err := os.WriteFile(srcPath, plaintext, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Encrypt file
	encPath := filepath.Join(tempDir, "encrypted.enc")
	passphrase := "file-passphrase"

	if err := EncryptFile(srcPath, encPath, passphrase); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	// Verify encrypted file exists
	if _, err := os.Stat(encPath); os.IsNotExist(err) {
		t.Fatal("Encrypted file was not created")
	}

	// Decrypt file
	dstPath := filepath.Join(tempDir, "decrypted.txt")

	if err := DecryptFile(encPath, dstPath, passphrase); err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	// Verify decrypted content matches original
	decrypted, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read decrypted file: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted file content doesn't match original")
	}
}

func TestEncryptFileNonexistent(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "nonexistent.txt")
	encPath := filepath.Join(tempDir, "encrypted.enc")

	err := EncryptFile(srcPath, encPath, "passphrase")
	if err == nil {
		t.Error("EncryptFile should fail with nonexistent source file")
	}
}

func TestDecryptFileWrongPassphrase(t *testing.T) {
	tempDir := t.TempDir()

	// Create and encrypt a file
	srcPath := filepath.Join(tempDir, "source.txt")
	plaintext := []byte("Secret content")
	if err := os.WriteFile(srcPath, plaintext, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	encPath := filepath.Join(tempDir, "encrypted.enc")
	correctPass := "correct"

	if err := EncryptFile(srcPath, encPath, correctPass); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	// Try to decrypt with wrong passphrase
	dstPath := filepath.Join(tempDir, "decrypted.txt")
	wrongPass := "wrong"

	err := DecryptFile(encPath, dstPath, wrongPass)
	if err == nil {
		t.Error("DecryptFile should fail with wrong passphrase")
	}
}

func TestEncryptDecryptSpecialCharacters(t *testing.T) {
	// Test with various special characters and unicode
	plaintext := []byte("Special: 日本語, émojis 🔒🔑, quotes \"'`, newlines\n\ttabs")
	passphrase := "pässwörd-with-ünicode-😀"

	ciphertext, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, passphrase)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Special characters roundtrip failed")
	}
}

func TestFilePermissions(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("secret"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	encPath := filepath.Join(tempDir, "encrypted.enc")

	if err := EncryptFile(srcPath, encPath, "pass"); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	// Check that encrypted file has restrictive permissions (0600)
	info, err := os.Stat(encPath)
	if err != nil {
		t.Fatalf("Failed to stat encrypted file: %v", err)
	}

	mode := info.Mode()
	if mode.Perm() != 0600 {
		t.Errorf("Encrypted file has wrong permissions: got %o, want 0600", mode.Perm())
	}
}
