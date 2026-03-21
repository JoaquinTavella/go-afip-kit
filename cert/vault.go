package cert

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// KeyVault encrypts and decrypts sensitive data (private keys) using AES-256-GCM.
//
// The master key should be a 32-byte key (64 hex characters) stored in an
// environment variable (e.g. AFIP_MASTER_KEY), never in code or version control.
type KeyVault struct {
	masterKey []byte
}

// NewKeyVault creates a KeyVault from a hex-encoded master key.
// The key must be exactly 64 hex characters (32 bytes).
func NewKeyVault(masterKeyHex string) (*KeyVault, error) {
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, fmt.Errorf("keyvault: invalid hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("keyvault: key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	return &KeyVault{masterKey: key}, nil
}

// NewKeyVaultFromBytes creates a KeyVault from raw bytes.
func NewKeyVaultFromBytes(key []byte) (*KeyVault, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("keyvault: key must be 32 bytes, got %d", len(key))
	}
	dst := make([]byte, 32)
	copy(dst, key)
	return &KeyVault{masterKey: dst}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns nonce+ciphertext (nonce prepended).
func (kv *KeyVault) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(kv.masterKey)
	if err != nil {
		return nil, fmt.Errorf("keyvault encrypt: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("keyvault encrypt: new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("keyvault encrypt: generate nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data encrypted by Encrypt (nonce+ciphertext).
func (kv *KeyVault) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(kv.masterKey)
	if err != nil {
		return nil, fmt.Errorf("keyvault decrypt: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("keyvault decrypt: new gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("keyvault decrypt: ciphertext too short")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("keyvault decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptToBase64 encrypts and returns a base64-encoded string (for DB storage).
func (kv *KeyVault) EncryptToBase64(plaintext []byte) (string, error) {
	encrypted, err := kv.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptFromBase64 decodes base64 and decrypts (for reading from DB).
func (kv *KeyVault) DecryptFromBase64(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("keyvault: base64 decode: %w", err)
	}
	return kv.Decrypt(decoded)
}
