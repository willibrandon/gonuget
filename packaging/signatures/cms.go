package signatures

import (
	"crypto"
	"encoding/asn1"
	"math/big"
)

// OID constants for CMS/PKCS#7 structures used in signature creation.
// These complement the OIDs defined in reader.go with those specific to signing operations.
var (
	// oidData is the CMS content-type for arbitrary data (RFC 5652).
	oidData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}

	// oidContentType is the PKCS#9 content-type authenticated attribute (RFC 2985).
	oidContentType = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}

	// oidMessageDigest is the PKCS#9 message-digest authenticated attribute (RFC 2985).
	oidMessageDigest = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}

	// oidSigningTime is the PKCS#9 signing-time authenticated attribute (RFC 2985).
	oidSigningTime = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}

	// oidSigningCertificateV2 is the ESS signing-certificate-v2 attribute (RFC 5035).
	oidSigningCertificateV2 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 47}

	// oidSHA256WithRSA is the RSA with SHA-256 signature algorithm (PKCS#1).
	oidSHA256WithRSA = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}

	// oidSHA384WithRSA is the RSA with SHA-384 signature algorithm (PKCS#1).
	oidSHA384WithRSA = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12}

	// oidSHA512WithRSA is the RSA with SHA-512 signature algorithm (PKCS#1).
	oidSHA512WithRSA = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13}
)

// ContentInfo represents the outer wrapper for CMS structures (RFC 5652).
// It encapsulates content with a content type identifier.
type ContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

// SigningCertificateV2 identifies the signing certificate using SHA-256 or stronger (RFC 5035).
// This attribute binds the signing certificate to the signature.
type SigningCertificateV2 struct {
	Certs []ESSCertIDv2 `asn1:"sequence"`
}

// ESSCertIDv2 identifies a certificate by its hash value (RFC 5035).
type ESSCertIDv2 struct {
	HashAlgorithm AlgorithmIdentifier `asn1:"optional"`
	CertHash      []byte
	IssuerSerial  IssuerSerial `asn1:"optional"`
}

// IssuerSerial identifies a certificate by issuer distinguished name and serial number.
type IssuerSerial struct {
	Issuer       []asn1.RawValue `asn1:"sequence"`
	SerialNumber *big.Int
}

// CommitmentTypeIndication represents a signer's commitment type (RFC 5126 Section 5.11.1).
// NuGet uses this to distinguish between Author and Repository signatures.
type CommitmentTypeIndication struct {
	CommitmentTypeID asn1.ObjectIdentifier
}

// getSignatureAlgorithmOID returns the OID for RSA signature algorithm with the given hash.
// Returns SHA256WithRSA as the default for unknown algorithms.
func getSignatureAlgorithmOID(hashAlg HashAlgorithmName) asn1.ObjectIdentifier {
	switch hashAlg {
	case HashAlgorithmSHA256:
		return oidSHA256WithRSA
	case HashAlgorithmSHA384:
		return oidSHA384WithRSA
	case HashAlgorithmSHA512:
		return oidSHA512WithRSA
	default:
		return oidSHA256WithRSA
	}
}

// getDigestAlgorithmOID returns the OID for the given hash algorithm.
// Returns SHA256 as the default for unknown algorithms.
func getDigestAlgorithmOID(hashAlg HashAlgorithmName) asn1.ObjectIdentifier {
	switch hashAlg {
	case HashAlgorithmSHA256:
		return oidSHA256
	case HashAlgorithmSHA384:
		return oidSHA384
	case HashAlgorithmSHA512:
		return oidSHA512
	default:
		return oidSHA256
	}
}

// getCryptoHash returns the crypto.Hash constant for the given hash algorithm name.
// Returns crypto.SHA256 as the default for unknown algorithms.
func getCryptoHash(hashAlg HashAlgorithmName) crypto.Hash {
	switch hashAlg {
	case HashAlgorithmSHA256:
		return crypto.SHA256
	case HashAlgorithmSHA384:
		return crypto.SHA384
	case HashAlgorithmSHA512:
		return crypto.SHA512
	default:
		return crypto.SHA256
	}
}
