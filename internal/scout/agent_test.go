package scout

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/ca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// writeTestCert generates a CA, signs a certificate for the given agentID with
// the given validity, and writes the cert, key, and CA cert to the given directory.
// Returns the authority and the parsed certificate.
func writeTestCert(t *testing.T, dir, agentID string, validity time.Duration) (*ca.Authority, *x509.Certificate) {
	t.Helper()

	caCfg := ca.Config{
		CertPath:     filepath.Join(dir, "ca.crt"),
		KeyPath:      filepath.Join(dir, "ca.key"),
		Organization: "TestOrg",
	}
	authority, err := ca.GenerateCA(caCfg, zaptest.NewLogger(t))
	require.NoError(t, err)

	// Generate agent keypair and CSR.
	agentKey, _, err := ca.GenerateKeypair()
	require.NoError(t, err)

	csrDER, err := ca.CreateCSR(agentKey, agentID, "test-host")
	require.NoError(t, err)

	certDER, _, _, err := authority.SignCSR(csrDER, agentID, validity)
	require.NoError(t, err)

	// Save agent cert.
	certPEM, err := ca.EncodeCertPEM(certDER)
	require.NoError(t, err)
	certPath := filepath.Join(dir, "agent.crt")
	require.NoError(t, os.WriteFile(certPath, certPEM, 0o600))

	// Save agent key.
	keyPEM, err := ca.EncodeKeyPEM(agentKey)
	require.NoError(t, err)
	keyPath := filepath.Join(dir, "agent.key")
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0o600))

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return authority, cert
}

func TestCheckCertificateRenewal_NotExpiring(t *testing.T) {
	dir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// Create a certificate that expires in 90 days (well above 30-day threshold).
	writeTestCert(t, dir, "agent-ok", 90*24*time.Hour)

	agent := NewAgent(&Config{
		ServerAddr:       "localhost:9090",
		CheckInterval:    30,
		AgentID:          "agent-ok",
		CertPath:         filepath.Join(dir, "agent.crt"),
		KeyPath:          filepath.Join(dir, "agent.key"),
		CACertPath:       filepath.Join(dir, "ca.crt"),
		Insecure:         false,
		RenewalThreshold: 30 * 24 * time.Hour,
	}, logger)

	csrDER, newKey, err := agent.checkCertificateRenewal()
	require.NoError(t, err)
	assert.Nil(t, csrDER, "should not generate CSR when cert is not expiring")
	assert.Nil(t, newKey, "should not generate key when cert is not expiring")
}

func TestCheckCertificateRenewal_ExpiringSoon(t *testing.T) {
	dir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// Create a certificate that expires in 5 days (within 30-day threshold).
	writeTestCert(t, dir, "agent-expiring", 5*24*time.Hour)

	agent := NewAgent(&Config{
		ServerAddr:       "localhost:9090",
		CheckInterval:    30,
		AgentID:          "agent-expiring",
		CertPath:         filepath.Join(dir, "agent.crt"),
		KeyPath:          filepath.Join(dir, "agent.key"),
		CACertPath:       filepath.Join(dir, "ca.crt"),
		Insecure:         false,
		RenewalThreshold: 30 * 24 * time.Hour,
	}, logger)

	csrDER, newKey, err := agent.checkCertificateRenewal()
	require.NoError(t, err)
	assert.NotNil(t, csrDER, "should generate CSR when cert is expiring soon")
	assert.NotNil(t, newKey, "should generate new key when cert is expiring soon")

	// Verify the CSR is valid.
	csr, err := x509.ParseCertificateRequest(csrDER)
	require.NoError(t, err)
	assert.Equal(t, "agent-expiring", csr.Subject.CommonName)
	require.NoError(t, csr.CheckSignature())
}

func TestCheckCertificateRenewal_InsecureMode(t *testing.T) {
	logger := zaptest.NewLogger(t)

	agent := NewAgent(&Config{
		ServerAddr:       "localhost:9090",
		CheckInterval:    30,
		Insecure:         true,
		RenewalThreshold: 30 * 24 * time.Hour,
	}, logger)

	csrDER, newKey, err := agent.checkCertificateRenewal()
	require.NoError(t, err)
	assert.Nil(t, csrDER, "should not attempt renewal in insecure mode")
	assert.Nil(t, newKey, "should not attempt renewal in insecure mode")
}

func TestCheckCertificateRenewal_NoCredentials(t *testing.T) {
	logger := zaptest.NewLogger(t)

	agent := NewAgent(&Config{
		ServerAddr:       "localhost:9090",
		CheckInterval:    30,
		Insecure:         false,
		CertPath:         "/nonexistent/agent.crt",
		KeyPath:          "/nonexistent/agent.key",
		RenewalThreshold: 30 * 24 * time.Hour,
	}, logger)

	csrDER, newKey, err := agent.checkCertificateRenewal()
	require.NoError(t, err)
	assert.Nil(t, csrDER, "should not attempt renewal when no credentials exist")
	assert.Nil(t, newKey, "should not attempt renewal when no credentials exist")
}

func TestCheckCertificateRenewal_ThresholdBoundary(t *testing.T) {
	dir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// Create a certificate that expires in exactly 29 days (within 30-day threshold).
	writeTestCert(t, dir, "agent-boundary", 29*24*time.Hour)

	agent := NewAgent(&Config{
		ServerAddr:       "localhost:9090",
		CheckInterval:    30,
		AgentID:          "agent-boundary",
		CertPath:         filepath.Join(dir, "agent.crt"),
		KeyPath:          filepath.Join(dir, "agent.key"),
		CACertPath:       filepath.Join(dir, "ca.crt"),
		Insecure:         false,
		RenewalThreshold: 30 * 24 * time.Hour,
	}, logger)

	csrDER, newKey, err := agent.checkCertificateRenewal()
	require.NoError(t, err)
	assert.NotNil(t, csrDER, "should generate CSR when cert expires within threshold")
	assert.NotNil(t, newKey, "should generate key when cert expires within threshold")
}
