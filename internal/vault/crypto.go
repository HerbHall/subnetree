package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters for master key derivation.
const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32 // AES-256
	saltLen      = 16
	dekLen       = 32 // AES-256
	nonceLen     = 12 // AES-GCM standard nonce size
)

// verificationMagic is a known plaintext encrypted with the KEK
// to verify passphrase correctness on unseal.
var verificationMagic = []byte("subnetree-vault-v1")

// DeriveKEK derives a 32-byte master key from a passphrase and salt
// using Argon2id.
func DeriveKEK(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

// GenerateSalt returns a cryptographically random 16-byte salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}

// GenerateDEK returns a cryptographically random 32-byte data encryption key.
func GenerateDEK() ([]byte, error) {
	dek := make([]byte, dekLen)
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("generate DEK: %w", err)
	}
	return dek, nil
}

// WrapDEK encrypts a DEK with the KEK using AES-256-GCM.
// Returns nonce (12 bytes) || ciphertext+tag.
func WrapDEK(kek, dek []byte) ([]byte, error) {
	return encrypt(kek, dek)
}

// UnwrapDEK decrypts a wrapped DEK using the KEK.
func UnwrapDEK(kek, wrappedDEK []byte) ([]byte, error) {
	return decrypt(kek, wrappedDEK)
}

// Encrypt encrypts plaintext with a DEK using AES-256-GCM.
// Returns nonce (12 bytes) || ciphertext+tag.
func Encrypt(dek, plaintext []byte) ([]byte, error) {
	return encrypt(dek, plaintext)
}

// Decrypt decrypts ciphertext with a DEK using AES-256-GCM.
func Decrypt(dek, ciphertext []byte) ([]byte, error) {
	return decrypt(dek, ciphertext)
}

// CreateVerificationBlob encrypts a known magic string with the KEK.
// Used to verify passphrase correctness on subsequent unseals.
func CreateVerificationBlob(kek []byte) ([]byte, error) {
	return encrypt(kek, verificationMagic)
}

// VerifyKEK decrypts the verification blob and checks it matches
// the expected magic string. Returns true if the passphrase is correct.
func VerifyKEK(kek, blob []byte) bool {
	plain, err := decrypt(kek, blob)
	if err != nil {
		return false
	}
	if len(plain) != len(verificationMagic) {
		return false
	}
	// Constant-time compare is not necessary here because the magic
	// string is not secret -- it's just a known plaintext.
	for i := range plain {
		if plain[i] != verificationMagic[i] {
			return false
		}
	}
	return true
}

// ZeroBytes overwrites a byte slice with zeros.
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// encrypt performs AES-256-GCM encryption. Returns nonce || ciphertext+tag.
func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt performs AES-256-GCM decryption. Expects nonce || ciphertext+tag.
func decrypt(key, data []byte) ([]byte, error) {
	if len(data) < nonceLen {
		return nil, errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := data[:nonceLen]
	ciphertext := data[nonceLen:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}
