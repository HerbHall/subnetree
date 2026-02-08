package vault

import (
	"bytes"
	"testing"
)

func TestDeriveKEK_Deterministic(t *testing.T) {
	salt := []byte("1234567890abcdef")
	k1 := DeriveKEK("password", salt)
	k2 := DeriveKEK("password", salt)

	if !bytes.Equal(k1, k2) {
		t.Error("same passphrase+salt should produce same KEK")
	}
}

func TestDeriveKEK_DifferentSalts(t *testing.T) {
	salt1 := []byte("1234567890abcdef")
	salt2 := []byte("fedcba0987654321")
	k1 := DeriveKEK("password", salt1)
	k2 := DeriveKEK("password", salt2)

	if bytes.Equal(k1, k2) {
		t.Error("different salts should produce different KEKs")
	}
}

func TestDeriveKEK_DifferentPassphrases(t *testing.T) {
	salt := []byte("1234567890abcdef")
	k1 := DeriveKEK("password1", salt)
	k2 := DeriveKEK("password2", salt)

	if bytes.Equal(k1, k2) {
		t.Error("different passphrases should produce different KEKs")
	}
}

func TestDeriveKEK_Length(t *testing.T) {
	salt := []byte("1234567890abcdef")
	k := DeriveKEK("password", salt)
	if len(k) != 32 {
		t.Errorf("KEK length = %d, want 32", len(k))
	}
}

func TestGenerateSalt(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}
	if len(salt) != saltLen {
		t.Errorf("salt length = %d, want %d", len(salt), saltLen)
	}
}

func TestGenerateSalt_Uniqueness(t *testing.T) {
	s1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}
	s2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}
	if bytes.Equal(s1, s2) {
		t.Error("two GenerateSalt calls should produce different salts")
	}
}

func TestGenerateDEK_Length(t *testing.T) {
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatalf("GenerateDEK() error = %v", err)
	}
	if len(dek) != dekLen {
		t.Errorf("DEK length = %d, want %d", len(dek), dekLen)
	}
}

func TestGenerateDEK_Uniqueness(t *testing.T) {
	d1, err := GenerateDEK()
	if err != nil {
		t.Fatalf("GenerateDEK() error = %v", err)
	}
	d2, err := GenerateDEK()
	if err != nil {
		t.Fatalf("GenerateDEK() error = %v", err)
	}
	if bytes.Equal(d1, d2) {
		t.Error("two GenerateDEK calls should produce different DEKs")
	}
}

func TestWrapUnwrapDEK_RoundTrip(t *testing.T) {
	kek := DeriveKEK("test-passphrase", []byte("1234567890abcdef"))
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatalf("GenerateDEK() error = %v", err)
	}

	wrapped, err := WrapDEK(kek, dek)
	if err != nil {
		t.Fatalf("WrapDEK() error = %v", err)
	}

	unwrapped, err := UnwrapDEK(kek, wrapped)
	if err != nil {
		t.Fatalf("UnwrapDEK() error = %v", err)
	}

	if !bytes.Equal(dek, unwrapped) {
		t.Error("unwrapped DEK should equal original DEK")
	}
}

func TestWrapDEK_DifferentNonces(t *testing.T) {
	kek := DeriveKEK("test-passphrase", []byte("1234567890abcdef"))
	dek, _ := GenerateDEK()

	w1, err := WrapDEK(kek, dek)
	if err != nil {
		t.Fatalf("WrapDEK() error = %v", err)
	}
	w2, err := WrapDEK(kek, dek)
	if err != nil {
		t.Fatalf("WrapDEK() error = %v", err)
	}

	if bytes.Equal(w1, w2) {
		t.Error("wrapping same DEK twice should produce different ciphertexts (different nonces)")
	}
}

func TestUnwrapDEK_WrongKEK(t *testing.T) {
	kek1 := DeriveKEK("passphrase-1", []byte("1234567890abcdef"))
	kek2 := DeriveKEK("passphrase-2", []byte("1234567890abcdef"))
	dek, _ := GenerateDEK()

	wrapped, err := WrapDEK(kek1, dek)
	if err != nil {
		t.Fatalf("WrapDEK() error = %v", err)
	}

	_, err = UnwrapDEK(kek2, wrapped)
	if err == nil {
		t.Error("UnwrapDEK with wrong KEK should return error")
	}
}

