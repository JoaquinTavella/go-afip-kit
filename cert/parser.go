// Package cert provides utilities for parsing, validating, and encrypting
// X.509 certificates and RSA private keys used with AFIP web services.
package cert

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// ParseCertificatePEM parses a PEM-encoded X.509 certificate.
func ParseCertificatePEM(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	return cert, nil
}

// ParsePrivateKeyPEM parses a PEM-encoded RSA private key.
// Supports both PKCS#1 (RSA PRIVATE KEY) and PKCS#8 (PRIVATE KEY) formats.
func ParsePrivateKeyPEM(keyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in private key")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS#1 private key: %w", err)
		}
		return key, nil

	case "PRIVATE KEY":
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS#8 private key: %w", err)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key is not RSA")
		}
		return rsaKey, nil

	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
}

// ParseCertAndKey parses both certificate and private key from PEM bytes.
func ParseCertAndKey(certPEM, keyPEM []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	cert, err := ParseCertificatePEM(certPEM)
	if err != nil {
		return nil, nil, err
	}
	key, err := ParsePrivateKeyPEM(keyPEM)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}
