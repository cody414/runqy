package vaults

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// InitResult indicates the outcome of vault master key initialization.
type InitResult int

const (
	// InitOK means the key was valid and vaults are enabled.
	InitOK InitResult = iota
	// InitNotSet means the environment variable was empty or missing.
	InitNotSet
	// InitInvalidKey means the environment variable was set but had an invalid format.
	InitInvalidKey
)

const (
	// EnvMasterKey is the environment variable name for the vault master key.
	EnvMasterKey = "RUNQY_VAULT_MASTER_KEY"

	// keyLength is the required length for AES-256 keys (32 bytes).
	keyLength = 32
)

var (
	// ErrVaultsDisabled is returned when the master key is not configured.
	ErrVaultsDisabled = errors.New("vaults not configured: RUNQY_VAULT_MASTER_KEY not set")

	// ErrInvalidKey is returned when the master key has an invalid format.
	ErrInvalidKey = errors.New("invalid master key: must be 32 bytes (256 bits) base64-encoded")

	// ErrDecryptionFailed is returned when decryption fails.
	ErrDecryptionFailed = errors.New("decryption failed: ciphertext invalid or corrupted")
)

// Encryptor handles AES-256-GCM encryption/decryption for vault entries.
type Encryptor struct {
	key     []byte
	enabled bool
	mu      sync.RWMutex
}

var (
	globalEncryptor *Encryptor
	once            sync.Once
)

// GetEncryptor returns the singleton Encryptor instance.
// It initializes from the RUNQY_VAULT_MASTER_KEY environment variable.
// Deprecated: Use InitEncryptor(masterKey) instead for centralized config.
func GetEncryptor() *Encryptor {
	once.Do(func() {
		globalEncryptor = &Encryptor{}
		globalEncryptor.initFromEnv()
	})
	return globalEncryptor
}

// InitEncryptor creates and sets the global Encryptor using the provided master key.
// This should be called during server startup with the key from config.
// Returns the Encryptor and an InitResult indicating the outcome.
func InitEncryptor(masterKey string) (*Encryptor, InitResult) {
	var result InitResult
	once.Do(func() {
		globalEncryptor = &Encryptor{}
		if masterKey == "" {
			globalEncryptor.enabled = false
			result = InitNotSet
			return
		}
		key, err := base64.StdEncoding.DecodeString(masterKey)
		if err != nil || len(key) != keyLength {
			globalEncryptor.enabled = false
			result = InitInvalidKey
			return
		}
		globalEncryptor.key = key
		globalEncryptor.enabled = true
		result = InitOK
	})
	return globalEncryptor, result
}

// initFromEnv initializes the encryptor from the environment variable.
func (e *Encryptor) initFromEnv() {
	keyStr := os.Getenv(EnvMasterKey)
	if keyStr == "" {
		e.enabled = false
		return
	}

	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil || len(key) != keyLength {
		e.enabled = false
		log.Println("[VAULTS] Error: RUNQY_VAULT_MASTER_KEY is set but has invalid format (expected base64-encoded 32-byte key). Vaults feature disabled.")
		log.Println("[VAULTS] Hint: generate a valid key with: openssl rand -base64 32")
		return
	}

	e.key = key
	e.enabled = true
}

// IsEnabled returns true if encryption is properly configured.
func (e *Encryptor) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext (nonce prepended).
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.enabled {
		return nil, ErrVaultsDisabled
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext that was encrypted with Encrypt.
// Expects nonce prepended to ciphertext.
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.enabled {
		return nil, ErrVaultsDisabled
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrDecryptionFailed
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptString is a convenience method for encrypting strings.
func (e *Encryptor) EncryptString(plaintext string) ([]byte, error) {
	return e.Encrypt([]byte(plaintext))
}

// DecryptString is a convenience method for decrypting to strings.
func (e *Encryptor) DecryptString(ciphertext []byte) (string, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// GenerateKey generates a new random 256-bit key and returns it base64-encoded.
// This can be used to generate a master key for the environment variable.
func GenerateKey() (string, error) {
	key := make([]byte, keyLength)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// MaskSecret returns a masked version of a secret value.
func MaskSecret(value string) string {
	if len(value) == 0 {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	// Show first 2 and last 2 characters
	return value[:2] + "****" + value[len(value)-2:]
}
