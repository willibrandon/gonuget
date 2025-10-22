package signatures

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"slices"
	"time"
)

// SigningOptions configures NuGet package signature creation.
// It specifies the signing certificate, private key, certificate chain, signature type,
// hash algorithm, and optional timestamp authority settings.
type SigningOptions struct {
	Certificate      *x509.Certificate
	PrivateKey       crypto.PrivateKey
	CertificateChain []*x509.Certificate
	SignatureType    SignatureType
	HashAlgorithm    HashAlgorithmName
	TimestampURL     string
	TimestampTimeout time.Duration
}

// DefaultSigningOptions returns signing options with sensible defaults.
// It creates an Author signature using SHA256 hash algorithm with a 30-second timeout
// for timestamp requests. No timestamp URL is configured by default.
func DefaultSigningOptions(cert *x509.Certificate, key crypto.PrivateKey) SigningOptions {
	return SigningOptions{
		Certificate:      cert,
		PrivateKey:       key,
		SignatureType:    SignatureTypeAuthor,
		HashAlgorithm:    HashAlgorithmSHA256,
		TimestampTimeout: 30 * time.Second,
	}
}

// Validate checks that signing options are valid and secure.
// It verifies that required fields are set, the private key matches the certificate,
// RSA keys are at least 2048 bits, and the hash algorithm is SHA256, SHA384, or SHA512.
// Returns an error if any validation check fails.
func (opts *SigningOptions) Validate() error {
	if opts.Certificate == nil {
		return fmt.Errorf("signing certificate is required")
	}

	if opts.PrivateKey == nil {
		return fmt.Errorf("private key is required")
	}

	// Verify key matches certificate
	if err := verifyKeyMatchesCertificate(opts.Certificate, opts.PrivateKey); err != nil {
		return fmt.Errorf("key does not match certificate: %w", err)
	}

	// Verify RSA key length
	if rsaKey, ok := opts.PrivateKey.(*rsa.PrivateKey); ok {
		if rsaKey.N.BitLen() < 2048 {
			return fmt.Errorf("RSA key must be at least 2048 bits")
		}
	}

	// Verify hash algorithm
	allowedAlgos := []HashAlgorithmName{HashAlgorithmSHA256, HashAlgorithmSHA384, HashAlgorithmSHA512}
	if !slices.Contains(allowedAlgos, opts.HashAlgorithm) {
		return fmt.Errorf("hash algorithm %s is not allowed", opts.HashAlgorithm)
	}

	return nil
}

// verifyKeyMatchesCertificate checks that the private key corresponds to the certificate's public key.
// Currently only RSA keys are supported. Returns an error if keys don't match or key type is unsupported.
func verifyKeyMatchesCertificate(cert *x509.Certificate, key crypto.PrivateKey) error {
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		priv, ok := key.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("certificate has RSA public key but private key is not RSA")
		}
		if pub.N.Cmp(priv.N) != 0 {
			return fmt.Errorf("public/private key mismatch")
		}
	default:
		return fmt.Errorf("unsupported key type")
	}

	return nil
}

// SignPackageData creates a PKCS#7/CMS signature for NuGet package content (RFC 5652).
// It generates a detached SignedData structure containing authenticated attributes
// (content-type, signing-time, message-digest, commitment-type, signing-certificate-v2)
// and optionally requests an RFC 3161 timestamp if TimestampURL is configured.
// The contentHash should be the SHA256/384/512 hash of the package ZIP archive.
// Returns the DER-encoded PKCS#7 signature bytes ready to be stored in the package.
func SignPackageData(contentHash []byte, opts SigningOptions) ([]byte, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid signing options: %w", err)
	}

	// Build SignedData structure
	signedData, err := createSignedData(contentHash, opts)
	if err != nil {
		return nil, fmt.Errorf("create signed data: %w", err)
	}

	// Encode SignedData
	signedDataBytes, err := asn1.Marshal(*signedData)
	if err != nil {
		return nil, fmt.Errorf("marshal signed data: %w", err)
	}

	// Wrap in ContentInfo
	contentInfo := ContentInfo{
		ContentType: oidSignedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      signedDataBytes,
		},
	}

	// Encode final PKCS#7 signature
	pkcs7Signature, err := asn1.Marshal(contentInfo)
	if err != nil {
		return nil, fmt.Errorf("marshal content info: %w", err)
	}

	return pkcs7Signature, nil
}

