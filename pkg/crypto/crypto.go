package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// AES-256 requires 32-byte key
	keySize = 32

	// Standard salt size for PBKDF2
	saltSize = 16

	// GCM standard nonce size
	nonceSize = 12

	// PBKDF2 iterations (100k is reasonable balance of security and speed)
	pbkdf2Iterations = 100000
)

// Encrypt encrypts plaintext with the given passphrase using AES-256-GCM
// Returns: [salt(16)][nonce(12)][ciphertext][tag(16)]
func Encrypt(plaintext []byte, passphrase string) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("passphrase cannot be empty")
	}

	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key from passphrase using PBKDF2
	key := pbkdf2.Key([]byte(passphrase), salt, pbkdf2Iterations, keySize, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Combine: salt + nonce + ciphertext (which includes auth tag)
	result := make([]byte, 0, saltSize+nonceSize+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt decrypts ciphertext with the given passphrase
func Decrypt(ciphertext []byte, passphrase string) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("passphrase cannot be empty")
	}

	// Minimum size: salt + nonce + tag
	minSize := saltSize + nonceSize + 16 // GCM tag is 16 bytes
	if len(ciphertext) < minSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract components
	salt := ciphertext[:saltSize]
	nonce := ciphertext[saltSize : saltSize+nonceSize]
	encrypted := ciphertext[saltSize+nonceSize:]

	// Derive key from passphrase using same parameters
	key := pbkdf2.Key([]byte(passphrase), salt, pbkdf2Iterations, keySize, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong passphrase or corrupted data): %w", err)
	}

	return plaintext, nil
}

// EncryptFile encrypts a file with the given passphrase
func EncryptFile(srcPath, dstPath, passphrase string) error {
	// Read source file
	plaintext, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Encrypt
	ciphertext, err := Encrypt(plaintext, passphrase)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	// Write encrypted file
	if err := os.WriteFile(dstPath, ciphertext, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}

// DecryptFile decrypts a file with the given passphrase
func DecryptFile(srcPath, dstPath, passphrase string) error {
	// Read encrypted file
	ciphertext, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	// Decrypt
	plaintext, err := Decrypt(ciphertext, passphrase)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// Write decrypted file
	if err := os.WriteFile(dstPath, plaintext, 0600); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	return nil
}
