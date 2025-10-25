package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"software.sslmate.com/src/go-pkcs12"
)

// loadCertificate loads a certificate from PEM, DER, or PFX format.
// For PFX files, password may be required.
func loadCertificate(path, password string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Try PEM format first
	block, _ := pem.Decode(data)
	if block != nil && block.Type == "CERTIFICATE" {
		return x509.ParseCertificate(block.Bytes)
	}

	// Try PFX format
	if password != "" || isPFX(data) {
		_, cert, _, err := pkcs12.DecodeChain(data, password)
		if err != nil {
			return nil, fmt.Errorf("decode PFX: %w", err)
		}
		return cert, nil
	}

	// Try DER format
	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, fmt.Errorf("parse certificate (tried PEM, PFX, DER): %w", err)
	}
	return cert, nil
}

// loadPrivateKey loads a private key from PEM format.
// Supports RSA, ECDSA, and Ed25519 keys in PKCS#8 format.
func loadPrivateKey(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Parse PEM block
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	// Try PKCS#8 format (most common)
	if block.Type == "PRIVATE KEY" {
		return x509.ParsePKCS8PrivateKey(block.Bytes)
	}

	// Try legacy RSA format
	if block.Type == "RSA PRIVATE KEY" {
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}

	// Try EC format
	if block.Type == "EC PRIVATE KEY" {
		return x509.ParseECPrivateKey(block.Bytes)
	}

	return nil, fmt.Errorf("unsupported key type: %s", block.Type)
}

// loadPrivateKeyFromPFX loads private key from PFX file.
func loadPrivateKeyFromPFX(path, password string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	key, _, _, err := pkcs12.DecodeChain(data, password)
	if err != nil {
		return nil, fmt.Errorf("decode PFX: %w", err)
	}

	return key, nil
}

// isPFX checks if data looks like a PFX file.
// PFX files start with the PKCS#12 magic bytes.
func isPFX(data []byte) bool {
	// PKCS#12 magic: 0x30 (SEQUENCE)
	if len(data) < 4 {
		return false
	}
	return data[0] == 0x30
}
