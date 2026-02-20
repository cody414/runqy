package vaults

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestEncryptor creates an Encryptor with a random valid key,
// bypassing the package-level sync.Once.
func newTestEncryptor(t *testing.T) *Encryptor {
	t.Helper()
	key := make([]byte, keyLength)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return &Encryptor{key: key, enabled: true}
}

// newTestEncryptorFromKey creates an Encryptor from a specific key.
func newTestEncryptorFromKey(key []byte) *Encryptor {
	return &Encryptor{key: key, enabled: true}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	enc := newTestEncryptor(t)

	plaintext := []byte("secret vault entry")
	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecryptStringRoundtrip(t *testing.T) {
	enc := newTestEncryptor(t)

	original := "hello world"
	ciphertext, err := enc.EncryptString(original)
	require.NoError(t, err)

	decrypted, err := enc.DecryptString(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestDecryptWithWrongKey(t *testing.T) {
	enc1 := newTestEncryptor(t)
	enc2 := newTestEncryptor(t)

	ciphertext, err := enc1.Encrypt([]byte("secret"))
	require.NoError(t, err)

	_, err = enc2.Decrypt(ciphertext)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	enc := newTestEncryptor(t)

	ciphertext, err := enc.Encrypt([]byte("secret"))
	require.NoError(t, err)

	// Corrupt a byte in the middle
	ciphertext[len(ciphertext)/2] ^= 0xFF

	_, err = enc.Decrypt(ciphertext)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestDecryptTooShortCiphertext(t *testing.T) {
	enc := newTestEncryptor(t)

	_, err := enc.Decrypt([]byte{0x01, 0x02})
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestEncryptWhenDisabled(t *testing.T) {
	enc := &Encryptor{enabled: false}

	_, err := enc.Encrypt([]byte("secret"))
	assert.ErrorIs(t, err, ErrVaultsDisabled)
}

func TestDecryptWhenDisabled(t *testing.T) {
	enc := &Encryptor{enabled: false}

	_, err := enc.Decrypt([]byte("doesn't matter"))
	assert.ErrorIs(t, err, ErrVaultsDisabled)
}

func TestIsEnabled(t *testing.T) {
	enabled := &Encryptor{enabled: true}
	disabled := &Encryptor{enabled: false}

	assert.True(t, enabled.IsEnabled())
	assert.False(t, disabled.IsEnabled())
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	enc := newTestEncryptor(t)
	plaintext := []byte("same input")

	ct1, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	ct2, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	// Different nonces → different ciphertexts
	assert.NotEqual(t, ct1, ct2)
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", "****"},
		{"ab", "****"},
		{"abc", "****"},
		{"abcd", "****"},
		{"abcde", "ab****de"},
		{"abcdef", "ab****ef"},
		{"my-secret-value", "my****ue"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskSecret(tt.input))
		})
	}
}

func TestGenerateKey(t *testing.T) {
	keyStr, err := GenerateKey()
	require.NoError(t, err)
	assert.NotEmpty(t, keyStr)

	// Should be valid base64
	decoded, err := base64.StdEncoding.DecodeString(keyStr)
	require.NoError(t, err)
	assert.Len(t, decoded, keyLength)
}

func TestGenerateKeyUniqueness(t *testing.T) {
	key1, err := GenerateKey()
	require.NoError(t, err)

	key2, err := GenerateKey()
	require.NoError(t, err)

	assert.NotEqual(t, key1, key2)
}
