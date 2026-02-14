package ca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func testConfig(t *testing.T) Config {
	t.Helper()
	dir := t.TempDir()
	return Config{
		CertPath:     filepath.Join(dir, "ca.crt"),
		KeyPath:      filepath.Join(dir, "ca.key"),
		Organization: "TestOrg",
	}
}

func TestGenerateCA(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	authority, err := GenerateCA(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, authority)

	cert := authority.CACert()

	// Verify it is a CA certificate.
	assert.True(t, cert.IsCA, "certificate should be a CA")
	assert.True(t, cert.BasicConstraintsValid, "basic constraints should be valid")

	// Verify self-signed: issuer == subject.
	assert.Equal(t, cert.Subject.CommonName, cert.Issuer.CommonName)

	// Verify key usage.
	assert.NotZero(t, cert.KeyUsage&x509.KeyUsageCertSign, "should have KeyUsageCertSign")
	assert.NotZero(t, cert.KeyUsage&x509.KeyUsageDigitalSignature, "should have KeyUsageDigitalSignature")
	assert.NotZero(t, cert.KeyUsage&x509.KeyUsageCRLSign, "should have KeyUsageCRLSign")

	// Verify validity period is approximately 10 years.
	validity := cert.NotAfter.Sub(cert.NotBefore)
	assert.InDelta(t, DefaultCAValidity.Hours(), validity.Hours(), 24, "CA validity should be ~10 years")

	// Verify organization.
	require.Len(t, cert.Subject.Organization, 1)
	assert.Equal(t, "TestOrg", cert.Subject.Organization[0])

	// Verify common name.
	assert.Equal(t, "TestOrg Internal CA", cert.Subject.CommonName)

	// Verify PEM is available.
	pem := authority.CACertPEM()
	assert.NotEmpty(t, pem)
	assert.Contains(t, string(pem), "BEGIN CERTIFICATE")
}

func TestLoadExistingCA(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	// Generate first.
	original, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	// Load from same paths.
	loaded, err := NewAuthority(cfg, logger)
	require.NoError(t, err)

	// Verify identical certificate.
	assert.Equal(t, original.CACert().SerialNumber, loaded.CACert().SerialNumber)
	assert.Equal(t, original.CACert().Subject.CommonName, loaded.CACert().Subject.CommonName)
	assert.Equal(t, original.CACertPEM(), loaded.CACertPEM())
}

func TestLoadOrGenerate_NewCA(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	// No files exist -- should generate.
	authority, err := LoadOrGenerate(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, authority)

	assert.True(t, authority.CACert().IsCA)
}

func TestLoadOrGenerate_ExistingCA(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	// Generate first.
	original, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	// LoadOrGenerate should load, not regenerate.
	loaded, err := LoadOrGenerate(cfg, logger)
	require.NoError(t, err)

	assert.Equal(t, original.CACert().SerialNumber, loaded.CACert().SerialNumber)
}

func TestSignCSR(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	authority, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	// Generate agent keypair and CSR.
	agentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	csrDER, err := CreateCSR(agentKey, "agent-001", "myhost.local")
	require.NoError(t, err)

	// Sign the CSR.
	certDER, serial, expiresAt, err := authority.SignCSR(csrDER, "agent-001", 0)
	require.NoError(t, err)

	// Parse the signed certificate.
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	// Verify issuer matches CA.
	assert.Equal(t, authority.CACert().Subject.CommonName, cert.Issuer.CommonName)

	// Verify subject CN is the agent ID.
	assert.Equal(t, "agent-001", cert.Subject.CommonName)

	// Verify key usage.
	assert.NotZero(t, cert.KeyUsage&x509.KeyUsageDigitalSignature)
	assert.NotZero(t, cert.KeyUsage&x509.KeyUsageKeyEncipherment)

	// Verify extended key usage (client auth).
	assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth)

	// Verify it is NOT a CA.
	assert.False(t, cert.IsCA)

	// Verify default validity (~90 days).
	validity := cert.NotAfter.Sub(cert.NotBefore)
	assert.InDelta(t, DefaultValidity.Hours(), validity.Hours(), 24, "default validity should be ~90 days")

	// Verify serial is a valid hex string.
	assert.NotEmpty(t, serial)
	_, err = hex.DecodeString(serial)
	assert.NoError(t, err, "serial should be valid hex")

	// Verify expiry is in the future.
	assert.True(t, expiresAt.After(time.Now()))

	// Verify the certificate chains to the CA.
	roots := x509.NewCertPool()
	roots.AddCert(authority.CACert())
	_, err = cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	assert.NoError(t, err, "agent cert should verify against CA")
}

func TestSignCSR_InvalidCSR(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	authority, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	// Pass garbage bytes.
	_, _, _, err = authority.SignCSR([]byte("not-a-csr"), "agent-bad", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse CSR")
}

func TestSignCSR_CustomValidity(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	authority, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	agentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	csrDER, err := CreateCSR(agentKey, "agent-custom", "")
	require.NoError(t, err)

	customValidity := 30 * 24 * time.Hour // 30 days
	certDER, _, expiresAt, err := authority.SignCSR(csrDER, "agent-custom", customValidity)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	validity := cert.NotAfter.Sub(cert.NotBefore)
	assert.InDelta(t, customValidity.Hours(), validity.Hours(), 24, "custom validity should be ~30 days")

	// Expiry should be approximately 30 days from now.
	assert.InDelta(t, customValidity.Hours(), time.Until(expiresAt).Hours(), 1)
}

func TestSignCSR_UniqueSerials(t *testing.T) {
	cfg := testConfig(t)
	logger := zaptest.NewLogger(t)

	authority, err := GenerateCA(cfg, logger)
	require.NoError(t, err)

	serials := make(map[string]bool)
	for i := range 10 {
		agentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		csrDER, err := CreateCSR(agentKey, "agent-serial-"+string(rune('0'+i)), "")
		require.NoError(t, err)

		_, serial, _, err := authority.SignCSR(csrDER, "agent-serial-"+string(rune('0'+i)), 0)
		require.NoError(t, err)

		assert.False(t, serials[serial], "serial %s should be unique", serial)
		serials[serial] = true
	}

	assert.Len(t, serials, 10, "should have 10 unique serials")
}

func TestNewAuthority_MissingFiles(t *testing.T) {
	cfg := Config{
		CertPath: "/nonexistent/ca.crt",
		KeyPath:  "/nonexistent/ca.key",
	}
	logger := zaptest.NewLogger(t)

	_, err := NewAuthority(cfg, logger)
	assert.Error(t, err)
}
