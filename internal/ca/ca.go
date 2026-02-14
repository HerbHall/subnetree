package ca

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"time"

	"go.uber.org/zap"
)

// serialBits is the number of random bits for certificate serial numbers.
const serialBits = 128

// Default configuration values.
const (
	DefaultValidity     = 90 * 24 * time.Hour  // 90 days for agent certs
	DefaultCAValidity   = 10 * 365 * 24 * time.Hour // 10 years for CA root
	DefaultOrganization = "SubNetree"
)

// Authority manages an internal certificate authority.
type Authority struct {
	cert    *x509.Certificate
	key     crypto.PrivateKey
	certPEM []byte // cached PEM encoding
	logger  *zap.Logger
}

// Config holds CA configuration.
type Config struct {
	CertPath     string        // path to CA certificate PEM file
	KeyPath      string        // path to CA private key PEM file
	Validity     time.Duration // default cert validity for signed certs (default 90 days)
	Organization string        // O= field in certificates (default "SubNetree")
}

// defaults fills in zero-value config fields with defaults.
func (c *Config) defaults() {
	if c.Validity == 0 {
		c.Validity = DefaultValidity
	}
	if c.Organization == "" {
		c.Organization = DefaultOrganization
	}
}

// NewAuthority loads an existing CA from disk.
// Returns an error if the cert or key files do not exist or are invalid.
func NewAuthority(cfg Config, logger *zap.Logger) (*Authority, error) {
	cfg.defaults()

	certPEM, err := LoadPEM(cfg.CertPath, "CERTIFICATE")
	if err != nil {
		return nil, fmt.Errorf("load CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certPEM)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}

	keyPEM, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read CA key: %w", err)
	}

	key, err := DecodeKeyPEM(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("decode CA key: %w", err)
	}

	encoded, err := EncodeCertPEM(cert.Raw)
	if err != nil {
		return nil, fmt.Errorf("encode CA cert PEM: %w", err)
	}

	logger.Info("loaded existing CA",
		zap.String("subject", cert.Subject.CommonName),
		zap.Time("not_after", cert.NotAfter),
	)

	return &Authority{
		cert:    cert,
		key:     key,
		certPEM: encoded,
		logger:  logger,
	}, nil
}

// GenerateCA generates a new CA keypair and self-signed root certificate,
// saves them to disk, and returns the Authority.
func GenerateCA(cfg Config, logger *zap.Logger) (*Authority, error) {
	cfg.defaults()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, fmt.Errorf("generate CA serial: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{cfg.Organization},
			CommonName:   cfg.Organization + " Internal CA",
		},
		NotBefore:             now.Add(-5 * time.Minute), // small clock skew allowance
		NotAfter:              now.Add(DefaultCAValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create CA certificate: %w", err)
	}

	// Save key to disk.
	keyPEM, err := EncodeKeyPEM(key)
	if err != nil {
		return nil, fmt.Errorf("encode CA key: %w", err)
	}
	if err := os.WriteFile(cfg.KeyPath, keyPEM, 0o600); err != nil {
		return nil, fmt.Errorf("write CA key: %w", err)
	}

	// Save cert to disk.
	certPEM, err := EncodeCertPEM(certDER)
	if err != nil {
		return nil, fmt.Errorf("encode CA cert: %w", err)
	}
	if err := SavePEM(cfg.CertPath, "CERTIFICATE", certDER); err != nil {
		return nil, fmt.Errorf("write CA cert: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("parse generated CA cert: %w", err)
	}

	logger.Info("generated new CA",
		zap.String("subject", cert.Subject.CommonName),
		zap.Time("not_after", cert.NotAfter),
		zap.String("serial", hex.EncodeToString(cert.SerialNumber.Bytes())),
	)

	return &Authority{
		cert:    cert,
		key:     key,
		certPEM: certPEM,
		logger:  logger,
	}, nil
}

// LoadOrGenerate loads an existing CA from disk if the cert file exists,
// or generates a new one if it does not.
func LoadOrGenerate(cfg Config, logger *zap.Logger) (*Authority, error) {
	if _, err := os.Stat(cfg.CertPath); err == nil {
		return NewAuthority(cfg, logger)
	}
	return GenerateCA(cfg, logger)
}

// CACertPEM returns the PEM-encoded CA certificate.
// Agents need this to verify the server certificate.
func (a *Authority) CACertPEM() []byte {
	out := make([]byte, len(a.certPEM))
	copy(out, a.certPEM)
	return out
}

// CACert returns the parsed CA certificate.
func (a *Authority) CACert() *x509.Certificate {
	return a.cert
}

// SignCSR parses a DER-encoded CSR, validates it, signs it with the CA key,
// and returns the DER-encoded certificate, serial number hex string, and expiry time.
func (a *Authority) SignCSR(csrDER []byte, agentID string, validity time.Duration) (certDER []byte, serial string, expiresAt time.Time, err error) {
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("parse CSR: %w", err)
	}

	if err := csr.CheckSignature(); err != nil {
		return nil, "", time.Time{}, fmt.Errorf("invalid CSR signature: %w", err)
	}

	if validity == 0 {
		validity = DefaultValidity
	}

	serialNum, err := randomSerial()
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now()
	expiresAt = now.Add(validity)

	template := &x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			Organization: []string{a.cert.Subject.Organization[0]},
			CommonName:   agentID,
		},
		NotBefore: now.Add(-5 * time.Minute), // small clock skew allowance
		NotAfter:  expiresAt,
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}

	certDER, err = x509.CreateCertificate(rand.Reader, template, a.cert, csr.PublicKey, a.key)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("sign certificate: %w", err)
	}

	serialHex := hex.EncodeToString(serialNum.Bytes())

	a.logger.Info("signed agent certificate",
		zap.String("agent_id", agentID),
		zap.String("serial", serialHex),
		zap.Time("expires_at", expiresAt),
	)

	return certDER, serialHex, expiresAt, nil
}

// randomSerial generates a cryptographically random 128-bit serial number.
func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), serialBits)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, fmt.Errorf("generate random serial: %w", err)
	}
	return serial, nil
}
