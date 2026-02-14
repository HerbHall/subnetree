package ca

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// GenerateKeypair generates an ECDSA P-256 keypair and returns the private key
// and the PEM-encoded public key.
func GenerateKeypair() (crypto.PrivateKey, []byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ECDSA key: %w", err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal public key: %w", err)
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	})

	return key, pubPEM, nil
}

// CreateCSR creates a DER-encoded certificate signing request for an agent.
func CreateCSR(key crypto.PrivateKey, agentID, hostname string) ([]byte, error) {
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			Organization: []string{DefaultOrganization},
			CommonName:   agentID,
		},
	}

	if hostname != "" {
		template.DNSNames = []string{hostname}
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return nil, fmt.Errorf("create CSR: %w", err)
	}

	return csrDER, nil
}

// ParseCertificate parses a DER-encoded certificate.
func ParseCertificate(certDER []byte) (*x509.Certificate, error) {
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	return cert, nil
}

// CertificateExpiry returns the expiration time of a certificate.
func CertificateExpiry(cert *x509.Certificate) time.Time {
	return cert.NotAfter
}

// IsCertificateExpiringSoon returns true if the certificate expires within
// the given threshold duration.
func IsCertificateExpiringSoon(cert *x509.Certificate, threshold time.Duration) bool {
	return time.Until(cert.NotAfter) < threshold
}

// SavePEM writes DER data as a PEM file with the given type.
// Uses 0644 permissions for certificates, but callers should use 0600 for keys.
func SavePEM(path, pemType string, data []byte) error {
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  pemType,
		Bytes: data,
	})

	perm := os.FileMode(0o644)
	if pemType == "EC PRIVATE KEY" {
		perm = 0o600
	}

	if err := os.WriteFile(path, pemData, perm); err != nil {
		return fmt.Errorf("write PEM file %s: %w", path, err)
	}

	return nil
}

// LoadPEM reads a PEM file and returns the decoded DER bytes.
// Returns an error if the file does not contain a PEM block of the expected type.
func LoadPEM(path, pemType string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read PEM file %s: %w", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}

	if block.Type != pemType {
		return nil, fmt.Errorf("expected PEM type %q, got %q in %s", pemType, block.Type, path)
	}

	return block.Bytes, nil
}

// EncodeKeyPEM marshals an ECDSA private key to PEM-encoded bytes.
func EncodeKeyPEM(key crypto.PrivateKey) ([]byte, error) {
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected *ecdsa.PrivateKey, got %T", key)
	}

	der, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		return nil, fmt.Errorf("marshal EC private key: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}), nil
}

// DecodeKeyPEM decodes PEM-encoded bytes to an ECDSA private key.
func DecodeKeyPEM(pemData []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	if block.Type != "EC PRIVATE KEY" {
		return nil, fmt.Errorf("expected PEM type %q, got %q", "EC PRIVATE KEY", block.Type)
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse EC private key: %w", err)
	}

	return key, nil
}

// EncodeCertPEM encodes DER-encoded certificate bytes to PEM.
func EncodeCertPEM(certDER []byte) ([]byte, error) {
	if len(certDER) == 0 {
		return nil, fmt.Errorf("empty certificate DER data")
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}), nil
}
