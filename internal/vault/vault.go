package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	keyLen   = 32
	nonceLen = 12
)

// Canary is a known plaintext we encrypt to test the key on unlock.
const Canary = "passmem-vault-v1"

// DeriveKey derives a 32-byte AES key from a password and salt using scrypt.
func DeriveKey(password string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(password), salt, 32768, 8, 1, keyLen)
}

// Vault wraps the 32-byte master key used to encrypt trainer passwords.
type Vault struct {
	key []byte
}

// New returns a Vault that uses the provided 32-byte key.
func New(key []byte) *Vault {
	return &Vault{key: key}
}

// Encrypt returns a base64 string containing nonce || ciphertext+tag.
func (v *Vault) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt returns the plaintext from a base64 nonce || ciphertext+tag string.
func (v *Vault) Decrypt(ciphertext string) (string, error) {
	ct, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	if len(ct) < nonceLen {
		return "", errors.New("ciphertext too short")
	}
	nonce := ct[:nonceLen]
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	pt, err := gcm.Open(nil, nonce, ct[nonceLen:], nil)
	if err != nil {
		return "", errors.New("decryption failed")
	}
	return string(pt), nil
}

// Verify checks that canaryCipher decrypts to the expected Canary string.
func (v *Vault) Verify(canaryCipher string) bool {
	pt, err := v.Decrypt(canaryCipher)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(pt), []byte(Canary)) == 1
}
