package main

import (
	"crypto"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/willibrandon/gonuget/packaging/signatures"
)

// SignPackageHandler creates a package signature using gonuget.
type SignPackageHandler struct{}

func (h *SignPackageHandler) ErrorCode() string { return "SIGN_001" }

func (h *SignPackageHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req SignPackageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if len(req.PackageHash) == 0 {
		return nil, fmt.Errorf("packageHash is required")
	}
	if req.CertPath == "" {
		return nil, fmt.Errorf("certPath is required")
	}
	if req.SignatureType == "" {
		return nil, fmt.Errorf("signatureType is required")
	}
	if req.HashAlgorithm == "" {
		return nil, fmt.Errorf("hashAlgorithm is required")
	}

	// Parse signature type
	var sigType signatures.SignatureType
	switch req.SignatureType {
	case "Author":
		sigType = signatures.SignatureTypeAuthor
	case "Repository":
		sigType = signatures.SignatureTypeRepository
	default:
		return nil, fmt.Errorf("invalid signatureType: %s (must be 'Author' or 'Repository')", req.SignatureType)
	}

	// Parse hash algorithm
	var hashAlgo signatures.HashAlgorithmName
	switch req.HashAlgorithm {
	case "SHA256":
		hashAlgo = signatures.HashAlgorithmSHA256
	case "SHA384":
		hashAlgo = signatures.HashAlgorithmSHA384
	case "SHA512":
		hashAlgo = signatures.HashAlgorithmSHA512
	default:
		return nil, fmt.Errorf("invalid hashAlgorithm: %s (must be 'SHA256', 'SHA384', or 'SHA512')", req.HashAlgorithm)
	}

	// Load certificate
	cert, err := loadCertificate(req.CertPath, req.CertPassword)
	if err != nil {
		return nil, fmt.Errorf("load certificate: %w", err)
	}

	// Load private key
	var privateKey interface{}
	if req.KeyPath != "" {
		// Load from separate key file
		privateKey, err = loadPrivateKey(req.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("load private key: %w", err)
		}
	} else if req.CertPassword != "" {
		// Extract from PFX
		privateKey, err = loadPrivateKeyFromPFX(req.CertPath, req.CertPassword)
		if err != nil {
			return nil, fmt.Errorf("load private key from PFX: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either keyPath or certPassword (for PFX) must be provided")
	}

	// Create signing options
	opts := signatures.SigningOptions{
		Certificate:   cert,
		PrivateKey:    privateKey,
		SignatureType: sigType,
		HashAlgorithm: hashAlgo,
		TimestampURL:  req.TimestampURL,
	}

	// Create signature using gonuget API
	sigBytes, err := signatures.SignPackageData(req.PackageHash, opts)
	if err != nil {
		return nil, fmt.Errorf("create signature: %w", err)
	}

	return &SignPackageResponse{
		Signature: sigBytes,
	}, nil
}

// ParseSignatureHandler parses a signature and extracts metadata.
type ParseSignatureHandler struct{}

func (h *ParseSignatureHandler) ErrorCode() string { return "PARSE_001" }

func (h *ParseSignatureHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ParseSignatureRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if len(req.Signature) == 0 {
		return nil, fmt.Errorf("signature is required")
	}

	// Parse signature using gonuget API
	sig, err := signatures.ReadSignature(req.Signature)
	if err != nil {
		return nil, fmt.Errorf("parse signature: %w", err)
	}

	// Extract signature type
	var sigType string
	switch sig.Type {
	case signatures.SignatureTypeAuthor:
		sigType = "Author"
	case signatures.SignatureTypeRepository:
		sigType = "Repository"
	default:
		sigType = "Unknown"
	}

	// Extract hash algorithm
	var hashAlgo string
	switch sig.HashAlgorithm {
	case signatures.HashAlgorithmSHA256:
		hashAlgo = "SHA256"
	case signatures.HashAlgorithmSHA384:
		hashAlgo = "SHA384"
	case signatures.HashAlgorithmSHA512:
		hashAlgo = "SHA512"
	default:
		hashAlgo = "Unknown"
	}

	// Extract signer certificate hash
	var signerCertHash string
	if sig.SignerCertificate != nil {
		// Use SubjectKeyId if available, otherwise use SHA-256 of certificate
		if len(sig.SignerCertificate.SubjectKeyId) > 0 {
			signerCertHash = hex.EncodeToString(sig.SignerCertificate.SubjectKeyId)
		} else {
			hash := crypto.SHA256.New()
			hash.Write(sig.SignerCertificate.Raw)
			signerCertHash = hex.EncodeToString(hash.Sum(nil))
		}
	}

	// Extract timestamps
	var timestampTimes []string
	for _, ts := range sig.Timestamps {
		timestampTimes = append(timestampTimes, ts.Time.Format(time.RFC3339))
	}

	// Count certificates in chain
	certCount := len(sig.Certificates)

	return &ParseSignatureResponse{
		Type:           sigType,
		HashAlgorithm:  hashAlgo,
		SignerCertHash: signerCertHash,
		TimestampCount: len(sig.Timestamps),
		TimestampTimes: timestampTimes,
		Certificates:   certCount,
	}, nil
}

// VerifySignatureHandler validates a signature.
type VerifySignatureHandler struct{}

func (h *VerifySignatureHandler) ErrorCode() string { return "VERIFY_001" }

func (h *VerifySignatureHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req VerifySignatureRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if len(req.Signature) == 0 {
		return nil, fmt.Errorf("signature is required")
	}

	// Parse signature using gonuget API
	sig, err := signatures.ReadSignature(req.Signature)
	if err != nil {
		return nil, fmt.Errorf("parse signature: %w", err)
	}

	// Build verification options
	opts := signatures.DefaultVerificationOptions()
	opts.AllowUntrustedRoot = req.AllowUntrustedRoot
	opts.RequireTimestamp = req.RequireTimestamp

	// Add trusted roots if provided
	if len(req.TrustedRoots) > 0 {
		trustStore := signatures.NewTrustStore()
		for i, rootDER := range req.TrustedRoots {
			cert, err := x509.ParseCertificate(rootDER)
			if err != nil {
				return nil, fmt.Errorf("parse trusted root %d: %w", i, err)
			}
			trustStore.AddCertificate(cert)
		}
		opts.TrustStore = trustStore
	}

	// Verify signature
	result := signatures.VerifySignature(sig, opts)

	// Build response
	resp := &VerifySignatureResponse{
		Valid: result.IsValid,
	}

	// Extract errors and warnings
	for _, err := range result.Errors {
		resp.Errors = append(resp.Errors, err.Error())
	}
	for _, warn := range result.Warnings {
		resp.Warnings = append(resp.Warnings, warn)
	}

	// Extract signer subject if available
	if sig.SignerCertificate != nil {
		resp.SignerSubject = sig.SignerCertificate.Subject.String()
	}

	return resp, nil
}
