package vault

import (
	"bytes"
	"testing"
)

func TestKeyManager_IsSealed_Initially(t *testing.T) {
	km := NewKeyManager()
	if !km.IsSealed() {
		t.Error("new KeyManager should be sealed")
	}
}

func TestKeyManager_IsInitialized_Initially(t *testing.T) {
	km := NewKeyManager()
	if km.IsInitialized() {
		t.Error("new KeyManager should not be initialized")
	}
}

func TestKeyManager_FirstRunSetup(t *testing.T) {
	km := NewKeyManager()

	salt, verification, err := km.FirstRunSetup("test-passphrase")
	if err != nil {
		t.Fatalf("FirstRunSetup() error = %v", err)
	}
	if len(salt) != saltLen {
		t.Errorf("salt length = %d, want %d", len(salt), saltLen)
	}
	if len(verification) == 0 {
		t.Error("verification blob should not be empty")
	}
	if km.IsSealed() {
		t.Error("should be unsealed after FirstRunSetup")
	}
	if !km.IsInitialized() {
		t.Error("should be initialized after FirstRunSetup")
	}
}

func TestKeyManager_FirstRunSetup_AlreadyInitialized(t *testing.T) {
	km := NewKeyManager()
	_, _, _ = km.FirstRunSetup("pass")

	_, _, err := km.FirstRunSetup("pass2")
	if err == nil {
		t.Error("second FirstRunSetup should return error")
	}
}

func TestKeyManager_Unseal_CorrectPassphrase(t *testing.T) {
	km := NewKeyManager()
	salt, verification, _ := km.FirstRunSetup("my-secret")
	km.Seal()

	// Simulate server restart: new KeyManager with stored data.
	km2 := NewKeyManager()
	km2.Initialize(salt, verification)

	if err := km2.Unseal("my-secret"); err != nil {
		t.Fatalf("Unseal() error = %v", err)
	}
	if km2.IsSealed() {
		t.Error("should be unsealed after correct passphrase")
	}
}

func TestKeyManager_Unseal_WrongPassphrase(t *testing.T) {
	km := NewKeyManager()
	salt, verification, _ := km.FirstRunSetup("correct-pass")
	km.Seal()

	km2 := NewKeyManager()
	km2.Initialize(salt, verification)

	err := km2.Unseal("wrong-pass")
	if err != ErrWrongPassphrase {
		t.Errorf("Unseal(wrong) error = %v, want ErrWrongPassphrase", err)
	}
	if !km2.IsSealed() {
		t.Error("should remain sealed after wrong passphrase")
	}
}

func TestKeyManager_Unseal_NotInitialized(t *testing.T) {
	km := NewKeyManager()
	err := km.Unseal("pass")
	if err == nil {
		t.Error("Unseal on uninitialized KeyManager should return error")
	}
}

func TestKeyManager_Unseal_AlreadyUnsealed(t *testing.T) {
	km := NewKeyManager()
	km.FirstRunSetup("pass")

	// Calling Unseal when already unsealed should be a no-op.
	if err := km.Unseal("pass"); err != nil {
		t.Errorf("Unseal on already-unsealed should not error: %v", err)
	}
}

func TestKeyManager_Seal(t *testing.T) {
	km := NewKeyManager()
	km.FirstRunSetup("pass")

	km.Seal()
	if !km.IsSealed() {
		t.Error("should be sealed after Seal()")
	}

	// Double seal should be safe.
	km.Seal()
	if !km.IsSealed() {
		t.Error("should remain sealed after double Seal()")
	}
}

func TestKeyManager_WrapUnwrapDEK(t *testing.T) {
	km := NewKeyManager()
	km.FirstRunSetup("pass")

	dek, _ := GenerateDEK()

	wrapped, err := km.WrapDEK(dek)
	if err != nil {
		t.Fatalf("WrapDEK() error = %v", err)
	}

	unwrapped, err := km.UnwrapDEK(wrapped)
	if err != nil {
		t.Fatalf("UnwrapDEK() error = %v", err)
	}

	if !bytes.Equal(dek, unwrapped) {
		t.Error("unwrapped DEK should match original")
	}
}

func TestKeyManager_WrapDEK_WhenSealed(t *testing.T) {
	km := NewKeyManager()
	km.FirstRunSetup("pass")
	km.Seal()

	dek, _ := GenerateDEK()
	_, err := km.WrapDEK(dek)
	if err != ErrVaultSealed {
		t.Errorf("WrapDEK when sealed error = %v, want ErrVaultSealed", err)
	}
}

func TestKeyManager_UnwrapDEK_WhenSealed(t *testing.T) {
	km := NewKeyManager()
	km.FirstRunSetup("pass")

	dek, _ := GenerateDEK()
	wrapped, _ := km.WrapDEK(dek)

	km.Seal()

	_, err := km.UnwrapDEK(wrapped)
	if err != ErrVaultSealed {
		t.Errorf("UnwrapDEK when sealed error = %v, want ErrVaultSealed", err)
	}
}

func TestKeyManager_RotateKEK(t *testing.T) {
	km := NewKeyManager()
	km.FirstRunSetup("old-pass")

	dek, _ := GenerateDEK()
	wrapped, err := km.WrapDEK(dek)
	if err != nil {
		t.Fatalf("WrapDEK() error = %v", err)
	}

	newSalt, newVerification, rewrap, err := km.RotateKEK("new-pass")
	if err != nil {
		t.Fatalf("RotateKEK() error = %v", err)
	}
	if len(newSalt) != saltLen {
		t.Errorf("new salt length = %d, want %d", len(newSalt), saltLen)
	}
	if len(newVerification) == 0 {
		t.Error("new verification blob should not be empty")
	}

	newWrapped, err := rewrap(wrapped)
	if err != nil {
		t.Fatalf("rewrap() error = %v", err)
	}

	// Unwrap with the new KEK (which is now active in km).
	unwrapped, err := km.UnwrapDEK(newWrapped)
	if err != nil {
		t.Fatalf("UnwrapDEK(newWrapped) error = %v", err)
	}

	if !bytes.Equal(dek, unwrapped) {
		t.Error("DEK after rotation should match original")
	}
}

func TestKeyManager_RotateKEK_WhenSealed(t *testing.T) {
	km := NewKeyManager()
	km.FirstRunSetup("pass")
	km.Seal()

	_, _, _, err := km.RotateKEK("new-pass")
	if err != ErrVaultSealed {
		t.Errorf("RotateKEK when sealed error = %v, want ErrVaultSealed", err)
	}
}

func TestKeyManager_RotateKEK_VerifyNewPassphrase(t *testing.T) {
	km := NewKeyManager()
	km.FirstRunSetup("old-pass")

	newSalt, newVerification, _, err := km.RotateKEK("new-pass")
	if err != nil {
		t.Fatalf("RotateKEK() error = %v", err)
	}

	// Simulate restart with new master key data.
	km2 := NewKeyManager()
	km2.Initialize(newSalt, newVerification)

	if err := km2.Unseal("new-pass"); err != nil {
		t.Fatalf("Unseal with new passphrase error = %v", err)
	}

	// Old passphrase should fail.
	km3 := NewKeyManager()
	km3.Initialize(newSalt, newVerification)

	if err := km3.Unseal("old-pass"); err != ErrWrongPassphrase {
		t.Errorf("Unseal with old passphrase error = %v, want ErrWrongPassphrase", err)
	}
}