// createSignedData builds the RFC 5652 SignedData structure for a NuGet package signature.
// It creates a detached signature (EncapsulatedContentInfo has no content), includes the signing
// certificate and chain, specifies the digest algorithm, and builds the SignerInfo with
// authenticated and optionally unsigned (timestamp) attributes.
func createSignedData(contentHash []byte, opts SigningOptions) (*SignedData, error) {
	// 1. Build EncapsulatedContentInfo (package hash as data)
	encapContentInfo := EncapsulatedContentInfo{
		ContentType: oidData,
		// Content is OPTIONAL - for detached signatures, we omit it
	}

	// 2. Build DigestAlgorithms SET
	digestAlgOID := getDigestAlgorithmOID(opts.HashAlgorithm)
	digestAlgorithms := []AlgorithmIdentifier{
		{Algorithm: digestAlgOID},
	}

	// 3. Build certificates SET
	var certBytes []byte
	certs := []*x509.Certificate{opts.Certificate}
	certs = append(certs, opts.CertificateChain...)

	for _, cert := range certs {
		certBytes = append(certBytes, cert.Raw...)
	}

	certificates := asn1.RawValue{
		Class:      asn1.ClassContextSpecific,
		Tag:        0,
		IsCompound: true,
		Bytes:      certBytes,
	}

	// 4. Build SignerInfo
	signerInfo, err := createSignerInfo(contentHash, opts)
	if err != nil {
		return nil, fmt.Errorf("create signer info: %w", err)
	}

	// 5. Assemble SignedData
	signedData := &SignedData{
		Version:          1,
		DigestAlgorithms: digestAlgorithms,
		ContentInfo:      encapContentInfo,
		Certificates:     certificates,
		SignerInfos:      []SignerInfo{*signerInfo},
	}

	return signedData, nil
}

// createSignerInfo builds the SignerInfo structure for a signer (RFC 5652 Section 5.3).
// It creates the signer identifier (SubjectKeyIdentifier or IssuerAndSerialNumber),
// builds authenticated attributes, signs them with the private key using RSA-PKCS#1 v1.5,
// and optionally adds timestamp unsigned attributes if a timestamp URL is configured.
func createSignerInfo(contentHash []byte, opts SigningOptions) (*SignerInfo, error) {
	// 1. Build SignerIdentifier (use IssuerAndSerialNumber or SubjectKeyIdentifier)
	var sid asn1.RawValue

	// Check if certificate has SubjectKeyId extension
	if len(opts.Certificate.SubjectKeyId) > 0 {
		// Use SubjectKeyIdentifier [0] IMPLICIT
		sid = asn1.RawValue{
			Class: asn1.ClassContextSpecific,
			Tag:   0,
			Bytes: opts.Certificate.SubjectKeyId,
		}
	} else {
		// Use IssuerAndSerialNumber
		issuerAndSerial := IssuerAndSerialNumber{
			Issuer:       asn1.RawValue{FullBytes: opts.Certificate.RawIssuer},
			SerialNumber: asn1.RawValue{FullBytes: opts.Certificate.SerialNumber.Bytes()},
		}
		sidBytes, err := asn1.Marshal(issuerAndSerial)
		if err != nil {
			return nil, fmt.Errorf("marshal issuer and serial: %w", err)
		}
		sid = asn1.RawValue{FullBytes: sidBytes}
	}

	// 2. Build signed attributes
	signedAttrs, err := BuildSignedAttributes(
		contentHash,
		opts.SignatureType,
		opts.Certificate,
		opts.HashAlgorithm,
	)
	if err != nil {
		return nil, fmt.Errorf("build signed attributes: %w", err)
	}

	// 3. Encode signed attributes for signing
	signedAttrsBytes, err := EncodeAttributesForSigning(signedAttrs)
	if err != nil {
		return nil, fmt.Errorf("encode signed attributes: %w", err)
	}

	// 4. Sign the encoded attributes
	signature, err := signAttributes(signedAttrsBytes, opts)
	if err != nil {
		return nil, fmt.Errorf("sign attributes: %w", err)
	}

	// 5. Build SignerInfo
	digestAlgOID := getDigestAlgorithmOID(opts.HashAlgorithm)
	signatureAlgOID := getSignatureAlgorithmOID(opts.HashAlgorithm)

	signerInfo := &SignerInfo{
		Version: 1,
		SID:     sid,
		DigestAlgorithm: AlgorithmIdentifier{
			Algorithm: digestAlgOID,
		},
		SignedAttrs: createSignedAttrsRawValue(signedAttrsBytes),
		SignatureAlgorithm: AlgorithmIdentifier{
			Algorithm: signatureAlgOID,
		},
		Signature: signature,
	}

	// 6. Add timestamp to unsigned attributes (if TimestampURL provided)
	// Matches NuGet.Client behavior: X509SignatureProvider.cs:51-58
	// Timestamps are optional - only added when TimestampURL is configured
	if opts.TimestampURL != "" {
		timestampAttr, err := createTimestampAttribute(signature, opts)
		if err != nil {
			return nil, fmt.Errorf("create timestamp: %w", err)
		}

		// Encode unsigned attributes as SET
		unsignedAttrsBytes, err := asn1.MarshalWithParams([]Attribute{timestampAttr}, "set")
		if err != nil {
			return nil, fmt.Errorf("marshal unsigned attributes: %w", err)
		}

		// Parse to extract content bytes for [1] IMPLICIT
		var raw asn1.RawValue
		if _, err := asn1.Unmarshal(unsignedAttrsBytes, &raw); err != nil {
			return nil, fmt.Errorf("unmarshal unsigned attributes: %w", err)
		}

		signerInfo.UnsignedAttrs = asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        1,
			IsCompound: true,
			Bytes:      raw.Bytes, // SET content (tag+length stripped by Unmarshal)
		}
	}

	return signerInfo, nil
}

