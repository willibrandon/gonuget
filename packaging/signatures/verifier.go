package signatures

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"slices"
	"time"
)

// VerificationOptions configures signature verification
type VerificationOptions struct {
	// TrustStore contains trusted root certificates
	TrustStore *TrustStore

	// AllowUntrustedRoot allows signatures with untrusted roots
	AllowUntrustedRoot bool

	// RequireTimestamp requires signatures to be timestamped
	RequireTimestamp bool

	// VerifyTimestamp enables timestamp verification
	VerifyTimestamp bool

	// AllowedSignatureTypes restricts signature types
	AllowedSignatureTypes []SignatureType

	// AllowedHashAlgorithms restricts hash algorithms
	AllowedHashAlgorithms []HashAlgorithmName

	// VerificationTime is the time at which to verify (defaults to Now)
	VerificationTime *time.Time
}

// DefaultVerificationOptions returns secure default options
func DefaultVerificationOptions() VerificationOptions {
	return VerificationOptions{
		TrustStore:            NewTrustStore(),
		AllowUntrustedRoot:    false,
		RequireTimestamp:      false,
		VerifyTimestamp:       true,
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor, SignatureTypeRepository},
		AllowedHashAlgorithms: []HashAlgorithmName{
			HashAlgorithmSHA256,
			HashAlgorithmSHA384,
			HashAlgorithmSHA512,
		},
	}
}

// VerificationResult contains verification results
type VerificationResult struct {
	// IsValid indicates if signature is valid
	IsValid bool

	// Errors contains any verification errors
	Errors []error

	// Warnings contains non-fatal warnings
	Warnings []string

	// SignatureType is the verified signature type
	SignatureType SignatureType

	// SignerCertificate is the verified signer certificate
	SignerCertificate *x509.Certificate

	// TrustedRoot is the trusted root certificate (if found)
	TrustedRoot *x509.Certificate

	// TimestampValid indicates if timestamp is valid
	TimestampValid bool

	// SigningTime is the verified signing time (from timestamp)
	SigningTime *time.Time
}

// VerifySignature verifies a package signature
func VerifySignature(sig *PrimarySignature, opts VerificationOptions) VerificationResult {
	result := VerificationResult{
		IsValid:       true,
		SignatureType: sig.Type,
	}

	// Verify signature type is allowed
	if !isSignatureTypeAllowed(sig.Type, opts.AllowedSignatureTypes) {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Errorf("signature type %s is not allowed", sig.Type))
		return result
	}

	// Verify hash algorithm is allowed
	// Reference: SigningSpecificationsV1.cs allowed algorithms
	if !isHashAlgorithmAllowed(sig.HashAlgorithm, opts.AllowedHashAlgorithms) {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Errorf("hash algorithm %s is not allowed", sig.HashAlgorithm))
		return result
	}

	// Verify certificate chain
	chainResult := verifyCertificateChain(sig, opts)
	result.SignerCertificate = chainResult.SignerCertificate
	result.TrustedRoot = chainResult.TrustedRoot

	if !chainResult.IsValid {
		result.IsValid = false
		result.Errors = append(result.Errors, chainResult.Errors...)

		if !opts.AllowUntrustedRoot {
			return result
		}

		// Continue with untrusted root if allowed
		result.Warnings = append(result.Warnings, "Signature has untrusted root certificate")
	}

	// Verify timestamp if present
	if len(sig.Timestamps) > 0 {
		tsResult := verifyTimestamp(sig.Timestamps[0], opts)
		result.TimestampValid = tsResult.IsValid
		result.SigningTime = &tsResult.SigningTime

		if !tsResult.IsValid {
			if opts.RequireTimestamp {
				result.IsValid = false
				result.Errors = append(result.Errors, fmt.Errorf("timestamp verification failed"))
			} else {
				result.Warnings = append(result.Warnings, "Timestamp verification failed but not required")
			}
		}
	} else if opts.RequireTimestamp {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Errorf("signature does not have a timestamp"))
	}

	// Verify RSA key length (minimum 2048 bits)
	// Reference: SigningSpecificationsV1.cs RSA minimum 2048 bits
	if err := verifySignerKeyLength(sig.SignerCertificate); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, err)
	}

	return result
}

func isSignatureTypeAllowed(sigType SignatureType, allowed []SignatureType) bool {
	return slices.Contains(allowed, sigType)
}

func isHashAlgorithmAllowed(hashAlg HashAlgorithmName, allowed []HashAlgorithmName) bool {
	return slices.Contains(allowed, hashAlg)
}

func verifySignerKeyLength(cert *x509.Certificate) error {
	// RSA minimum 2048 bits
	// Reference: SigningSpecificationsV1.cs
	if cert.PublicKeyAlgorithm == x509.RSA {
		rsaPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("invalid RSA public key")
		}

		if rsaPubKey.N.BitLen() < 2048 {
			return fmt.Errorf("RSA key length %d is less than minimum 2048 bits", rsaPubKey.N.BitLen())
		}
	}

	return nil
}

// CertificateChainResult represents certificate chain verification result
type CertificateChainResult struct {
	IsValid           bool
	SignerCertificate *x509.Certificate
	TrustedRoot       *x509.Certificate
	Errors            []error
}

func verifyCertificateChain(sig *PrimarySignature, opts VerificationOptions) CertificateChainResult {
	result := CertificateChainResult{
		IsValid:           true,
		SignerCertificate: sig.SignerCertificate,
	}

	if sig.SignerCertificate == nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Errorf("signer certificate not found"))
		return result
	}

	// Build certificate pool from signature
	intermediates := x509.NewCertPool()
	for _, cert := range sig.Certificates {
		if cert != sig.SignerCertificate {
			intermediates.AddCert(cert)
		}
	}

	// Get verification time
	verifyTime := time.Now()
	if opts.VerificationTime != nil {
		verifyTime = *opts.VerificationTime
	}

	// Verify chain
	verifyOpts := x509.VerifyOptions{
		Intermediates: intermediates,
		CurrentTime:   verifyTime,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}

	// Add roots from trust store
	if opts.TrustStore != nil {
		verifyOpts.Roots = opts.TrustStore.GetRootPool()
	}

	chains, err := sig.SignerCertificate.Verify(verifyOpts)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Errorf("certificate chain verification failed: %w", err))
		return result
	}

	// Get trusted root from first valid chain
	if len(chains) > 0 && len(chains[0]) > 0 {
		result.TrustedRoot = chains[0][len(chains[0])-1]
	}

	return result
}

// TimestampResult represents timestamp verification result
type TimestampResult struct {
	IsValid     bool
	SigningTime time.Time
	Errors      []error
}

func verifyTimestamp(ts Timestamp, opts VerificationOptions) TimestampResult {
	result := TimestampResult{
		IsValid:     true,
		SigningTime: ts.Time,
	}

	// Verify TSA certificate
	if ts.SignerCertificate == nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Errorf("timestamp authority certificate not found"))
		return result
	}

	// Verify TSA certificate chain
	tsaOpts := x509.VerifyOptions{
		CurrentTime: ts.Time,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	}

	if opts.TrustStore != nil {
		tsaOpts.Roots = opts.TrustStore.GetRootPool()
	}

	_, err := ts.SignerCertificate.Verify(tsaOpts)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Errorf("TSA certificate verification failed: %w", err))
	}

	return result
}
