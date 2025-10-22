package signatures

import (
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"time"
)

// BuildSignedAttributes creates the authenticated attributes for a CMS signature (RFC 5652).
//
// The returned attributes include:
//   - content-type: identifies the signed content type (REQUIRED)
//   - signing-time: when the signature was created
//   - message-digest: hash of the signed content (REQUIRED)
//   - commitment-type-indication: NuGet signature type (Author/Repository)
//   - signing-certificate-v2: binds the signing certificate to the signature
//
// This implementation matches NuGet.Client's SigningUtility.CreateSignedAttributes behavior.
func BuildSignedAttributes(
	contentHash []byte,
	sigType SignatureType,
	cert *x509.Certificate,
	hashAlg HashAlgorithmName,
) ([]Attribute, error) {
	var attributes []Attribute

	// 1. content-type attribute (REQUIRED per RFC 5652)
	contentTypeAttr, err := createContentTypeAttribute()
	if err != nil {
		return nil, fmt.Errorf("create content-type: %w", err)
	}
	attributes = append(attributes, contentTypeAttr)

	// 2. signing-time attribute (Pkcs9SigningTime)
	signingTimeAttr, err := createSigningTimeAttribute(time.Now())
	if err != nil {
		return nil, fmt.Errorf("create signing-time: %w", err)
	}
	attributes = append(attributes, signingTimeAttr)

	// 3. message-digest attribute (REQUIRED per RFC 5652)
	messageDigestAttr, err := createMessageDigestAttribute(contentHash)
	if err != nil {
		return nil, fmt.Errorf("create message-digest: %w", err)
	}
	attributes = append(attributes, messageDigestAttr)

	// 4. commitment-type-indication (NuGet signature type)
	if sigType != SignatureTypeUnknown {
		commitmentAttr, err := createCommitmentTypeIndicationAttribute(sigType)
		if err != nil {
			return nil, fmt.Errorf("create commitment-type: %w", err)
		}
		attributes = append(attributes, commitmentAttr)
	}

	// 5. signing-certificate-v2 (ESS - binds certificate to signature)
	signingCertAttr, err := createSigningCertificateV2Attribute(cert, hashAlg)
	if err != nil {
		return nil, fmt.Errorf("create signing-certificate-v2: %w", err)
	}
	attributes = append(attributes, signingCertAttr)

	return attributes, nil
}

// createContentTypeAttribute creates the content-type authenticated attribute (RFC 5652 Section 11.1).
// This attribute identifies the type of content being signed. For NuGet signatures,
// this is always oidData (arbitrary octet string).
// Returns an Attribute with type oidContentType containing oidData.
func createContentTypeAttribute() (Attribute, error) {
	// ContentType ::= OBJECT IDENTIFIER (data)
	value, err := asn1.Marshal(oidData)
	if err != nil {
		return Attribute{}, err
	}

	values, err := asn1.Marshal([]asn1.RawValue{{FullBytes: value}})
	if err != nil {
		return Attribute{}, err
	}

	return Attribute{
		Type:   oidContentType,
		Values: asn1.RawValue{FullBytes: values},
	}, nil
}

// createSigningTimeAttribute creates the signing-time authenticated attribute (RFC 5652 Section 11.3).
// This attribute indicates when the signature was created. The time is encoded as
// UTCTime (for dates before 2050) or GeneralizedTime (for dates from 2050 onwards).
// Returns an Attribute with type oidSigningTime containing the encoded time.
func createSigningTimeAttribute(t time.Time) (Attribute, error) {
	// SigningTime ::= Time (UTCTime or GeneralizedTime)
	value, err := asn1.Marshal(t)
	if err != nil {
		return Attribute{}, err
	}

	values, err := asn1.Marshal([]asn1.RawValue{{FullBytes: value}})
	if err != nil {
		return Attribute{}, err
	}

	return Attribute{
		Type:   oidSigningTime,
		Values: asn1.RawValue{FullBytes: values},
	}, nil
}

// createMessageDigestAttribute creates the message-digest authenticated attribute (RFC 5652 Section 11.2).
// This attribute contains the hash of the content being signed. It is a required
// authenticated attribute per RFC 5652 and provides integrity protection.
// Returns an Attribute with type oidMessageDigest containing the content hash.
func createMessageDigestAttribute(digest []byte) (Attribute, error) {
	// MessageDigest ::= OCTET STRING
	value, err := asn1.Marshal(digest)
	if err != nil {
		return Attribute{}, err
	}

	values, err := asn1.Marshal([]asn1.RawValue{{FullBytes: value}})
	if err != nil {
		return Attribute{}, err
	}

	return Attribute{
		Type:   oidMessageDigest,
		Values: asn1.RawValue{FullBytes: values},
	}, nil
}