// signAttributes signs the DER-encoded authenticated attributes using RSA-PKCS#1 v1.5.
// It hashes the attributes with the configured hash algorithm (SHA256/384/512) and
// signs the digest using the private key. This matches NuGet.Client signing behavior.
// Returns the signature bytes.
func signAttributes(attributesBytes []byte, opts SigningOptions) ([]byte, error) {
	// Sign using RSA-PKCS#1 v1.5 (matches NuGet.Client behavior)
	rsaKey, ok := opts.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("only RSA keys are supported")
	}

	// Hash the attributes
	h := getCryptoHash(opts.HashAlgorithm)
	hasher := h.New()
	hasher.Write(attributesBytes)
	digest := hasher.Sum(nil)

	// Sign with RSA-PKCS#1 v1.5
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, h, digest)
	if err != nil {
		return nil, fmt.Errorf("RSA sign: %w", err)
	}

	return signature, nil
}

// createTimestampAttribute requests an RFC 3161 timestamp token from a timestamp authority
// and creates the unsigned timestamp attribute to be added to SignerInfo.
// It hashes the signature bytes and sends a timestamp request to the configured TimestampURL.
// This function is only called when opts.TimestampURL is non-empty.
// Returns an Attribute with type oidTimestampToken containing the timestamp response.
// Matches NuGet.Client behavior: X509SignatureProvider.TimestampPrimarySignatureAsync
func createTimestampAttribute(signature []byte, opts SigningOptions) (Attribute, error) {
	// Request RFC 3161 timestamp token from TSA
	client := NewTimestampClient(opts.TimestampURL, opts.TimestampTimeout)

	// Hash the signature
	h := getCryptoHash(opts.HashAlgorithm)
	hasher := h.New()
	hasher.Write(signature)
	signatureHash := hasher.Sum(nil)

	// Request timestamp token
	timestampToken, err := client.RequestTimestamp(signatureHash, opts.HashAlgorithm)
	if err != nil {
		return Attribute{}, fmt.Errorf("request timestamp: %w", err)
	}

	// Timestamp token is already a ContentInfo, just wrap it in a SET
	values, err := asn1.Marshal([]asn1.RawValue{{FullBytes: timestampToken}})
	if err != nil {
		return Attribute{}, err
	}

	// Parse back to RawValue to ensure both Bytes and FullBytes are set correctly
	var valuesRaw asn1.RawValue
	if _, err := asn1.Unmarshal(values, &valuesRaw); err != nil {
		return Attribute{}, err
	}

	return Attribute{
		Type:   oidTimestampToken,
		Values: valuesRaw,
	}, nil
}

// createSignedAttrsRawValue prepares authenticated attributes for encoding in SignerInfo.
// It extracts the content bytes from a DER-encoded SET OF Attributes and wraps them with
// [0] IMPLICIT context-specific tagging as required by RFC 5652 SignerInfo.signedAttrs.
// The SET tag and length are stripped because [0] IMPLICIT replaces the outer tag.
// Returns an asn1.RawValue ready to be marshaled into SignerInfo.
func createSignedAttrsRawValue(signedAttrsBytes []byte) asn1.RawValue {
	// Parse the SET to extract its content bytes
	var raw asn1.RawValue
	if _, err := asn1.Unmarshal(signedAttrsBytes, &raw); err != nil {
		// Fallback: just use the bytes as-is
		return asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      signedAttrsBytes[1:], // Simple: skip SET tag byte
		}
	}
	// Return content bytes with proper [0] IMPLICIT tagging
	return asn1.RawValue{
		Class:      asn1.ClassContextSpecific,
		Tag:        0,
		IsCompound: true,
		Bytes:      raw.Bytes, // SET content (tag+length already stripped by Unmarshal)
	}
}
