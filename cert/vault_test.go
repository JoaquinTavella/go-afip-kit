package cert

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

func testMasterKeyHex() string {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return hex.EncodeToString(key)
}

func TestNewKeyVault_ValidKey(t *testing.T) {
	_, err := NewKeyVault(testMasterKeyHex())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewKeyVault_InvalidHex(t *testing.T) {
	_, err := NewKeyVault("not-hex")
	if err == nil {
		t.Fatal("expected error for invalid hex")
	}
}

func TestNewKeyVault_WrongLength(t *testing.T) {
	_, err := NewKeyVault("aabbccdd") // 4 bytes instead of 32
	if err == nil {
		t.Fatal("expected error for wrong key length")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	vault, err := NewKeyVault(testMasterKeyHex())
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIE... (test key data)\n-----END RSA PRIVATE KEY-----")

	encrypted, err := vault.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if bytes.Equal(encrypted, plaintext) {
		t.Fatal("encrypted should differ from plaintext")
	}

	decrypted, err := vault.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("decrypted does not match original plaintext")
	}
}

func TestEncryptDecryptBase64(t *testing.T) {
	vault, err := NewKeyVault(testMasterKeyHex())
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("secret private key content")

	encoded, err := vault.EncryptToBase64(plaintext)
	if err != nil {
		t.Fatalf("encrypt to base64: %v", err)
	}

	if encoded == string(plaintext) {
		t.Fatal("encoded should differ from plaintext")
	}

	decrypted, err := vault.DecryptFromBase64(encoded)
	if err != nil {
		t.Fatalf("decrypt from base64: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("decrypted does not match original")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	vault1, _ := NewKeyVault(testMasterKeyHex())
	vault2, _ := NewKeyVault(testMasterKeyHex())

	encrypted, err := vault1.Encrypt([]byte("test data"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = vault2.Decrypt(encrypted)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	vault, _ := NewKeyVault(testMasterKeyHex())
	_, err := vault.Decrypt([]byte("short"))
	if err == nil {
		t.Fatal("expected error for ciphertext too short")
	}
}

func TestEncrypt_DifferentNonce(t *testing.T) {
	vault, _ := NewKeyVault(testMasterKeyHex())
	plaintext := []byte("same data")

	enc1, _ := vault.Encrypt(plaintext)
	enc2, _ := vault.Encrypt(plaintext)

	if bytes.Equal(enc1, enc2) {
		t.Fatal("two encryptions of the same data should produce different ciphertext (different nonces)")
	}
}
