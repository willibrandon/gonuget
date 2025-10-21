package signatures

import (
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"math/big"
	"time"
)

// OID constants from RFC 5652 and NuGet specifications
var (
	// RFC 5652 - SignedData content type
	oidSignedData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}

	// RFC 5126 - Commitment Type Indication
	oidCommitmentTypeIndication = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 16}
	oidAuthorSignature          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 6, 1} // ProofOfOrigin
	oidRepositorySignature      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 6, 2} // ProofOfReceipt

	// RFC 3161 - Timestamp token OID
	oidTimestampToken = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 14}

	// Hash algorithm OIDs
	oidSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	oidSHA384 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
	oidSHA512 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}
)

// ReadSignature reads and parses a PKCS#7/CMS signature
func ReadSignature(signatureData []byte) (*PrimarySignature, error) {
	if len(signatureData) == 0 {
		return nil, fmt.Errorf("signature data is empty")
	}

	sig := &PrimarySignature{
		RawData: signatureData,
	}

	// Parse outer ContentInfo wrapper
	var contentInfo struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue `asn1:"explicit,tag:0"`
	}

	rest, err := asn1.Unmarshal(signatureData, &contentInfo)
	if err != nil {
		return nil, fmt.Errorf("unmarshal content info: %w", err)
	}
	if len(rest) > 0 {
		return nil, fmt.Errorf("trailing data after content info")
	}

	// Verify this is SignedData
	if !contentInfo.ContentType.Equal(oidSignedData) {
		return nil, fmt.Errorf("not a SignedData structure (got OID %v)", contentInfo.ContentType)
	}

	// Parse SignedData
	var signedData SignedData
	if _, err := asn1.Unmarshal(contentInfo.Content.Bytes, &signedData); err != nil {
		return nil, fmt.Errorf("unmarshal signed data: %w", err)
	}
	sig.SignedData = &signedData

	// Extract certificates
	certs, err := parseCertificates(signedData.Certificates)
	if err != nil {
		return nil, fmt.Errorf("parse certificates: %w", err)
	}
	sig.Certificates = certs

	// Get first signer (NuGet packages have exactly one)
	if len(signedData.SignerInfos) == 0 {
		return nil, fmt.Errorf("no signer infos found")
	}
	signerInfo := signedData.SignerInfos[0]

	// Find signer certificate
	signerCert, err := findSignerCertificate(signerInfo, certs)
	if err != nil {
		return nil, fmt.Errorf("find signer certificate: %w", err)
	}
	sig.SignerCertificate = signerCert

	// Determine signature type from signed attributes
	sig.Type = determineSignatureType(signerInfo)

	// Extract hash algorithm
	sig.HashAlgorithm = oidToHashAlgorithm(signerInfo.DigestAlgorithm.Algorithm)

	// Extract timestamps from unsigned attributes
	timestamps, _ := extractTimestamps(signerInfo)
	sig.Timestamps = timestamps

	return sig, nil
}

// parseCertificates extracts X.509 certificates from the raw value
func parseCertificates(certData asn1.RawValue) ([]*x509.Certificate, error) {
	if len(certData.Bytes) == 0 {
		return []*x509.Certificate{}, nil
	}

	// Certificates are in a SET, parse and extract
	certs, err := x509.ParseCertificates(certData.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse x509 certificates: %w", err)
	}

	return certs, nil
}