func TestUnwrapDEK_CorruptedData(t *testing.T) {
	kek := DeriveKEK("passphrase", []byte("1234567890abcdef"))
	dek, _ := GenerateDEK()

	wrapped, err := WrapDEK(kek, dek)
	if err != nil {
		t.Fatalf("WrapDEK() error = %v", err)
	}

	// Corrupt a byte in the ciphertext portion.
	wrapped[nonceLen+2] ^= 0xFF

	_, err = UnwrapDEK(kek, wrapped)
	if err == nil {
		t.Error("UnwrapDEK with corrupted data should return error")
	}
}

func TestUnwrapDEK_TooShort(t *testing.T) {
	kek := DeriveKEK("passphrase", []byte("1234567890abcdef"))

	_, err := UnwrapDEK(kek, []byte("short"))
	if err == nil {
		t.Error("UnwrapDEK with too-short data should return error")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	dek, _ := GenerateDEK()
	plaintext := []byte(`{"username":"admin","password":"secret123"}`)

	encrypted, err := Encrypt(dek, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := Decrypt(dek, encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("decrypted data should equal original plaintext")
	}
}

func TestEncrypt_DifferentNonces(t *testing.T) {
	dek, _ := GenerateDEK()
	plaintext := []byte("same data")

	e1, _ := Encrypt(dek, plaintext)
	e2, _ := Encrypt(dek, plaintext)

	if bytes.Equal(e1, e2) {
		t.Error("encrypting same plaintext twice should produce different ciphertexts")
	}
}

func TestDecrypt_WrongDEK(t *testing.T) {
	dek1, _ := GenerateDEK()
	dek2, _ := GenerateDEK()
	plaintext := []byte("secret data")

	encrypted, _ := Encrypt(dek1, plaintext)

	_, err := Decrypt(dek2, encrypted)
	if err == nil {
		t.Error("Decrypt with wrong DEK should return error")
	}
}

func TestDecrypt_CorruptedData(t *testing.T) {
	dek, _ := GenerateDEK()
	plaintext := []byte("secret data")

	encrypted, _ := Encrypt(dek, plaintext)
	encrypted[nonceLen+2] ^= 0xFF

	_, err := Decrypt(dek, encrypted)
	if err == nil {
		t.Error("Decrypt with corrupted data should return error")
	}
}

func TestEncryptDecrypt_LargeData(t *testing.T) {
	dek, _ := GenerateDEK()
	// Simulate an SSH private key (~4KB).
	plaintext := make([]byte, 4096)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	encrypted, err := Encrypt(dek, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := Decrypt(dek, encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("decrypted large data should equal original")
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	dek, _ := GenerateDEK()

	encrypted, err := Encrypt(dek, []byte{})
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := Decrypt(dek, encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("decrypted empty plaintext should be empty, got %d bytes", len(decrypted))
	}
}

func TestCreateVerificationBlob_VerifyKEK(t *testing.T) {
	kek := DeriveKEK("test-pass", []byte("1234567890abcdef"))

	blob, err := CreateVerificationBlob(kek)
	if err != nil {
		t.Fatalf("CreateVerificationBlob() error = %v", err)
	}

	if !VerifyKEK(kek, blob) {
		t.Error("VerifyKEK should return true for correct KEK")
	}
}

func TestVerifyKEK_WrongKey(t *testing.T) {
	kek1 := DeriveKEK("pass-1", []byte("1234567890abcdef"))
	kek2 := DeriveKEK("pass-2", []byte("1234567890abcdef"))

	blob, _ := CreateVerificationBlob(kek1)

	if VerifyKEK(kek2, blob) {
		t.Error("VerifyKEK should return false for wrong KEK")
	}
}

func TestVerifyKEK_CorruptedBlob(t *testing.T) {
	kek := DeriveKEK("test-pass", []byte("1234567890abcdef"))
	blob, _ := CreateVerificationBlob(kek)

	blob[len(blob)-1] ^= 0xFF

	if VerifyKEK(kek, blob) {
		t.Error("VerifyKEK should return false for corrupted blob")
	}
}

func TestZeroBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	ZeroBytes(data)

	for i, b := range data {
		if b != 0 {
			t.Errorf("byte[%d] = %d, want 0", i, b)
		}
	}
}

func TestZeroBytes_Empty(t *testing.T) {
	// Should not panic.
	ZeroBytes([]byte{})
	ZeroBytes(nil)
}
