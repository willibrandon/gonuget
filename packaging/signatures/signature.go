// Package signatures provides PKCS#7/CMS signature reading and parsing for NuGet packages.
//
// This package implements RFC 5652 (Cryptographic Message Syntax) and RFC 3161 (Time-Stamp Protocol)
// to read and verify package signatures from signed .nupkg files. It supports both Author and
// Repository signatures, certificate chain extraction, and RFC 3161 timestamp validation.
package signatures

import (
	"crypto/x509"
	"encoding/asn1"
	"time"
)

// SignatureType indicates the type of signature
type SignatureType string

const (
	// SignatureTypeAuthor indicates an author signature
	SignatureTypeAuthor SignatureType = "Author"

	// SignatureTypeRepository indicates a repository signature
	SignatureTypeRepository SignatureType = "Repository"

	// SignatureTypeUnknown indicates an unknown signature type
	SignatureTypeUnknown SignatureType = "Unknown"
)

// PrimarySignature represents the primary package signature
type PrimarySignature struct {
	// Raw PKCS#7 data
	RawData []byte

	// Parsed CMS structure
	SignedData *SignedData

	// Signature type (Author or Repository)
	Type SignatureType

	// Signer certificate
	SignerCertificate *x509.Certificate

	// Certificate chain
	Certificates []*x509.Certificate

	// Timestamp information (RFC 3161)
	Timestamps []Timestamp

	// Content hash algorithm
	HashAlgorithm HashAlgorithmName
}

// Timestamp represents an RFC 3161 timestamp
type Timestamp struct {
	// Timestamp value
	Time time.Time

	// Timestamp authority certificate
	SignerCertificate *x509.Certificate

	// Hash algorithm used
	HashAlgorithm HashAlgorithmName

	// Accuracy (optional)
	Accuracy time.Duration
}

// HashAlgorithmName represents cryptographic hash algorithms
type HashAlgorithmName string

const (
	// HashAlgorithmSHA256 represents the SHA-256 hash algorithm.
	HashAlgorithmSHA256 HashAlgorithmName = "SHA256"
	// HashAlgorithmSHA384 represents the SHA-384 hash algorithm.
	HashAlgorithmSHA384 HashAlgorithmName = "SHA384"
	// HashAlgorithmSHA512 represents the SHA-512 hash algorithm.
	HashAlgorithmSHA512 HashAlgorithmName = "SHA512"
)

// SignedData represents CMS SignedData structure (RFC 5652)
type SignedData struct {
	Version          int                   `asn1:"default:1"`
	DigestAlgorithms []AlgorithmIdentifier `asn1:"set"`
	ContentInfo      EncapsulatedContentInfo
	Certificates     asn1.RawValue `asn1:"optional,tag:0"`
	CRLs             asn1.RawValue `asn1:"optional,tag:1"`
	SignerInfos      []SignerInfo  `asn1:"set"`
}

// EncapsulatedContentInfo represents the signed content
type EncapsulatedContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"optional,explicit,tag:0"`
}

// SignerInfo represents signer information (RFC 5652)
type SignerInfo struct {
	Version            int           `asn1:"default:1"`
	SID                asn1.RawValue // SignerIdentifier (CHOICE)
	DigestAlgorithm    AlgorithmIdentifier
	SignedAttrs        asn1.RawValue `asn1:"optional,tag:0"`
	SignatureAlgorithm AlgorithmIdentifier
	Signature          []byte
	UnsignedAttrs      asn1.RawValue `asn1:"optional,tag:1"`
}

// IssuerAndSerialNumber identifies a certificate
type IssuerAndSerialNumber struct {
	Issuer       asn1.RawValue
	SerialNumber asn1.RawValue
}

// AlgorithmIdentifier represents an algorithm
type AlgorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

// Attribute represents a CMS attribute
type Attribute struct {
	Type   asn1.ObjectIdentifier
	Values asn1.RawValue `asn1:"set"`
}
