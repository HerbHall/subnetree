package vault

import (
	"errors"
	"sync"
)

// ErrVaultSealed is returned when an operation requires an unsealed vault.
var ErrVaultSealed = errors.New("vault is sealed")

// ErrWrongPassphrase is returned when the passphrase does not match.
var ErrWrongPassphrase = errors.New("wrong passphrase")

// KeyManager holds the KEK in memory and provides seal/unseal operations.
// All methods are safe for concurrent use.
type KeyManager struct {
	mu               sync.RWMutex
	kek              []byte // nil when sealed
	salt             []byte // loaded from DB during Initialize
	verificationBlob []byte // loaded from DB during Initialize
	initialized      bool   // true if master key record exists in DB
}

// NewKeyManager creates a new KeyManager in sealed state.
func NewKeyManager() *KeyManager {
	return &KeyManager{}
}

// Initialize loads the salt and verification blob from storage.
// Called during Init(). Does NOT unseal the vault.
func (km *KeyManager) Initialize(salt, verificationBlob []byte) {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.salt = salt
	km.verificationBlob = verificationBlob
	km.initialized = true
}

// IsSealed returns true if the vault has no KEK in memory.
func (km *KeyManager) IsSealed() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.kek == nil
}

// IsInitialized returns true if a master key record has been loaded.
func (km *KeyManager) IsInitialized() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.initialized
}

// Unseal derives the KEK from the passphrase and stored salt, then
// verifies it against the stored verification blob. Returns an error
// if the vault is not initialized or the passphrase is wrong.
func (km *KeyManager) Unseal(passphrase string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if !km.initialized {
		return errors.New("vault not initialized: call FirstRunSetup first")
	}

	if km.kek != nil {
		return nil // already unsealed
	}

	kek := DeriveKEK(passphrase, km.salt)
	if !VerifyKEK(kek, km.verificationBlob) {
		ZeroBytes(kek)
		return ErrWrongPassphrase
	}

	km.kek = kek
	return nil
}

// FirstRunSetup creates a new salt, derives the KEK, and creates a
// verification blob. Returns the salt and verification blob for
// persistence to the database. The vault is unsealed after this call.
func (km *KeyManager) FirstRunSetup(passphrase string) (salt []byte, verification []byte, err error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.initialized {
		return nil, nil, errors.New("vault already initialized")
	}

	salt, err = GenerateSalt()
	if err != nil {
		return nil, nil, err
	}

	kek := DeriveKEK(passphrase, salt)
	verification, err = CreateVerificationBlob(kek)
	if err != nil {
		ZeroBytes(kek)
		return nil, nil, err
	}

	km.salt = salt
	km.verificationBlob = verification
	km.kek = kek
	km.initialized = true

	return salt, verification, nil
}

// Seal zeroes the KEK and removes it from memory.
func (km *KeyManager) Seal() {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.kek != nil {
		ZeroBytes(km.kek)
		km.kek = nil
	}
}

// WrapDEK wraps a DEK with the current KEK. Returns ErrVaultSealed if sealed.
func (km *KeyManager) WrapDEK(dek []byte) ([]byte, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if km.kek == nil {
		return nil, ErrVaultSealed
	}
	return WrapDEK(km.kek, dek)
}

// UnwrapDEK unwraps a wrapped DEK using the current KEK.
// Returns ErrVaultSealed if sealed.
func (km *KeyManager) UnwrapDEK(wrappedDEK []byte) ([]byte, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if km.kek == nil {
		return nil, ErrVaultSealed
	}
	return UnwrapDEK(km.kek, wrappedDEK)
}

// RotateKEK derives a new KEK from a new passphrase and returns the
// new salt, verification blob, and a rewrap function that re-wraps
// existing DEKs from the old KEK to the new KEK. The old KEK is
// replaced after this call.
func (km *KeyManager) RotateKEK(newPassphrase string) (salt, verification []byte, rewrap func([]byte) ([]byte, error), err error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.kek == nil {
		return nil, nil, nil, ErrVaultSealed
	}

	salt, err = GenerateSalt()
	if err != nil {
		return nil, nil, nil, err
	}

	newKEK := DeriveKEK(newPassphrase, salt)
	verification, err = CreateVerificationBlob(newKEK)
	if err != nil {
		ZeroBytes(newKEK)
		return nil, nil, nil, err
	}

	// Copy old KEK for the rewrap closure (the original will be zeroed).
	oldKEK := make([]byte, len(km.kek))
	copy(oldKEK, km.kek)

	rewrap = func(wrappedDEK []byte) ([]byte, error) {
		dek, err := UnwrapDEK(oldKEK, wrappedDEK)
		if err != nil {
			return nil, err
		}
		defer ZeroBytes(dek)
		return WrapDEK(newKEK, dek)
	}

	// Replace keys. The caller must use rewrap, then call ZeroBytes(oldKEK)
	// via the returned closure when done.
	ZeroBytes(km.kek)
	km.kek = newKEK
	km.salt = salt
	km.verificationBlob = verification

	return salt, verification, rewrap, nil
}