// findSignerCertificate matches the signer info to a certificate
func findSignerCertificate(signerInfo SignerInfo, certs []*x509.Certificate) (*x509.Certificate, error) {
	// SignerIdentifier is a CHOICE:
	// - IssuerAndSerialNumber (SEQUENCE)
	// - subjectKeyIdentifier [0] IMPLICIT (20-byte hash)

	// Check if it's subjectKeyIdentifier [0]
	if signerInfo.SID.Tag == 0 && signerInfo.SID.Class == 2 {
		// subjectKeyIdentifier is a 20-byte OCTET STRING
		subjectKeyID := signerInfo.SID.Bytes

		// Match against certificate's SubjectKeyId extension
		for _, cert := range certs {
			if len(cert.SubjectKeyId) > 0 && bytesEqual(cert.SubjectKeyId, subjectKeyID) {
				return cert, nil
			}
		}
	} else {
		// Try IssuerAndSerialNumber (SEQUENCE)
		var issuerAndSerial IssuerAndSerialNumber
		if _, err := asn1.Unmarshal(signerInfo.SID.FullBytes, &issuerAndSerial); err == nil {
			// Parse serial number
			var serialNumber *big.Int
			if _, err := asn1.Unmarshal(issuerAndSerial.SerialNumber.FullBytes, &serialNumber); err != nil {
				return nil, fmt.Errorf("parse serial number: %w", err)
			}

			// Find certificate with matching serial number
			for _, cert := range certs {
				if cert.SerialNumber.Cmp(serialNumber) == 0 {
					return cert, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("signer certificate not found")
}

// bytesEqual compares two byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// determineSignatureType extracts the signature type from signed attributes
func determineSignatureType(signerInfo SignerInfo) SignatureType {
	if len(signerInfo.SignedAttrs.Bytes) == 0 {
		return SignatureTypeUnknown
	}

	// SignedAttributes uses [0] IMPLICIT, so the SET tag is replaced
	// We need to parse individual Attribute SEQUENCEs manually
	data := signerInfo.SignedAttrs.Bytes

	for len(data) > 0 {
		var attr Attribute
		rest, err := asn1.Unmarshal(data, &attr)
		if err != nil {
			break
		}
		data = rest

		// Look for commitment-type-indication attribute
		if attr.Type.Equal(oidCommitmentTypeIndication) {
			// Parse the attribute values (SET OF)
			var values []asn1.RawValue
			if _, err := asn1.Unmarshal(attr.Values.Bytes, &values); err != nil {
				continue
			}

			// Parse each value as OID
			for _, val := range values {
				var commitmentOID asn1.ObjectIdentifier
				if _, err := asn1.Unmarshal(val.FullBytes, &commitmentOID); err != nil {
					continue
				}

				if commitmentOID.Equal(oidAuthorSignature) {
					return SignatureTypeAuthor
				}
				if commitmentOID.Equal(oidRepositorySignature) {
					return SignatureTypeRepository
				}
			}
		}
	}

	return SignatureTypeUnknown
}

// oidToHashAlgorithm converts an OID to a hash algorithm name
func oidToHashAlgorithm(oid asn1.ObjectIdentifier) HashAlgorithmName {
	switch {
	case oid.Equal(oidSHA256):
		return HashAlgorithmSHA256
	case oid.Equal(oidSHA384):
		return HashAlgorithmSHA384
	case oid.Equal(oidSHA512):
		return HashAlgorithmSHA512
	default:
		return ""
	}
}

// extractTimestamps extracts RFC 3161 timestamps from unsigned attributes
func extractTimestamps(signerInfo SignerInfo) ([]Timestamp, error) {
	var timestamps []Timestamp

	if len(signerInfo.UnsignedAttrs.Bytes) == 0 {
		return timestamps, nil
	}

	// UnsignedAttributes uses [1] IMPLICIT, parse manually like SignedAttributes
	data := signerInfo.UnsignedAttrs.Bytes

	for len(data) > 0 {
		var attr Attribute
		rest, err := asn1.Unmarshal(data, &attr)
		if err != nil {
			break
		}
		data = rest

		// Look for timestamp token attributes
		if attr.Type.Equal(oidTimestampToken) {
			// The attribute value is a SET containing a ContentInfo (timestamp token)
			// Parse it directly as ContentInfo
			type ContentInfo struct {
				ContentType asn1.ObjectIdentifier
				Content     asn1.RawValue `asn1:"explicit,tag:0"`
			}

			var ci ContentInfo
			if _, err := asn1.Unmarshal(attr.Values.Bytes, &ci); err != nil {
				continue
			}

			// Verify it's a signedData
			if !ci.ContentType.Equal(oidSignedData) {
				continue
			}

			// Reconstruct full ContentInfo and parse as timestamp
			reconstructed, err := asn1.Marshal(ci)
			if err != nil {
				continue
			}

			ts, err := parseTimestampToken(reconstructed)
			if err != nil {
				continue // Skip invalid timestamps
			}
			timestamps = append(timestamps, ts)
		}
	}

	return timestamps, nil
}

// parseTimestampToken parses an RFC 3161 timestamp token
func parseTimestampToken(data []byte) (Timestamp, error) {
	var ts Timestamp

	// Timestamp is a SignedData structure
	sig, err := ReadSignature(data)
	if err != nil {
		return ts, err
	}

	// Extract TSTInfo from ContentInfo
	if sig.SignedData == nil {
		return ts, fmt.Errorf("no signed data in timestamp")
	}

	// Parse TSTInfo structure (RFC 3161)
	var tstInfo struct {
		Version        int
		Policy         asn1.ObjectIdentifier
		MessageImprint struct {
			HashAlgorithm AlgorithmIdentifier
			HashedMessage []byte
		}
		SerialNumber *big.Int
		GenTime      time.Time // GeneralizedTime
	}

	if len(sig.SignedData.ContentInfo.Content.Bytes) > 0 {
		// The Content is [0] EXPLICIT OCTET STRING containing TSTInfo
		// First unwrap the OCTET STRING
		var tstInfoBytes []byte
		if _, err := asn1.Unmarshal(sig.SignedData.ContentInfo.Content.Bytes, &tstInfoBytes); err != nil {
			return ts, fmt.Errorf("unmarshal TSTInfo OCTET STRING: %w", err)
		}

		// Now parse the TSTInfo SEQUENCE
		if _, err := asn1.Unmarshal(tstInfoBytes, &tstInfo); err != nil {
			return ts, fmt.Errorf("parse TSTInfo: %w", err)
		}

		// Extract timestamp
		ts.Time = tstInfo.GenTime
	}

	// Get timestamp signer certificate
	if sig.SignerCertificate != nil {
		ts.SignerCertificate = sig.SignerCertificate
	}

	// Get hash algorithm
	ts.HashAlgorithm = oidToHashAlgorithm(tstInfo.MessageImprint.HashAlgorithm.Algorithm)

	return ts, nil
}