// createCommitmentTypeIndicationAttribute creates the commitment-type-indication attribute (RFC 5126 Section 5.11.1).
// This attribute indicates the commitment type of the signer. NuGet uses this to distinguish
// between Author signatures (created by package authors) and Repository signatures (created by repositories).
// The attribute value is the commitment type OID without a SEQUENCE wrapper, matching NuGet.Client behavior.
// Returns an Attribute with type oidCommitmentTypeIndication containing the signature type OID.
func createCommitmentTypeIndicationAttribute(sigType SignatureType) (Attribute, error) {
	var commitmentOID asn1.ObjectIdentifier
	switch sigType {
	case SignatureTypeAuthor:
		commitmentOID = oidAuthorSignature
	case SignatureTypeRepository:
		commitmentOID = oidRepositorySignature
	default:
		return Attribute{}, fmt.Errorf("unknown signature type: %s", sigType)
	}

	// Per NuGet.Client: CommitmentTypeIndication is just an OID (not SEQUENCE wrapper)
	// Reference: CertificateUtility.cs GetCommitmentTypeIndication
	value, err := asn1.Marshal(commitmentOID)
	if err != nil {
		return Attribute{}, err
	}

	// Attribute values are SET OF, so marshal with `set` parameter
	values, err := asn1.MarshalWithParams([]asn1.RawValue{{FullBytes: value}}, "set")
	if err != nil {
		return Attribute{}, err
	}

	// Parse to get both FullBytes and Bytes set correctly
	var valuesRaw asn1.RawValue
	if _, err := asn1.Unmarshal(values, &valuesRaw); err != nil {
		return Attribute{}, err
	}

	return Attribute{
		Type:   oidCommitmentTypeIndication,
		Values: valuesRaw, // Use parsed RawValue with both Bytes and FullBytes
	}, nil
}

// createSigningCertificateV2Attribute creates the signing-certificate-v2 attribute (RFC 5035).
// This ESS (Enhanced Security Services) attribute binds the signing certificate to the signature
// by including a hash of the certificate. This prevents certificate substitution attacks.
// The attribute includes the certificate hash using the specified hash algorithm, along with
// the certificate's issuer distinguished name and serial number for identification.
// Returns an Attribute with type oidSigningCertificateV2 containing the SigningCertificateV2 structure.
func createSigningCertificateV2Attribute(cert *x509.Certificate, hashAlg HashAlgorithmName) (Attribute, error) {
	// Hash the certificate
	h := getCryptoHash(hashAlg)
	hasher := h.New()
	hasher.Write(cert.Raw)
	certHash := hasher.Sum(nil)

	// Build IssuerSerial
	issuerSerial := IssuerSerial{
		Issuer:       []asn1.RawValue{{FullBytes: cert.RawIssuer}},
		SerialNumber: cert.SerialNumber,
	}

	// Build ESSCertIDv2
	essCertID := ESSCertIDv2{
		HashAlgorithm: AlgorithmIdentifier{
			Algorithm: getDigestAlgorithmOID(hashAlg),
		},
		CertHash:     certHash,
		IssuerSerial: issuerSerial,
	}

	// Build SigningCertificateV2
	signingCert := SigningCertificateV2{
		Certs: []ESSCertIDv2{essCertID},
	}

	value, err := asn1.Marshal(signingCert)
	if err != nil {
		return Attribute{}, err
	}

	values, err := asn1.Marshal([]asn1.RawValue{{FullBytes: value}})
	if err != nil {
		return Attribute{}, err
	}

	return Attribute{
		Type:   oidSigningCertificateV2,
		Values: asn1.RawValue{FullBytes: values},
	}, nil
}

// EncodeAttributesForSigning encodes authenticated attributes for signing using DER with SET tag.
// Per RFC 5652 Section 5.3, the signature is computed over the DER encoding of the signedAttrs
// field with the SET OF tag. This function marshals the attributes as a SET (tag 17, constructed)
// which is then hashed and signed by the signer.
// Returns the DER-encoded SET OF Attribute ready for hashing and signing.
func EncodeAttributesForSigning(attributes []Attribute) ([]byte, error) {
	// Encode as SET (tag 17, constructed)
	encoded, err := asn1.MarshalWithParams(attributes, "set")
	if err != nil {
		return nil, err
	}
	return encoded, nil
}
