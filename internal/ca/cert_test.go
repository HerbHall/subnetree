package ca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestGenerateKeypair(t *testing.T) {
	key, pubPEM, err := GenerateKeypair()
	require.NoError(t, err)

	// Verify key is ECDSA P-256.
	ecKey, ok := key.(*ecdsa.PrivateKey)
	require.True(t, ok, "key should be *ecdsa.PrivateKey")
	assert.Equal(t, elliptic.P256(), ecKey.Curve)

	// Verify PEM encodes correctly.
	assert.Contains(t, string(pubPEM), "BEGIN PUBLIC KEY")
	assert.Contains(t, string(pubPEM), "END PUBLIC KEY")
}

func TestCreateCSR(t *testing.T) {
	key, _, err := GenerateKeypair()
	require.NoError(t, err)

	csrDER, err := CreateCSR(key, "agent-csr-test", "myhost.local")
	require.NoError(t, err)
	require.NotEmpty(t, csrDER)

	// Parse CSR and verify fields.
	csr, err := x509.ParseCertificateRequest(csrDER)
	require.NoError(t, err)

	assert.Equal(t, "agent-csr-test", csr.Subject.CommonName)
	require.Len(t, csr.Subject.Organization, 1)
	assert.Equal(t, DefaultOrganization, csr.Subject.Organization[0])
	assert.Contains(t, csr.DNSNames, "myhost.local")

	// Verify CSR signature.
	assert.NoError(t, csr.CheckSignature())
}

func TestCreateCSR_NoHostname(t *testing.T) {
	key, _, err := GenerateKeypair()
	require.NoError(t, err)

	csrDER, err := CreateCSR(key, "agent-no-host", "")
	require.NoError(t, err)

	csr, err := x509.ParseCertificateRequest(csrDER)
	require.NoError(t, err)

	assert.Equal(t, "agent-no-host", csr.Subject.CommonName)
	assert.Empty(t, csr.DNSNames, "no DNS names when hostname is empty")
}

func TestSavePEM_LoadPEM_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pem")

	// Generate some DER data (a certificate).
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	// Save as PEM.
	err = SavePEM(path, "PUBLIC KEY", der)
	require.NoError(t, err)

	// Load back.
	loaded, err := LoadPEM(path, "PUBLIC KEY")
	require.NoError(t, err)

	assert.Equal(t, der, loaded)
}

func TestSavePEM_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permissions not enforced on Windows")
	}

	dir := t.TempDir()

	// Test key file (should be 0600).
	keyPath := filepath.Join(dir, "test.key")
	err := SavePEM(keyPath, "EC PRIVATE KEY", []byte("fake-key-data"))
	require.NoError(t, err)

	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "key file should be 0600")

	// Test cert file (should be 0644).
	certPath := filepath.Join(dir, "test.crt")
	err = SavePEM(certPath, "CERTIFICATE", []byte("fake-cert-data"))
	require.NoError(t, err)

	info, err = os.Stat(certPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm(), "cert file should be 0644")
}

func TestLoadPEM_WrongType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pem")

	err := SavePEM(path, "CERTIFICATE", []byte("some-data"))
	require.NoError(t, err)

	_, err = LoadPEM(path, "EC PRIVATE KEY")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected PEM type")
}

func TestLoadPEM_FileNotFound(t *testing.T) {
	_, err := LoadPEM("/nonexistent/path.pem", "CERTIFICATE")
	assert.Error(t, err)
}

func TestEncodeKeyPEM_DecodeKeyPEM_Roundtrip(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	encoded, err := EncodeKeyPEM(key)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), "BEGIN EC PRIVATE KEY")

	decoded, err := DecodeKeyPEM(encoded)
	require.NoError(t, err)

	decodedEC, ok := decoded.(*ecdsa.PrivateKey)
	require.True(t, ok)

	// Keys should be equal.
	assert.True(t, key.Equal(decodedEC), "decoded key should match original")
}

func TestEncodeKeyPEM_NonECDSA(t *testing.T) {
	// Passing a non-ECDSA key should fail.
	_, err := EncodeKeyPEM("not-a-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *ecdsa.PrivateKey")
}

func TestDecodeKeyPEM_InvalidPEM(t *testing.T) {
	_, err := DecodeKeyPEM([]byte("not-pem-data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no PEM block found")
}

func TestDecodeKeyPEM_WrongType(t *testing.T) {
	_, err := DecodeKeyPEM([]byte("-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected PEM type")
}

func TestIsCertificateExpiringSoon(t *testing.T) {
	// Create a CA and sign a short-lived cert.
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	authority, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	agentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	csrDER, err := CreateCSR(agentKey, "agent-expiry", "")
	require.NoError(t, err)

	// Sign with 10-day validity.
	certDER, _, _, err := authority.SignCSR(csrDER, "agent-expiry", 10*24*time.Hour)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	// 30-day threshold: cert expiring in ~10 days should be "expiring soon".
	assert.True(t, IsCertificateExpiringSoon(cert, 30*24*time.Hour),
		"cert expiring in 10 days should be expiring soon with 30-day threshold")
}

func TestIsCertificateExpiringSoon_NotExpiring(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	authority, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	agentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	csrDER, err := CreateCSR(agentKey, "agent-long", "")
	require.NoError(t, err)

	// Sign with 60-day validity.
	certDER, _, _, err := authority.SignCSR(csrDER, "agent-long", 60*24*time.Hour)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	// 30-day threshold: cert expiring in ~60 days should NOT be "expiring soon".
	assert.False(t, IsCertificateExpiringSoon(cert, 30*24*time.Hour),
		"cert expiring in 60 days should not be expiring soon with 30-day threshold")
}

func TestCertificateExpiry(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	authority, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	expiry := CertificateExpiry(authority.CACert())
	assert.True(t, expiry.After(time.Now()), "CA cert expiry should be in the future")
}

func TestParseCertificate(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	authority, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	// Parse the CA cert DER.
	cert, err := ParseCertificate(authority.CACert().Raw)
	require.NoError(t, err)
	assert.Equal(t, authority.CACert().SerialNumber, cert.SerialNumber)
}

func TestParseCertificate_Invalid(t *testing.T) {
	_, err := ParseCertificate([]byte("invalid-der"))
	assert.Error(t, err)
}

func TestEncodeCertPEM_Empty(t *testing.T) {
	_, err := EncodeCertPEM(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty certificate")
}

func TestEncodeCertPEM_Valid(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	authority, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	pem, err := EncodeCertPEM(authority.CACert().Raw)
	require.NoError(t, err)
	assert.Contains(t, string(pem), "BEGIN CERTIFICATE")
	assert.Contains(t, string(pem), "END CERTIFICATE")
}
