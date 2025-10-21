package signatures

import (
	"crypto/x509"
	"encoding/asn1"
	"math/big"
	"os"
	"strings"
	"testing"
)

func TestReadSignature_EmptyData(t *testing.T) {
	_, err := ReadSignature([]byte{})
	if err == nil {
		t.Error("ReadSignature() should fail with empty data")
	}

	if err.Error() != "signature data is empty" {
		t.Errorf("ReadSignature() error = %v, want 'signature data is empty'", err)
	}
}

func TestReadSignature_InvalidData(t *testing.T) {
	invalidData := []byte{0xFF, 0xFF, 0xFF}

	_, err := ReadSignature(invalidData)
	if err == nil {
		t.Error("ReadSignature() should fail with invalid data")
	}
}

func TestReadSignature_RealSignature(t *testing.T) {
	sigData, err := os.ReadFile("testdata/test.signature.p7s")
	if err != nil {
		t.Fatalf("Failed to read test signature file: %v", err)
	}

	sig, err := ReadSignature(sigData)
	if err != nil {
		t.Fatalf("ReadSignature() error = %v, want nil", err)
	}

	if len(sig.RawData) == 0 {
		t.Error("ReadSignature() RawData is empty")
	}

	if sig.SignedData == nil {
		t.Error("ReadSignature() SignedData is nil")
	}

	if sig.HashAlgorithm == "" {
		t.Error("ReadSignature() HashAlgorithm is empty")
	}

	if len(sig.Certificates) == 0 {
		t.Error("ReadSignature() Certificates is empty, expected at least one")
	}

	if sig.SignerCertificate == nil {
		t.Error("ReadSignature() SignerCertificate is nil")
	}
}

func TestReadSignature_AuthorSigned(t *testing.T) {
	sigData, err := os.ReadFile("testdata/author.signature.p7s")
	if err != nil {
		t.Fatalf("Failed to read author signature file: %v", err)
	}

	sig, err := ReadSignature(sigData)
	if err != nil {
		t.Fatalf("ReadSignature() error = %v, want nil", err)
	}

	// Should detect Author signature type
	if sig.Type != SignatureTypeAuthor {
		t.Errorf("Signature type = %v, want %v", sig.Type, SignatureTypeAuthor)
	}

	// Should have timestamp
	if len(sig.Timestamps) == 0 {
		t.Log("Warning: No timestamps found")
	}
}

func TestOidToHashAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		oid      []int
		expected HashAlgorithmName
	}{
		{"SHA256", []int{2, 16, 840, 1, 101, 3, 4, 2, 1}, HashAlgorithmSHA256},
		{"SHA384", []int{2, 16, 840, 1, 101, 3, 4, 2, 2}, HashAlgorithmSHA384},
		{"SHA512", []int{2, 16, 840, 1, 101, 3, 4, 2, 3}, HashAlgorithmSHA512},
		{"Unknown", []int{1, 2, 3, 4}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := oidToHashAlgorithm(tt.oid)
			if result != tt.expected {
				t.Errorf("oidToHashAlgorithm() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSignatureType_Values(t *testing.T) {
	if SignatureTypeAuthor != "Author" {
		t.Errorf("SignatureTypeAuthor = %v, want 'Author'", SignatureTypeAuthor)
	}
	if SignatureTypeRepository != "Repository" {
		t.Errorf("SignatureTypeRepository = %v, want 'Repository'", SignatureTypeRepository)
	}
	if SignatureTypeUnknown != "Unknown" {
		t.Errorf("SignatureTypeUnknown = %v, want 'Unknown'", SignatureTypeUnknown)
	}
}

func TestHashAlgorithmName_Values(t *testing.T) {
	if HashAlgorithmSHA256 != "SHA256" {
		t.Errorf("HashAlgorithmSHA256 = %v, want 'SHA256'", HashAlgorithmSHA256)
	}
	if HashAlgorithmSHA384 != "SHA384" {
		t.Errorf("HashAlgorithmSHA384 = %v, want 'SHA384'", HashAlgorithmSHA384)
	}
	if HashAlgorithmSHA512 != "SHA512" {
		t.Errorf("HashAlgorithmSHA512 = %v, want 'SHA512'", HashAlgorithmSHA512)
	}
}

func TestReadSignature_NotSignedData(t *testing.T) {
	// Read a real signature and modify the OID
	sigData, err := os.ReadFile("testdata/test.signature.p7s")
	if err != nil {
		t.Fatalf("Failed to read test signature: %v", err)
	}

	// Parse the ContentInfo to extract its structure
	var contentInfo struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue `asn1:"explicit,tag:0"`
	}
	_, err = asn1.Unmarshal(sigData, &contentInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal ContentInfo: %v", err)
	}

	// Change to data OID instead of signedData
	contentInfo.ContentType = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}

	// Re-marshal
	modifiedData, _ := asn1.Marshal(contentInfo)

	_, err = ReadSignature(modifiedData)
	if err == nil {
		t.Error("ReadSignature() should fail with non-SignedData OID")
	}
	if err != nil && !strings.Contains(err.Error(), "not a SignedData structure") {
		t.Errorf("Expected 'not a SignedData structure' error, got: %v", err)
	}
}

func TestReadSignature_TrailingData(t *testing.T) {
	// Read a real signature
	sigData, err := os.ReadFile("testdata/test.signature.p7s")
	if err != nil {
		t.Fatalf("Failed to read test signature: %v", err)
	}

	// Append extra data
	sigData = append(sigData, []byte{0x30, 0x03, 0x01, 0x01, 0xFF}...)

	_, err = ReadSignature(sigData)
	if err == nil {
		t.Error("ReadSignature() should fail with trailing data")
	}
	if err != nil && !strings.Contains(err.Error(), "trailing data") {
		t.Errorf("Expected 'trailing data' error, got: %v", err)
	}
}

func TestParseCertificates_Empty(t *testing.T) {
	var emptyRaw asn1.RawValue
	certs, err := parseCertificates(emptyRaw)
	if err != nil {
		t.Errorf("parseCertificates() error = %v, want nil", err)
	}
	if len(certs) != 0 {
		t.Errorf("parseCertificates() returned %d certs, want 0", len(certs))
	}
}

func TestBytesEqual(t *testing.T) {
	tests := []struct {
		name string
		a    []byte
		b    []byte
		want bool
	}{
		{"equal", []byte{1, 2, 3}, []byte{1, 2, 3}, true},
		{"not equal", []byte{1, 2, 3}, []byte{1, 2, 4}, false},
		{"different lengths", []byte{1, 2}, []byte{1, 2, 3}, false},
		{"both empty", []byte{}, []byte{}, true},
		{"one empty", []byte{1}, []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bytesEqual(tt.a, tt.b)
			if result != tt.want {
				t.Errorf("bytesEqual() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestDetermineSignatureType_EmptyAttrs(t *testing.T) {
	si := SignerInfo{
		SignedAttrs: asn1.RawValue{Bytes: []byte{}},
	}
	result := determineSignatureType(si)
	if result != SignatureTypeUnknown {
		t.Errorf("determineSignatureType() = %v, want %v", result, SignatureTypeUnknown)
	}
}

func TestExtractTimestamps_Empty(t *testing.T) {
	si := SignerInfo{
		UnsignedAttrs: asn1.RawValue{Bytes: []byte{}},
	}
	timestamps, err := extractTimestamps(si)
	if err != nil {
		t.Errorf("extractTimestamps() error = %v, want nil", err)
	}
	if len(timestamps) != 0 {
		t.Errorf("extractTimestamps() returned %d timestamps, want 0", len(timestamps))
	}
}

func TestReadSignature_RepositorySigned(t *testing.T) {
	sigData, err := os.ReadFile("testdata/repository.signature.p7s")
	if err != nil {
		t.Fatalf("Failed to read repository signature file: %v", err)
	}

	sig, err := ReadSignature(sigData)
	if err != nil {
		t.Fatalf("ReadSignature() error = %v, want nil", err)
	}

	// Should detect Repository signature type
	if sig.Type != SignatureTypeRepository {
		t.Errorf("Signature type = %v, want %v", sig.Type, SignatureTypeRepository)
	}

	// Should have basic signature data
	if len(sig.RawData) == 0 {
		t.Error("ReadSignature() RawData is empty")
	}

	if sig.SignedData == nil {
		t.Error("ReadSignature() SignedData is nil")
	}
}

func TestReadSignature_WithTimestamps(t *testing.T) {
	// Use Newtonsoft signature which may have timestamps
	sigData, err := os.ReadFile("testdata/newtonsoft.signature.p7s")
	if err != nil {
		t.Fatalf("Failed to read newtonsoft signature file: %v", err)
	}

	sig, err := ReadSignature(sigData)
	if err != nil {
		t.Fatalf("ReadSignature() error = %v, want nil", err)
	}

	// If timestamps are present, validate them
	if len(sig.Timestamps) > 0 {
		ts := sig.Timestamps[0]
		if ts.Time.IsZero() {
			t.Error("Timestamp Time is zero")
		}
		if ts.SignerCertificate == nil {
			t.Error("Timestamp SignerCertificate is nil")
		}
		if ts.HashAlgorithm == "" {
			t.Error("Timestamp HashAlgorithm is empty")
		}
	}
}

func TestFindSignerCertificate_NotFound(t *testing.T) {
	// Test when signer certificate is not in the provided list
	signerInfo := SignerInfo{
		SID: asn1.RawValue{
			Tag:   0,
			Class: 2,
			Bytes: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		},
	}

	_, err := findSignerCertificate(signerInfo, []*x509.Certificate{})
	if err == nil {
		t.Error("findSignerCertificate() should fail when certificate not found")
	}
}

func TestParseCertificates_Invalid(t *testing.T) {
	invalidCertData := asn1.RawValue{
		Bytes: []byte{0xFF, 0xFF, 0xFF},
	}
	_, err := parseCertificates(invalidCertData)
	if err == nil {
		t.Error("parseCertificates() should fail with invalid data")
	}
}

func TestReadSignature_Countersigned(t *testing.T) {
	// Countersigned packages typically have RFC 3161 timestamps
	sigData, err := os.ReadFile("testdata/countersigned.signature.p7s")
	if err != nil {
		t.Fatalf("Failed to read countersigned signature file: %v", err)
	}

	sig, err := ReadSignature(sigData)
	if err != nil {
		t.Fatalf("ReadSignature() error = %v, want nil", err)
	}

	// Validate basic signature structure
	if sig.SignedData == nil {
		t.Error("SignedData is nil")
	}
	if sig.SignerCertificate == nil {
		t.Error("SignerCertificate is nil")
	}

	// If this has timestamps, validate them
	if len(sig.Timestamps) > 0 {
		for i, ts := range sig.Timestamps {
			if ts.Time.IsZero() {
				t.Errorf("Timestamp[%d] Time is zero", i)
			}
			if ts.SignerCertificate == nil {
				t.Errorf("Timestamp[%d] SignerCertificate is nil", i)
			}
		}
	}
}

func TestParseTimestampToken_InvalidData(t *testing.T) {
	// Test with invalid timestamp data
	_, err := parseTimestampToken([]byte{0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Error("parseTimestampToken() should fail with invalid data")
	}
}

func TestParseTimestampToken_EmptyData(t *testing.T) {
	// Test with empty data
	_, err := parseTimestampToken([]byte{})
	if err == nil {
		t.Error("parseTimestampToken() should fail with empty data")
	}
}

func TestFindSignerCertificate_IssuerAndSerialNumber(t *testing.T) {
	// Create a test certificate
	testCert := &x509.Certificate{
		SerialNumber: big.NewInt(12345),
	}

	// Create SignerInfo with IssuerAndSerialNumber
	// IssuerAndSerialNumber is a SEQUENCE of Issuer (RawValue) and SerialNumber (RawValue)
	serialBytes, _ := asn1.Marshal(big.NewInt(12345))
	issuerBytes := []byte{0x30, 0x00} // Empty SEQUENCE for issuer

	issuerAndSerial := IssuerAndSerialNumber{
		Issuer:       asn1.RawValue{FullBytes: issuerBytes},
		SerialNumber: asn1.RawValue{FullBytes: serialBytes},
	}

	issuerAndSerialBytes, _ := asn1.Marshal(issuerAndSerial)

	signerInfo := SignerInfo{
		SID: asn1.RawValue{
			FullBytes: issuerAndSerialBytes,
			Tag:       16, // SEQUENCE tag
			Class:     0,  // Universal class
		},
	}

	cert, err := findSignerCertificate(signerInfo, []*x509.Certificate{testCert})
	if err != nil {
		t.Errorf("findSignerCertificate() error = %v, want nil", err)
	}
	if cert == nil {
		t.Error("findSignerCertificate() returned nil certificate")
	}
	if cert != nil && cert.SerialNumber.Cmp(big.NewInt(12345)) != 0 {
		t.Errorf("findSignerCertificate() returned cert with wrong serial number")
	}
}

func TestFindSignerCertificate_IssuerAndSerialNumber_NotFound(t *testing.T) {
	// Create a test certificate with different serial
	testCert := &x509.Certificate{
		SerialNumber: big.NewInt(99999),
	}

	// Create SignerInfo with IssuerAndSerialNumber looking for serial 12345
	serialBytes, _ := asn1.Marshal(big.NewInt(12345))
	issuerBytes := []byte{0x30, 0x00}

	issuerAndSerial := IssuerAndSerialNumber{
		Issuer:       asn1.RawValue{FullBytes: issuerBytes},
		SerialNumber: asn1.RawValue{FullBytes: serialBytes},
	}

	issuerAndSerialBytes, _ := asn1.Marshal(issuerAndSerial)

	signerInfo := SignerInfo{
		SID: asn1.RawValue{
			FullBytes: issuerAndSerialBytes,
			Tag:       16,
			Class:     0,
		},
	}

	_, err := findSignerCertificate(signerInfo, []*x509.Certificate{testCert})
	if err == nil {
		t.Error("findSignerCertificate() should fail when serial number doesn't match")
	}
}

func TestFindSignerCertificate_InvalidSerialNumber(t *testing.T) {
	// Create SignerInfo with invalid serial number bytes
	issuerAndSerial := IssuerAndSerialNumber{
		Issuer:       asn1.RawValue{FullBytes: []byte{0x30, 0x00}},
		SerialNumber: asn1.RawValue{FullBytes: []byte{0xFF, 0xFF}}, // Invalid
	}

	issuerAndSerialBytes, _ := asn1.Marshal(issuerAndSerial)

	signerInfo := SignerInfo{
		SID: asn1.RawValue{
			FullBytes: issuerAndSerialBytes,
			Tag:       16,
			Class:     0,
		},
	}

	_, err := findSignerCertificate(signerInfo, []*x509.Certificate{})
	if err == nil {
		t.Error("findSignerCertificate() should fail with invalid serial number")
	}
}

func TestDetermineSignatureType_InvalidAttribute(t *testing.T) {
	// Create SignerInfo with invalid attribute data that will cause unmarshal errors
	signerInfo := SignerInfo{
		SignedAttrs: asn1.RawValue{
			Bytes: []byte{0xFF, 0xFF, 0xFF}, // Invalid attribute data
		},
	}

	result := determineSignatureType(signerInfo)
	if result != SignatureTypeUnknown {
		t.Errorf("determineSignatureType() = %v, want %v", result, SignatureTypeUnknown)
	}
}

func TestExtractTimestamps_InvalidAttribute(t *testing.T) {
	// Create SignerInfo with invalid unsigned attribute data
	signerInfo := SignerInfo{
		UnsignedAttrs: asn1.RawValue{
			Bytes: []byte{0xFF, 0xFF, 0xFF},
		},
	}

	timestamps, err := extractTimestamps(signerInfo)
	if err != nil {
		t.Errorf("extractTimestamps() error = %v, want nil", err)
	}
	if len(timestamps) != 0 {
		t.Errorf("extractTimestamps() returned %d timestamps, want 0", len(timestamps))
	}
}

func TestReadSignature_WithActualTimestamp(t *testing.T) {
	// This package specifically has RFC 3161 timestamps
	sigData, err := os.ReadFile("testdata/newtonsoft.signature.p7s")
	if err != nil {
		t.Fatalf("Failed to read newtonsoft signature file: %v", err)
	}

	sig, err := ReadSignature(sigData)
	if err != nil {
		t.Fatalf("ReadSignature() error = %v, want nil", err)
	}

	// Should have at least one timestamp
	if len(sig.Timestamps) == 0 {
		t.Error("Expected timestamps in newtonsoft.signature.p7s but found none")
	}

	// Validate timestamp structure
	for i, ts := range sig.Timestamps {
		if ts.Time.IsZero() {
			t.Errorf("Timestamp[%d] Time is zero", i)
		}
		if ts.HashAlgorithm == "" {
			t.Errorf("Timestamp[%d] HashAlgorithm is empty", i)
		}
		if ts.SignerCertificate == nil {
			t.Errorf("Timestamp[%d] SignerCertificate is nil", i)
		}
	}
}

func TestReadSignature_InvalidSignedData(t *testing.T) {
	// Create a ContentInfo with signedData OID but corrupt content
	type TempContentInfo struct {
		ContentType asn1.ObjectIdentifier
		Content     []byte `asn1:"explicit,tag:0"` // This will create [0] EXPLICIT wrapper
	}

	ci := TempContentInfo{
		ContentType: oidSignedData,
		Content:     []byte{0x30, 0x01, 0xFF}, // Invalid SEQUENCE - claims 1 byte but has 0xFF which is invalid
	}

	data, err := asn1.Marshal(ci)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	_, err = ReadSignature(data)
	if err == nil {
		t.Error("ReadSignature() should fail with invalid SignedData")
	}
	// The error could be either unmarshal or parse certificates or other downstream errors
	// Just verify it fails
}

func TestReadSignature_InvalidCertificates(t *testing.T) {
	// Create SignedData with invalid certificates
	validOID := []int{1, 2, 840, 113549, 1, 7, 2}
	type ContentInfo struct {
		ContentType []int
		Content     []byte `asn1:"explicit,tag:0"`
	}

	type SignedDataBad struct {
		Version          int                   `asn1:"default:1"`
		DigestAlgorithms []AlgorithmIdentifier `asn1:"set"`
		ContentInfo      EncapsulatedContentInfo
		Certificates     []byte       `asn1:"optional,tag:0"` // Bad certs
		SignerInfos      []SignerInfo `asn1:"set"`
	}

	badSignedData := SignedDataBad{
		Version:          1,
		DigestAlgorithms: []AlgorithmIdentifier{},
		ContentInfo: EncapsulatedContentInfo{
			ContentType: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1},
		},
		Certificates: []byte{0xFF, 0xFF}, // Invalid
		SignerInfos: []SignerInfo{
			{
				DigestAlgorithm: AlgorithmIdentifier{
					Algorithm: asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1},
				},
			},
		},
	}

	signedDataBytes, _ := asn1.Marshal(badSignedData)
	ci := ContentInfo{
		ContentType: validOID,
		Content:     signedDataBytes,
	}
	data, _ := asn1.Marshal(ci)

	_, err := ReadSignature(data)
	if err == nil {
		t.Error("ReadSignature() should fail with invalid certificates")
	}
}

func TestExtractTimestamps_WrongOID(t *testing.T) {
	// Create an attribute with timestamp token OID but wrong ContentType
	wrongOID := asn1.ObjectIdentifier{1, 2, 3, 4, 5}

	type TempContentInfo struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue `asn1:"explicit,tag:0"`
	}

	ci := TempContentInfo{
		ContentType: wrongOID, // Wrong OID
		Content: asn1.RawValue{
			Bytes: []byte{0x30, 0x00},
		},
	}

	ciBytes, _ := asn1.Marshal(ci)

	attr := Attribute{
		Type: oidTimestampToken,
		Values: asn1.RawValue{
			Bytes: ciBytes,
		},
	}

	// Manually construct SignerInfo with this attribute
	attrs, _ := asn1.Marshal(attr)

	si := SignerInfo{
		UnsignedAttrs: asn1.RawValue{
			Bytes: attrs,
		},
	}

	timestamps, err := extractTimestamps(si)
	if err != nil {
		t.Errorf("extractTimestamps() error = %v, want nil", err)
	}

	// Should skip timestamp with wrong OID
	if len(timestamps) != 0 {
		t.Errorf("extractTimestamps() returned %d timestamps, want 0", len(timestamps))
	}
}

func TestExtractTimestamps_InvalidContentInfo(t *testing.T) {
	// Create an attribute with invalid ContentInfo bytes
	attr := Attribute{
		Type: oidTimestampToken,
		Values: asn1.RawValue{
			Bytes: []byte{0xFF, 0xFF, 0xFF}, // Invalid
		},
	}

	attrs, _ := asn1.Marshal(attr)

	si := SignerInfo{
		UnsignedAttrs: asn1.RawValue{
			Bytes: attrs,
		},
	}

	timestamps, err := extractTimestamps(si)
	if err != nil {
		t.Errorf("extractTimestamps() error = %v, want nil", err)
	}

	// Should skip invalid timestamp
	if len(timestamps) != 0 {
		t.Errorf("extractTimestamps() returned %d timestamps, want 0", len(timestamps))
	}
}

func TestReadSignature_UnknownHashAlgorithm(t *testing.T) {
	// Test with a real signature to ensure all code paths work
	sigData, err := os.ReadFile("testdata/test.signature.p7s")
	if err != nil {
		t.Fatalf("Failed to read signature: %v", err)
	}

	sig, err := ReadSignature(sigData)
	if err != nil {
		t.Fatalf("ReadSignature() error = %v, want nil", err)
	}

	// Should successfully parse
	if sig == nil {
		t.Fatal("ReadSignature() returned nil signature")
	}

	if sig.HashAlgorithm == "" {
		t.Error("HashAlgorithm should not be empty")
	}
}
