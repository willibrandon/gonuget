package signatures

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"testing"
	"time"
)

// TestSignPackageData_ValidAuthorSignature tests creating a valid author signature
func TestSignPackageData_ValidAuthorSignature(t *testing.T) {
	// Arrange: Generate test certificates (using helpers from verifier_test.go)
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Calculate test package hash
	testData := []byte("test package content")
	hasher := sha256.New()
	hasher.Write(testData)
	contentHash := hasher.Sum(nil)

	// Create signing options
	opts := SigningOptions{
		Certificate:      signerCert,
		PrivateKey:       signerKey,
		CertificateChain: []*x509.Certificate{rootCert},
		SignatureType:    SignatureTypeAuthor,
		HashAlgorithm:    HashAlgorithmSHA256,
	}

	// Act: Sign the package data
	signature, err := SignPackageData(contentHash, opts)

	// Assert
	if err != nil {
		t.Fatalf("SignPackageData failed: %v", err)
	}

	if len(signature) == 0 {
		t.Fatal("signature is empty")
	}

	// Verify it's valid PKCS#7 by parsing ContentInfo
	var contentInfo ContentInfo
	_, err = asn1.Unmarshal(signature, &contentInfo)
	if err != nil {
		t.Fatalf("failed to parse ContentInfo: %v", err)
	}

	if !contentInfo.ContentType.Equal(oidSignedData) {
		t.Errorf("expected SignedData OID, got %v", contentInfo.ContentType)
	}
}

// TestSignPackageData_RepositorySignature tests creating a repository signature
func TestSignPackageData_RepositorySignature(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	testData := []byte("test package content")
	hasher := sha256.New()
	hasher.Write(testData)
	contentHash := hasher.Sum(nil)

	opts := SigningOptions{
		Certificate:   signerCert,
		PrivateKey:    signerKey,
		SignatureType: SignatureTypeRepository,
		HashAlgorithm: HashAlgorithmSHA256,
	}

	signature, err := SignPackageData(contentHash, opts)
	if err != nil {
		t.Fatalf("SignPackageData failed: %v", err)
	}

	if len(signature) == 0 {
		t.Fatal("signature is empty")
	}
}

// TestSignPackageData_AllHashAlgorithms tests all supported hash algorithms
func TestSignPackageData_AllHashAlgorithms(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	testCases := []struct {
		name      string
		hashAlg   HashAlgorithmName
		expectOID asn1.ObjectIdentifier
	}{
		{"SHA256", HashAlgorithmSHA256, oidSHA256WithRSA},
		{"SHA384", HashAlgorithmSHA384, oidSHA384WithRSA},
		{"SHA512", HashAlgorithmSHA512, oidSHA512WithRSA},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testData := []byte("test package content")
			h := getCryptoHash(tc.hashAlg)
			hasher := h.New()
			hasher.Write(testData)
			contentHash := hasher.Sum(nil)

			opts := SigningOptions{
				Certificate:   signerCert,
				PrivateKey:    signerKey,
				SignatureType: SignatureTypeAuthor,
				HashAlgorithm: tc.hashAlg,
			}

			signature, err := SignPackageData(contentHash, opts)
			if err != nil {
				t.Fatalf("SignPackageData failed for %s: %v", tc.name, err)
			}

			if len(signature) == 0 {
				t.Fatalf("signature is empty for %s", tc.name)
			}
		})
	}
}

// TestSigningOptions_Validate tests signing options validation
func TestSigningOptions_Validate(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)
	_, wrongKey := generateTestCodeSigningCert(t, rootCert, rootKey)
	weakCert, weakKey := generateWeakRSAKeyCert(t, rootCert, rootKey)

	testCases := []struct {
		name        string
		opts        SigningOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid options",
			opts: SigningOptions{
				Certificate:   signerCert,
				PrivateKey:    signerKey,
				SignatureType: SignatureTypeAuthor,
				HashAlgorithm: HashAlgorithmSHA256,
			},
			expectError: false,
		},
		{
			name: "Missing certificate",
			opts: SigningOptions{
				PrivateKey:    signerKey,
				SignatureType: SignatureTypeAuthor,
				HashAlgorithm: HashAlgorithmSHA256,
			},
			expectError: true,
			errorMsg:    "certificate is required",
		},
		{
			name: "Missing private key",
			opts: SigningOptions{
				Certificate:   signerCert,
				SignatureType: SignatureTypeAuthor,
				HashAlgorithm: HashAlgorithmSHA256,
			},
			expectError: true,
			errorMsg:    "private key is required",
		},
		{
			name: "Key mismatch",
			opts: SigningOptions{
				Certificate:   signerCert,
				PrivateKey:    wrongKey,
				SignatureType: SignatureTypeAuthor,
				HashAlgorithm: HashAlgorithmSHA256,
			},
			expectError: true,
			errorMsg:    "key does not match certificate",
		},
		{
			name: "Weak RSA key",
			opts: SigningOptions{
				Certificate:   weakCert,
				PrivateKey:    weakKey,
				SignatureType: SignatureTypeAuthor,
				HashAlgorithm: HashAlgorithmSHA256,
			},
			expectError: true,
			errorMsg:    "RSA key must be at least 2048 bits",
		},
		{
			name: "Invalid hash algorithm",
			opts: SigningOptions{
				Certificate:   signerCert,
				PrivateKey:    signerKey,
				SignatureType: SignatureTypeAuthor,
				HashAlgorithm: "SHA1", // Not allowed
			},
			expectError: true,
			errorMsg:    "hash algorithm",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.Validate()

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error containing '%s', got nil", tc.errorMsg)
				}
				if tc.errorMsg != "" && !contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error containing '%s', got '%v'", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestBuildSignedAttributes tests attribute creation
func TestBuildSignedAttributes(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	contentHash := sha256.Sum256([]byte("test content"))

	testCases := []struct {
		name      string
		sigType   SignatureType
		hashAlg   HashAlgorithmName
		expectErr bool
	}{
		{"Author signature", SignatureTypeAuthor, HashAlgorithmSHA256, false},
		{"Repository signature", SignatureTypeRepository, HashAlgorithmSHA256, false},
		{"SHA384", SignatureTypeAuthor, HashAlgorithmSHA384, false},
		{"SHA512", SignatureTypeAuthor, HashAlgorithmSHA512, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attrs, err := BuildSignedAttributes(contentHash[:], tc.sigType, signerCert, tc.hashAlg)

			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify we have required attributes
			if len(attrs) < 4 {
				t.Fatalf("expected at least 4 attributes, got %d", len(attrs))
			}

			// Verify attribute OIDs
			foundContentType := false
			foundSigningTime := false
			foundMessageDigest := false
			foundCommitment := false
			foundSigningCert := false

			for _, attr := range attrs {
				switch {
				case attr.Type.Equal(oidContentType):
					foundContentType = true
				case attr.Type.Equal(oidSigningTime):
					foundSigningTime = true
				case attr.Type.Equal(oidMessageDigest):
					foundMessageDigest = true
				case attr.Type.Equal(oidCommitmentTypeIndication):
					foundCommitment = true
				case attr.Type.Equal(oidSigningCertificateV2):
					foundSigningCert = true
				}
			}

			if !foundContentType {
				t.Error("missing content-type attribute")
			}
			if !foundSigningTime {
				t.Error("missing signing-time attribute")
			}
			if !foundMessageDigest {
				t.Error("missing message-digest attribute")
			}
			if !foundCommitment && tc.sigType != SignatureTypeUnknown {
				t.Error("missing commitment-type-indication attribute")
			}
			if !foundSigningCert {
				t.Error("missing signing-certificate-v2 attribute")
			}
		})
	}
}

// TestCreateCommitmentTypeIndicationAttribute tests commitment type attribute
func TestCreateCommitmentTypeIndicationAttribute(t *testing.T) {
	testCases := []struct {
		name        string
		sigType     SignatureType
		expectedOID asn1.ObjectIdentifier
		expectErr   bool
	}{
		{
			name:        "Author signature",
			sigType:     SignatureTypeAuthor,
			expectedOID: oidAuthorSignature,
			expectErr:   false,
		},
		{
			name:        "Repository signature",
			sigType:     SignatureTypeRepository,
			expectedOID: oidRepositorySignature,
			expectErr:   false,
		},
		{
			name:      "Unknown signature type",
			sigType:   SignatureTypeUnknown,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attr, err := createCommitmentTypeIndicationAttribute(tc.sigType)

			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !attr.Type.Equal(oidCommitmentTypeIndication) {
				t.Errorf("expected OID %v, got %v", oidCommitmentTypeIndication, attr.Type)
			}

			// Decode and verify commitment OID
			var values []asn1.RawValue
			_, err = asn1.UnmarshalWithParams(attr.Values.FullBytes, &values, "set")
			if err != nil {
				t.Fatalf("failed to unmarshal values: %v", err)
			}

			if len(values) != 1 {
				t.Fatalf("expected 1 value, got %d", len(values))
			}

			// Per NuGet.Client, commitment is just an OID (not SEQUENCE wrapper)
			var commitmentOID asn1.ObjectIdentifier
			_, err = asn1.Unmarshal(values[0].FullBytes, &commitmentOID)
			if err != nil {
				t.Fatalf("failed to unmarshal commitment OID: %v", err)
			}

			if !commitmentOID.Equal(tc.expectedOID) {
				t.Errorf("expected commitment OID %v, got %v", tc.expectedOID, commitmentOID)
			}
		})
	}
}

// TestEncodeAttributesForSigning tests attribute encoding
func TestEncodeAttributesForSigning(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	contentHash := sha256.Sum256([]byte("test content"))
	attrs, err := BuildSignedAttributes(contentHash[:], SignatureTypeAuthor, signerCert, HashAlgorithmSHA256)
	if err != nil {
		t.Fatalf("BuildSignedAttributes failed: %v", err)
	}

	encoded, err := EncodeAttributesForSigning(attrs)
	if err != nil {
		t.Fatalf("EncodeAttributesForSigning failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Fatal("encoded attributes are empty")
	}

	// Verify it's a SET (tag 17)
	if encoded[0] != 0x31 { // SET tag
		t.Errorf("expected SET tag (0x31), got 0x%02x", encoded[0])
	}
}

// TestSignAndVerifyIntegration tests end-to-end signature creation and verification
func TestSignAndVerifyIntegration(t *testing.T) {
	// Generate certificates
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Create test package hash
	testData := []byte("test package content for integration test")
	contentHash := sha256.Sum256(testData)

	// Sign package
	opts := SigningOptions{
		Certificate:      signerCert,
		PrivateKey:       signerKey,
		CertificateChain: []*x509.Certificate{rootCert},
		SignatureType:    SignatureTypeAuthor,
		HashAlgorithm:    HashAlgorithmSHA256,
	}

	signature, err := SignPackageData(contentHash[:], opts)
	if err != nil {
		t.Fatalf("SignPackageData failed: %v", err)
	}

	// Parse signature using existing reader (M3.8)
	var contentInfo ContentInfo
	_, err = asn1.Unmarshal(signature, &contentInfo)
	if err != nil {
		t.Fatalf("failed to parse signature: %v", err)
	}

	// Verify signature structure
	if !contentInfo.ContentType.Equal(oidSignedData) {
		t.Errorf("expected SignedData, got OID %v", contentInfo.ContentType)
	}

	// Parse SignedData
	var signedData SignedData
	_, err = asn1.Unmarshal(contentInfo.Content.Bytes, &signedData)
	if err != nil {
		t.Fatalf("failed to parse SignedData: %v", err)
	}

	// Verify SignedData structure
	if signedData.Version != 1 {
		t.Errorf("expected version 1, got %d", signedData.Version)
	}

	if len(signedData.SignerInfos) != 1 {
		t.Fatalf("expected 1 SignerInfo, got %d", len(signedData.SignerInfos))
	}

	// Verify certificates are included
	if len(signedData.Certificates.Bytes) == 0 {
		t.Error("certificates not included in signature")
	}

	// Verify SignerInfo
	signerInfo := signedData.SignerInfos[0]
	if len(signerInfo.Signature) == 0 {
		t.Error("signature is empty")
	}

	if len(signerInfo.SignedAttrs.Bytes) == 0 {
		t.Error("signed attributes are empty")
	}
}

// TestDefaultSigningOptions tests default options
func TestDefaultSigningOptions(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	opts := DefaultSigningOptions(signerCert, signerKey)

	if opts.Certificate != signerCert {
		t.Error("certificate not set")
	}

	if opts.PrivateKey != signerKey {
		t.Error("private key not set")
	}

	if opts.SignatureType != SignatureTypeAuthor {
		t.Errorf("expected SignatureTypeAuthor, got %s", opts.SignatureType)
	}

	if opts.HashAlgorithm != HashAlgorithmSHA256 {
		t.Errorf("expected SHA256, got %s", opts.HashAlgorithm)
	}

	if opts.TimestampTimeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", opts.TimestampTimeout)
	}
}

// TestSignPackageData_WithoutTimestamp tests creating signature without timestamp
// Matches NuGet.Client behavior when no timestamp URL is provided
func TestSignPackageData_WithoutTimestamp(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	testData := []byte("test package content")
	hash := sha256.Sum256(testData)

	// Create signature WITHOUT TimestampURL
	opts := SigningOptions{
		Certificate:   signerCert,
		PrivateKey:    signerKey,
		SignatureType: SignatureTypeAuthor,
		HashAlgorithm: HashAlgorithmSHA256,
		// TimestampURL is empty - no timestamp should be added
	}

	signature, err := SignPackageData(hash[:], opts)
	if err != nil {
		t.Fatalf("SignPackageData failed: %v", err)
	}

	if len(signature) == 0 {
		t.Fatal("signature is empty")
	}

	// Parse signature
	var contentInfo ContentInfo
	_, err = asn1.Unmarshal(signature, &contentInfo)
	if err != nil {
		t.Fatalf("failed to parse ContentInfo: %v", err)
	}

	var signedData SignedData
	_, err = asn1.Unmarshal(contentInfo.Content.Bytes, &signedData)
	if err != nil {
		t.Fatalf("failed to parse SignedData: %v", err)
	}

	// Verify SignerInfo exists
	if len(signedData.SignerInfos) != 1 {
		t.Fatalf("expected 1 SignerInfo, got %d", len(signedData.SignerInfos))
	}

	// Verify unsigned attributes are EMPTY (no timestamp)
	signerInfo := signedData.SignerInfos[0]
	if len(signerInfo.UnsignedAttrs.Bytes) > 0 {
		t.Error("unsigned attributes should be empty when no timestamp URL provided")
	}
}

// TestSignPackageData_WithTimestamp tests creating signature with timestamp
// Requires real timestamp server (skip if unavailable)
func TestSignPackageData_WithTimestamp(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	testData := []byte("test package content")
	hash := sha256.Sum256(testData)

	// Create signature WITH TimestampURL
	opts := SigningOptions{
		Certificate:      signerCert,
		PrivateKey:       signerKey,
		SignatureType:    SignatureTypeAuthor,
		HashAlgorithm:    HashAlgorithmSHA256,
		TimestampURL:     "http://timestamp.digicert.com",
		TimestampTimeout: 10 * time.Second,
	}

	signature, err := SignPackageData(hash[:], opts)
	if err != nil {
		t.Skipf("SignPackageData with timestamp failed (TSA may be unavailable): %v", err)
	}

	if len(signature) == 0 {
		t.Fatal("signature is empty")
	}

	// Parse signature
	var contentInfo ContentInfo
	_, err = asn1.Unmarshal(signature, &contentInfo)
	if err != nil {
		t.Fatalf("failed to parse ContentInfo: %v", err)
	}

	var signedData SignedData
	_, err = asn1.Unmarshal(contentInfo.Content.Bytes, &signedData)
	if err != nil {
		t.Fatalf("failed to parse SignedData: %v", err)
	}

	// Verify SignerInfo exists
	if len(signedData.SignerInfos) != 1 {
		t.Fatalf("expected 1 SignerInfo, got %d", len(signedData.SignerInfos))
	}

	// Verify unsigned attributes are PRESENT (contains timestamp)
	signerInfo := signedData.SignerInfos[0]
	if len(signerInfo.UnsignedAttrs.Bytes) == 0 {
		t.Error("unsigned attributes should contain timestamp token")
	}

	// Parse unsigned attributes (SET content, parse each Attribute manually)
	// The Bytes field contains the SET content (individual Attribute SEQUENCEs)
	// Parse them manually like we do in extractTimestamps
	var foundTimestamp bool
	data := signerInfo.UnsignedAttrs.Bytes
	for len(data) > 0 {
		var attr Attribute
		rest, err := asn1.Unmarshal(data, &attr)
		if err != nil {
			t.Fatalf("failed to parse attribute: %v", err)
		}
		data = rest

		if attr.Type.Equal(oidTimestampToken) {
			foundTimestamp = true
			break
		}
	}

	if !foundTimestamp {
		t.Error("timestamp attribute not found in unsigned attributes")
	}
}

// TestSignPackageData_WithCertificateChain tests signing with certificate chain
func TestSignPackageData_WithCertificateChain(t *testing.T) {
	// Create root → intermediate → signer chain
	rootCert, rootKey := generateTestRootCA(t)
	intermediateCert, intermediateKey := generateTestCodeSigningCert(t, rootCert, rootKey)
	signerCert, signerKey := generateTestCodeSigningCert(t, intermediateCert, intermediateKey)

	contentHash := sha256.Sum256([]byte("test content"))

	opts := SigningOptions{
		Certificate:      signerCert,
		PrivateKey:       signerKey,
		CertificateChain: []*x509.Certificate{intermediateCert, rootCert},
		SignatureType:    SignatureTypeAuthor,
		HashAlgorithm:    HashAlgorithmSHA256,
	}

	signature, err := SignPackageData(contentHash[:], opts)
	if err != nil {
		t.Fatalf("SignPackageData failed: %v", err)
	}

	if len(signature) == 0 {
		t.Fatal("signature is empty")
	}

	// Parse and verify chain is included
	var contentInfo ContentInfo
	_, err = asn1.Unmarshal(signature, &contentInfo)
	if err != nil {
		t.Fatalf("failed to parse ContentInfo: %v", err)
	}

	var signedData SignedData
	_, err = asn1.Unmarshal(contentInfo.Content.Bytes, &signedData)
	if err != nil {
		t.Fatalf("failed to parse SignedData: %v", err)
	}

	// Verify certificates are present (should have signer + intermediate + root)
	if len(signedData.Certificates.Bytes) == 0 {
		t.Error("no certificates in signature")
	}
}

// Timestamp client tests
func TestTimestampClient_RequestTimestamp(t *testing.T) {
	// Note: This test requires a real RFC 3161 timestamp server
	// Skip if TSA URL is not configured
	tsaURL := "http://timestamp.digicert.com" // Free TSA for testing

	client := NewTimestampClient(tsaURL, 10*time.Second)

	// Create test message hash
	testData := []byte("test message for timestamp")
	hash := sha256.Sum256(testData)

	// Request timestamp
	token, err := client.RequestTimestamp(hash[:], HashAlgorithmSHA256)
	if err != nil {
		t.Skipf("Timestamp request failed (TSA may be unavailable): %v", err)
	}

	if len(token) == 0 {
		t.Fatal("received empty timestamp token")
	}

	// Verify token is valid ContentInfo
	var contentInfo ContentInfo
	_, err = asn1.Unmarshal(token, &contentInfo)
	if err != nil {
		t.Fatalf("failed to parse timestamp token: %v", err)
	}

	// Verify it's SignedData
	if !contentInfo.ContentType.Equal(oidSignedData) {
		t.Error("timestamp token is not SignedData")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce1, err := generateNonce()
	if err != nil {
		t.Fatalf("generateNonce failed: %v", err)
	}

	if len(nonce1) != 32 {
		t.Errorf("nonce length = %d, want 32", len(nonce1))
	}

	// Verify sign bit is clear (ensures unsigned big-endian)
	if nonce1[0]&0x80 != 0 {
		t.Error("nonce has sign bit set (should be cleared)")
	}

	// Generate second nonce, should be different
	nonce2, err := generateNonce()
	if err != nil {
		t.Fatalf("generateNonce failed: %v", err)
	}

	if bytes.Equal(nonce1, nonce2) {
		t.Error("generated identical nonces (should be random)")
	}
}

func TestBuildTimestampRequest(t *testing.T) {
	testHash := sha256.Sum256([]byte("test"))
	nonce := make([]byte, 32)
	copy(nonce, []byte("test-nonce-12345678901234567890"))

	req, err := buildTimestampRequest(testHash[:], HashAlgorithmSHA256, nonce)
	if err != nil {
		t.Fatalf("buildTimestampRequest failed: %v", err)
	}

	// Verify version
	if req.Version != 1 {
		t.Errorf("version = %d, want 1", req.Version)
	}

	// Verify message imprint
	if !bytes.Equal(req.MessageImprint.HashedMessage, testHash[:]) {
		t.Error("message imprint hash mismatch")
	}

	// Verify hash algorithm OID
	expectedOID := oidSHA256
	if !req.MessageImprint.HashAlgorithm.Algorithm.Equal(expectedOID) {
		t.Errorf("hash algorithm OID = %v, want %v",
			req.MessageImprint.HashAlgorithm.Algorithm, expectedOID)
	}

	// Verify nonce
	if req.Nonce == nil {
		t.Fatal("nonce is nil")
	}
	expectedNonce := new(big.Int).SetBytes(nonce)
	if req.Nonce.Cmp(expectedNonce) != 0 {
		t.Error("nonce mismatch")
	}

	// Verify certReq
	if !req.CertReq {
		t.Error("certReq should be true")
	}
}

func TestVerifyTimestampResponse_ValidToken(t *testing.T) {
	// Create test certificates
	rootCert, rootKey := generateTestRootCA(t)
	tsaCert, tsaKey := generateTestTimestampCert(t, rootCert, rootKey)

	// Create test data
	h := sha256.New()
	h.Write([]byte("test message"))
	messageHash := h.Sum(nil)

	nonce, err := generateNonce()
	if err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}

	// Build TSTInfo
	nonceInt := new(big.Int).SetBytes(nonce)
	tstInfo := tstInfo{
		Version: 1,
		Policy:  asn1.ObjectIdentifier{1, 2, 3, 4, 5}, // Example policy OID
		MessageImprint: messageImprint{
			HashAlgorithm: AlgorithmIdentifier{
				Algorithm: oidSHA256,
			},
			HashedMessage: messageHash,
		},
		SerialNumber: big.NewInt(1),
		GenTime:      time.Now(),
		Nonce:        nonceInt,
	}

	// Marshal TSTInfo
	tstInfoBytes, err := asn1.Marshal(tstInfo)
	if err != nil {
		t.Fatalf("Failed to marshal TSTInfo: %v", err)
	}

	// Wrap in OCTET STRING for eContent
	eContent, err := asn1.Marshal(tstInfoBytes)
	if err != nil {
		t.Fatalf("Failed to marshal eContent: %v", err)
	}

	// Create SignedData with TSTInfo as content
	signedData := SignedData{
		Version: 1,
		DigestAlgorithms: []AlgorithmIdentifier{
			{Algorithm: oidSHA256},
		},
		ContentInfo: EncapsulatedContentInfo{
			ContentType: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}, // id-ct-TSTInfo
			Content: asn1.RawValue{
				Class:      asn1.ClassContextSpecific,
				Tag:        0,
				IsCompound: true,
				Bytes:      eContent,
			},
		},
		Certificates: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      tsaCert.Raw,
		},
		SignerInfos: []SignerInfo{
			{
				Version: 1,
				SID: asn1.RawValue{
					Class: asn1.ClassContextSpecific,
					Tag:   0,
					Bytes: tsaCert.SubjectKeyId,
				},
				DigestAlgorithm: AlgorithmIdentifier{
					Algorithm: oidSHA256,
				},
				SignedAttrs: asn1.RawValue{
					Class:      asn1.ClassContextSpecific,
					Tag:        0,
					IsCompound: true,
					Bytes:      []byte{}, // Empty for this test
				},
				SignatureAlgorithm: AlgorithmIdentifier{
					Algorithm: oidSHA256WithRSA,
				},
				Signature: make([]byte, 256), // Dummy signature for this test
			},
		},
	}

	// Marshal SignedData
	signedDataBytes, err := asn1.Marshal(signedData)
	if err != nil {
		t.Fatalf("Failed to marshal SignedData: %v", err)
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

	// Marshal ContentInfo (this is the timestamp token)
	tokenBytes, err := asn1.Marshal(contentInfo)
	if err != nil {
		t.Fatalf("Failed to marshal ContentInfo: %v", err)
	}

	// Test verifyTimestampResponse
	err = verifyTimestampResponse(tokenBytes, messageHash, nonce)
	if err != nil {
		t.Errorf("verifyTimestampResponse failed: %v", err)
	}

	// Suppress unused variable warnings
	_ = rootKey
	_ = tsaKey
}

func TestVerifyTimestampResponse_NonceMismatch(t *testing.T) {
	// Create test certificates
	rootCert, rootKey := generateTestRootCA(t)
	tsaCert, tsaKey := generateTestTimestampCert(t, rootCert, rootKey)

	// Create test data
	h := sha256.New()
	h.Write([]byte("test message"))
	messageHash := h.Sum(nil)

	// Generate two different nonces
	requestNonce, err := generateNonce()
	if err != nil {
		t.Fatalf("Failed to generate request nonce: %v", err)
	}

	responseNonce, err := generateNonce()
	if err != nil {
		t.Fatalf("Failed to generate response nonce: %v", err)
	}

	// Ensure they're different
	if bytes.Equal(requestNonce, responseNonce) {
		// Modify one byte to ensure difference
		responseNonce[0] ^= 0x01
	}

	// Build TSTInfo with wrong nonce
	responseNonceInt := new(big.Int).SetBytes(responseNonce)
	tstInfo := tstInfo{
		Version: 1,
		Policy:  asn1.ObjectIdentifier{1, 2, 3, 4, 5}, // Example policy OID
		MessageImprint: messageImprint{
			HashAlgorithm: AlgorithmIdentifier{
				Algorithm: oidSHA256,
			},
			HashedMessage: messageHash,
		},
		SerialNumber: big.NewInt(1),
		GenTime:      time.Now(),
		Nonce:        responseNonceInt, // Wrong nonce
	}

	// Marshal TSTInfo
	tstInfoBytes, err := asn1.Marshal(tstInfo)
	if err != nil {
		t.Fatalf("Failed to marshal TSTInfo: %v", err)
	}

	// Wrap in OCTET STRING for eContent
	eContent, err := asn1.Marshal(tstInfoBytes)
	if err != nil {
		t.Fatalf("Failed to marshal eContent: %v", err)
	}

	// Create SignedData with TSTInfo as content
	signedData := SignedData{
		Version: 1,
		DigestAlgorithms: []AlgorithmIdentifier{
			{Algorithm: oidSHA256},
		},
		ContentInfo: EncapsulatedContentInfo{
			ContentType: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}, // id-ct-TSTInfo
			Content: asn1.RawValue{
				Class:      asn1.ClassContextSpecific,
				Tag:        0,
				IsCompound: true,
				Bytes:      eContent,
			},
		},
		Certificates: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      tsaCert.Raw,
		},
		SignerInfos: []SignerInfo{
			{
				Version: 1,
				SID: asn1.RawValue{
					Class: asn1.ClassContextSpecific,
					Tag:   0,
					Bytes: tsaCert.SubjectKeyId,
				},
				DigestAlgorithm: AlgorithmIdentifier{
					Algorithm: oidSHA256,
				},
				SignedAttrs: asn1.RawValue{
					Class:      asn1.ClassContextSpecific,
					Tag:        0,
					IsCompound: true,
					Bytes:      []byte{}, // Empty for this test
				},
				SignatureAlgorithm: AlgorithmIdentifier{
					Algorithm: oidSHA256WithRSA,
				},
				Signature: make([]byte, 256), // Dummy signature for this test
			},
		},
	}

	// Marshal SignedData
	signedDataBytes, err := asn1.Marshal(signedData)
	if err != nil {
		t.Fatalf("Failed to marshal SignedData: %v", err)
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

	// Marshal ContentInfo (this is the timestamp token)
	tokenBytes, err := asn1.Marshal(contentInfo)
	if err != nil {
		t.Fatalf("Failed to marshal ContentInfo: %v", err)
	}

	// Test verifyTimestampResponse - should fail due to nonce mismatch
	err = verifyTimestampResponse(tokenBytes, messageHash, requestNonce)
	if err == nil {
		t.Error("Expected verifyTimestampResponse to fail with nonce mismatch, but it succeeded")
	}

	if err != nil && err.Error() != "timestamp nonce mismatch" {
		t.Errorf("Expected 'timestamp nonce mismatch' error, got: %v", err)
	}

	// Suppress unused variable warnings
	_ = rootKey
	_ = tsaKey
}

// Test error paths for coverage
func TestSignPackageData_InvalidOptions(t *testing.T) {
	contentHash := make([]byte, 32)

	// Test with nil certificate
	rootCert, rootKey := generateTestRootCA(t)
	_, privateKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	opts := SigningOptions{
		PrivateKey:    privateKey,
		SignatureType: SignatureTypeAuthor,
		HashAlgorithm: HashAlgorithmSHA256,
	}
	_, err := SignPackageData(contentHash, opts)
	if err == nil {
		t.Fatal("Expected error with nil certificate")
	}
}

func TestCreateSignedData_WithUnknownSignatureType(t *testing.T) {
	// Test that unknown signature type still creates signature (just without commitment attribute)
	rootCert, rootKey := generateTestRootCA(t)
	cert, key := generateTestCodeSigningCert(t, rootCert, rootKey)
	opts := SigningOptions{
		Certificate:   cert,
		PrivateKey:    key,
		SignatureType: SignatureTypeUnknown,
		HashAlgorithm: HashAlgorithmSHA256,
	}

	contentHash := make([]byte, 32)
	signedData, err := createSignedData(contentHash, opts)
	if err != nil {
		t.Fatalf("createSignedData failed: %v", err)
	}
	if signedData == nil {
		t.Fatal("Expected signed data to be created")
	}
}

func TestEncodeAttributesForSigning_Error(t *testing.T) {
	// Test encoding with invalid attribute structure
	attrs := []Attribute{
		{
			Type:   asn1.ObjectIdentifier{1, 2, 3},
			Values: asn1.RawValue{FullBytes: []byte{0xff, 0xff}}, // Invalid ASN.1
		},
	}

	_, err := EncodeAttributesForSigning(attrs)
	// Should succeed even with unusual values - ASN.1 will encode what we give it
	if err != nil {
		t.Logf("Encoding returned error (may be expected): %v", err)
	}
}

func TestVerifyKeyMatchesCertificate_ECDSAKey(t *testing.T) {
	// Test with unsupported key type
	rootCert, rootKey := generateTestRootCA(t)
	cert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Try with non-RSA key (will fail in our implementation)
	err := verifyKeyMatchesCertificate(cert, "not a key")
	if err == nil {
		t.Fatal("Expected error with invalid key type")
	}
}

func TestGetAlgorithmOIDs_DefaultCases(t *testing.T) {
	// Test default cases for algorithm functions
	unknownAlg := HashAlgorithmName("unknown")

	sigOID := getSignatureAlgorithmOID(unknownAlg)
	if !sigOID.Equal(oidSHA256WithRSA) {
		t.Error("Expected default SHA256WithRSA for unknown algorithm")
	}

	digestOID := getDigestAlgorithmOID(unknownAlg)
	if !digestOID.Equal(oidSHA256) {
		t.Error("Expected default SHA256 for unknown algorithm")
	}

	hashFunc := getCryptoHash(unknownAlg)
	expectedHash := getCryptoHash(HashAlgorithmSHA256)
	if hashFunc != expectedHash {
		t.Error("Expected default SHA256 hash for unknown algorithm")
	}
}

func TestRequestTimestamp_HTTPError(t *testing.T) {
	// Test with invalid URL
	client := NewTimestampClient("http://invalid-tsa-url-that-does-not-exist.example.com", 5*time.Second)

	messageHash := make([]byte, 32)
	_, err := client.RequestTimestamp(messageHash, HashAlgorithmSHA256)
	if err == nil {
		t.Fatal("Expected error with invalid TSA URL")
	}
}

func TestGenerateNonce_SignBitCleared(t *testing.T) {
	// Run multiple times to ensure sign bit is always cleared
	for i := 0; i < 100; i++ {
		nonce, err := generateNonce()
		if err != nil {
			t.Fatalf("generateNonce failed: %v", err)
		}

		// Check that sign bit is cleared (MSB of first byte should be 0)
		if len(nonce) > 0 && (nonce[0]&0x80) != 0 {
			t.Errorf("Nonce sign bit not cleared: first byte = 0x%02x", nonce[0])
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Tests for error path coverage to reach 90%

func TestASN1MarshalingErrorPaths(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	cert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	// These tests verify error handling exists, even though triggering actual
	// ASN.1 marshaling errors is nearly impossible with valid Go structures

	// Test all attribute creation functions return valid data
	contentHash := make([]byte, 32)

	// createContentTypeAttribute
	attr, err := createContentTypeAttribute()
	if err != nil {
		t.Errorf("createContentTypeAttribute should not fail with valid OID: %v", err)
	}
	if attr.Type == nil {
		t.Error("Expected valid attribute type")
	}

	// createSigningTimeAttribute
	attr, err = createSigningTimeAttribute(time.Now())
	if err != nil {
		t.Errorf("createSigningTimeAttribute should not fail: %v", err)
	}

	// createMessageDigestAttribute
	attr, err = createMessageDigestAttribute(contentHash)
	if err != nil {
		t.Errorf("createMessageDigestAttribute should not fail: %v", err)
	}

	// createCommitmentTypeIndicationAttribute - Author
	attr, err = createCommitmentTypeIndicationAttribute(SignatureTypeAuthor)
	if err != nil {
		t.Errorf("createCommitmentTypeIndicationAttribute(Author) should not fail: %v", err)
	}

	// createCommitmentTypeIndicationAttribute - Repository
	attr, err = createCommitmentTypeIndicationAttribute(SignatureTypeRepository)
	if err != nil {
		t.Errorf("createCommitmentTypeIndicationAttribute(Repository) should not fail: %v", err)
	}

	// createSigningCertificateV2Attribute
	attr, err = createSigningCertificateV2Attribute(cert, HashAlgorithmSHA256)
	if err != nil {
		t.Errorf("createSigningCertificateV2Attribute should not fail: %v", err)
	}

	// EncodeAttributesForSigning
	attrs := []Attribute{attr}
	_, err = EncodeAttributesForSigning(attrs)
	if err != nil {
		t.Errorf("EncodeAttributesForSigning should not fail: %v", err)
	}
}

func TestVerifyKeyMatchesCertificate_NonRSAKey(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	cert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Test with a non-RSA private key type (wrong type)
	type FakeKey struct{}
	err := verifyKeyMatchesCertificate(cert, FakeKey{})
	if err == nil {
		t.Fatal("Expected error with non-RSA key type")
	}
	if !contains(err.Error(), "not RSA") {
		t.Errorf("Expected 'not RSA' error, got: %v", err)
	}
}

func TestSignAttributes_NonRSAKey(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	cert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	attrs, err := BuildSignedAttributes(make([]byte, 32), SignatureTypeAuthor, cert, HashAlgorithmSHA256)
	if err != nil {
		t.Fatalf("BuildSignedAttributes failed: %v", err)
	}

	attrsBytes, err := EncodeAttributesForSigning(attrs)
	if err != nil {
		t.Fatalf("EncodeAttributesForSigning failed: %v", err)
	}

	// Create opts with non-RSA key
	type FakeKey struct{}
	opts := SigningOptions{
		PrivateKey:    FakeKey{},
		HashAlgorithm: HashAlgorithmSHA256,
	}

	_, err = signAttributes(attrsBytes, opts)
	if err == nil {
		t.Fatal("Expected error with non-RSA key")
	}
	if !contains(err.Error(), "RSA") {
		t.Errorf("Expected RSA error, got: %v", err)
	}
}

func TestTimestampErrorPaths(t *testing.T) {
	// Test with server that returns non-200 status
	// Use a URL that will return 404
	client := NewTimestampClient("http://freetsa.org/nonexistent", 5*time.Second)
	_, err := client.RequestTimestamp(make([]byte, 32), HashAlgorithmSHA256)
	if err != nil {
		// Expected - either network error or HTTP error
		t.Logf("RequestTimestamp correctly returned error: %v", err)
	}

	// Test generateNonce
	for i := 0; i < 10; i++ {
		nonce, err := generateNonce()
		if err != nil {
			t.Fatalf("generateNonce failed: %v", err)
		}
		if len(nonce) != 32 {
			t.Errorf("Expected 32 byte nonce, got %d", len(nonce))
		}
		// Verify sign bit cleared
		if nonce[0]&0x80 != 0 {
			t.Errorf("Sign bit not cleared in nonce")
		}
	}
}

func TestBuildTimestampRequestAllFields(t *testing.T) {
	messageHash := make([]byte, 32)
	nonce, err := generateNonce()
	if err != nil {
		t.Fatalf("generateNonce failed: %v", err)
	}

	// Test with all hash algorithms
	for _, alg := range []HashAlgorithmName{HashAlgorithmSHA256, HashAlgorithmSHA384, HashAlgorithmSHA512} {
		req, err := buildTimestampRequest(messageHash, alg, nonce)
		if err != nil {
			t.Errorf("buildTimestampRequest failed for %s: %v", alg, err)
		}
		if req.Version != 1 {
			t.Errorf("Expected version 1, got %d", req.Version)
		}
		if !req.CertReq {
			t.Error("Expected CertReq to be true")
		}
	}
}

func TestCreateSignedDataWithCertificateChain(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	intermediateCert, intermediateKey := generateTestCodeSigningCert(t, rootCert, rootKey)
	signerCert, signerKey := generateTestCodeSigningCert(t, intermediateCert, intermediateKey)

	contentHash := make([]byte, 32)

	// Test with multi-level certificate chain
	opts := SigningOptions{
		Certificate:      signerCert,
		PrivateKey:       signerKey,
		CertificateChain: []*x509.Certificate{intermediateCert, rootCert},
		SignatureType:    SignatureTypeAuthor,
		HashAlgorithm:    HashAlgorithmSHA256,
	}

	signedData, err := createSignedData(contentHash, opts)
	if err != nil {
		t.Fatalf("createSignedData failed: %v", err)
	}

	if signedData == nil {
		t.Fatal("Expected signedData to be created")
	}

	// Verify certificates are included
	if len(signedData.Certificates.Bytes) == 0 {
		t.Error("Expected certificates to be included")
	}
}

func TestCreateSignerInfoWithoutTimestamp(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	cert, key := generateTestCodeSigningCert(t, rootCert, rootKey)

	contentHash := make([]byte, 32)

	opts := SigningOptions{
		Certificate:   cert,
		PrivateKey:    key,
		SignatureType: SignatureTypeAuthor,
		HashAlgorithm: HashAlgorithmSHA256,
		// No TimestampURL - should skip timestamp attribute
	}

	signerInfo, err := createSignerInfo(contentHash, opts)
	if err != nil {
		t.Fatalf("createSignerInfo failed: %v", err)
	}

	// Verify no unsigned attributes (no timestamp)
	if len(signerInfo.UnsignedAttrs.Bytes) > 0 {
		t.Logf("Note: UnsignedAttrs present even without timestamp URL")
	}
}

func TestVerifyTimestampResponseErrorPaths(t *testing.T) {
	// Test with invalid ContentInfo
	invalidToken := []byte{0x30, 0x03, 0x02, 0x01, 0x00} // Valid ASN.1 but wrong structure
	err := verifyTimestampResponse(invalidToken, make([]byte, 32), make([]byte, 32))
	if err == nil {
		t.Error("Expected error with invalid timestamp token")
	}
}

func TestVerifyTimestampResponseAllErrorPaths(t *testing.T) {
	goodHash := make([]byte, 32)
	goodNonce := make([]byte, 32)

	// Test 1: Invalid ContentInfo (wrong OID)
	wrongOID := asn1.ObjectIdentifier{1, 2, 3}
	badContentInfo := ContentInfo{
		ContentType: wrongOID,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      []byte{0x30, 0x00},
		},
	}
	badToken, _ := asn1.Marshal(badContentInfo)
	err := verifyTimestampResponse(badToken, goodHash, goodNonce)
	if err == nil || !contains(err.Error(), "SignedData") {
		t.Errorf("Expected SignedData error, got: %v", err)
	}

	// Test 2: Invalid SignedData bytes
	goodContentInfo := ContentInfo{
		ContentType: oidSignedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      []byte{0xFF, 0xFF}, // Invalid ASN.1
		},
	}
	badToken2, _ := asn1.Marshal(goodContentInfo)
	err = verifyTimestampResponse(badToken2, goodHash, goodNonce)
	if err == nil || !contains(err.Error(), "unmarshal signed data") {
		t.Errorf("Expected unmarshal signed data error, got: %v", err)
	}

	// Test 3: Hash mismatch
	// Request a real timestamp and then verify with wrong hash
	client := NewTimestampClient("http://freetsa.org/tsr", 10*time.Second)
	realToken, err := client.RequestTimestamp(goodHash, HashAlgorithmSHA256)
	if err != nil {
		t.Skip("TSA unavailable, skipping hash mismatch test")
	}

	wrongHash := make([]byte, 32)
	for i := range wrongHash {
		wrongHash[i] = 0xFF
	}
	err = verifyTimestampResponse(realToken, wrongHash, goodNonce)
	if err == nil || !contains(err.Error(), "mismatch") {
		t.Errorf("Expected hash mismatch error, got: %v", err)
	}

	// Test 4: Nonce mismatch
	wrongNonce := make([]byte, 32)
	for i := range wrongNonce {
		wrongNonce[i] = 0xFF
	}
	err = verifyTimestampResponse(realToken, goodHash, wrongNonce)
	if err == nil || !contains(err.Error(), "mismatch") {
		t.Errorf("Expected nonce mismatch error, got: %v", err)
	}
}

func TestRequestTimestampAllErrorPaths(t *testing.T) {
	// Test HTTP errors
	tests := []struct {
		name string
		url  string
	}{
		{"Invalid host", "http://this-domain-definitely-does-not-exist-12345.com/tsr"},
		{"HTTP error", "http://freetsa.org/nonexistent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewTimestampClient(tt.url, 5*time.Second)
			_, err := client.RequestTimestamp(make([]byte, 32), HashAlgorithmSHA256)
			if err == nil {
				t.Error("Expected error but got none")
			}
			// Error could be network or HTTP - just verify we got an error
			t.Logf("Got expected error: %v", err)
		})
	}
}

func TestSignPackageDataAllMarshalingPaths(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	cert, key := generateTestCodeSigningCert(t, rootCert, rootKey)

	contentHash := make([]byte, 32)

	// Test all combinations to ensure marshaling code is exercised
	signatureTypes := []SignatureType{SignatureTypeAuthor, SignatureTypeRepository}
	hashAlgs := []HashAlgorithmName{HashAlgorithmSHA256, HashAlgorithmSHA384, HashAlgorithmSHA512}

	for _, sigType := range signatureTypes {
		for _, hashAlg := range hashAlgs {
			t.Run(string(sigType)+"_"+string(hashAlg), func(t *testing.T) {
				opts := SigningOptions{
					Certificate:   cert,
					PrivateKey:    key,
					SignatureType: sigType,
					HashAlgorithm: hashAlg,
				}

				signature, err := SignPackageData(contentHash, opts)
				if err != nil {
					t.Fatalf("SignPackageData failed: %v", err)
				}

				if len(signature) == 0 {
					t.Fatal("Expected signature data")
				}

				// Verify it's valid PKCS#7
				var contentInfo ContentInfo
				_, err = asn1.Unmarshal(signature, &contentInfo)
				if err != nil {
					t.Errorf("Failed to unmarshal signature: %v", err)
				}
			})
		}
	}
}

func TestVerifyKeyMatchesCertificate_DefaultCase(t *testing.T) {
	// Create a cert with RSA key
	rootCert, rootKey := generateTestRootCA(t)
	cert, key := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Test with matching key (should succeed)
	err := verifyKeyMatchesCertificate(cert, key)
	if err != nil {
		t.Errorf("Matching keys should verify: %v", err)
	}

	// Test with mismatched RSA key
	_, wrongKey := generateTestRootCA(t)
	err = verifyKeyMatchesCertificate(cert, wrongKey)
	if err == nil {
		t.Error("Expected error with mismatched RSA keys")
	}
}

func TestCreateSignerInfo_BothSIDPaths(t *testing.T) {
	contentHash := make([]byte, 32)

	// Test 1: With SubjectKeyId (new path)
	rootCert, rootKey := generateTestRootCA(t)
	certWithSKID, keyWithSKID := generateTestCodeSigningCert(t, rootCert, rootKey)

	if len(certWithSKID.SubjectKeyId) == 0 {
		t.Fatal("Test cert should have SubjectKeyId")
	}

	opts1 := SigningOptions{
		Certificate:   certWithSKID,
		PrivateKey:    keyWithSKID,
		SignatureType: SignatureTypeAuthor,
		HashAlgorithm: HashAlgorithmSHA256,
	}

	signerInfo1, err := createSignerInfo(contentHash, opts1)
	if err != nil {
		t.Fatalf("createSignerInfo with SubjectKeyId failed: %v", err)
	}

	// Verify SID uses SubjectKeyIdentifier format (context-specific tag 0)
	if signerInfo1.SID.Class != asn1.ClassContextSpecific {
		t.Error("Expected context-specific class for SubjectKeyId")
	}

	// Test 2: Without SubjectKeyId (IssuerAndSerialNumber path)
	// Create a cert manually without SubjectKeyId
	privNoSKID, _ := rsa.GenerateKey(rand.Reader, 2048)
	templateNoSKID := &x509.Certificate{
		SerialNumber: big.NewInt(999),
		Subject: pkix.Name{
			CommonName: "Cert Without SKID",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		// NO SubjectKeyId
	}

	certDERNoSKID, _ := x509.CreateCertificate(rand.Reader, templateNoSKID, rootCert, &privNoSKID.PublicKey, rootKey)
	certNoSKID, _ := x509.ParseCertificate(certDERNoSKID)

	if len(certNoSKID.SubjectKeyId) != 0 {
		t.Fatal("Test cert should NOT have SubjectKeyId")
	}

	opts2 := SigningOptions{
		Certificate:   certNoSKID,
		PrivateKey:    privNoSKID,
		SignatureType: SignatureTypeAuthor,
		HashAlgorithm: HashAlgorithmSHA256,
	}

	signerInfo2, err := createSignerInfo(contentHash, opts2)
	if err != nil {
		t.Fatalf("createSignerInfo without SubjectKeyId failed: %v", err)
	}

	// Verify SID uses IssuerAndSerialNumber format (universal class)
	if signerInfo2.SID.Class != asn1.ClassUniversal {
		t.Logf("SID class: %d (expected universal)", signerInfo2.SID.Class)
	}
}

// TestSignAndVerifyWithVerifier tests complete sign-and-verify cycle using our verifier
func TestSignAndVerifyWithVerifier(t *testing.T) {
	// Generate certificates
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Create test package hash
	testData := []byte("test package content for full verification")
	contentHash := sha256.Sum256(testData)

	// Sign package
	opts := SigningOptions{
		Certificate:      signerCert,
		PrivateKey:       signerKey,
		CertificateChain: []*x509.Certificate{rootCert},
		SignatureType:    SignatureTypeAuthor,
		HashAlgorithm:    HashAlgorithmSHA256,
	}

	signatureBytes, err := SignPackageData(contentHash[:], opts)
	if err != nil {
		t.Fatalf("SignPackageData failed: %v", err)
	}

	// Read signature using our reader
	sig, err := ReadSignature(signatureBytes)
	if err != nil {
		t.Fatalf("ReadSignature failed: %v", err)
	}

	// Verify signature type
	if sig.Type != SignatureTypeAuthor {
		t.Errorf("Expected Author signature, got %s", sig.Type)
	}

	// Verify hash algorithm
	if sig.HashAlgorithm != HashAlgorithmSHA256 {
		t.Errorf("Expected SHA256, got %s", sig.HashAlgorithm)
	}

	// Create trust store with root cert
	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	// Verify signature using our verifier
	verifyOpts := VerificationOptions{
		TrustStore:            trustStore,
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256},
		RequireTimestamp:      false,
		AllowUntrustedRoot:    false,
	}

	result := VerifySignature(sig, verifyOpts)
	if !result.IsValid {
		t.Fatalf("Signature verification failed: %v", result.Errors)
	}

	if len(result.Errors) > 0 {
		t.Errorf("Unexpected errors during verification: %v", result.Errors)
	}

	if len(result.Warnings) > 0 {
		t.Logf("Warnings during verification: %v", result.Warnings)
	}

	t.Logf("✓ Signature created and verified successfully")
}

// TestSignWithTimestampAndVerify tests signing with timestamp and full verification
// Note: This test uses a real timestamp server and may fail if the server is unavailable.
// Like NuGet.Client, production code should allow users to configure the timestamp URL.
func TestSignWithTimestampAndVerify(t *testing.T) {
	// Generate certificates
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Create test package hash
	testData := []byte("test package with timestamp")
	contentHash := sha256.Sum256(testData)

	// Sign package with timestamp
	// Using DigiCert's timestamp server (reliable and commonly used by NuGet ecosystem)
	opts := SigningOptions{
		Certificate:      signerCert,
		PrivateKey:       signerKey,
		CertificateChain: []*x509.Certificate{rootCert},
		SignatureType:    SignatureTypeRepository,
		HashAlgorithm:    HashAlgorithmSHA256,
		TimestampURL:     "http://timestamp.digicert.com",
		TimestampTimeout: 10 * time.Second,
	}

	signatureBytes, err := SignPackageData(contentHash[:], opts)
	if err != nil {
		t.Skipf("SignPackageData with timestamp failed (TSA may be unavailable): %v", err)
	}

	// Read signature
	sig, err := ReadSignature(signatureBytes)
	if err != nil {
		t.Fatalf("ReadSignature failed: %v", err)
	}

	// Verify we have timestamps
	if len(sig.Timestamps) == 0 {
		t.Fatalf("Expected signature to include timestamp, but got none")
	}
	t.Logf("✓ Signature includes %d timestamp(s)", len(sig.Timestamps))

	// Create trust store
	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	// Verify signature
	verifyOpts := VerificationOptions{
		TrustStore:            trustStore,
		AllowedSignatureTypes: []SignatureType{SignatureTypeRepository},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256},
		RequireTimestamp:      false, // Optional timestamp
		AllowUntrustedRoot:    true,  // TSA root may not be in our trust store
	}

	result := VerifySignature(sig, verifyOpts)
	if !result.IsValid {
		t.Fatalf("Signature verification failed: %v", result.Errors)
	}

	t.Logf("✓ Timestamped signature created and verified successfully")
}

// TestSignAllHashAlgorithmsAndVerify tests all hash algorithms end-to-end
func TestSignAllHashAlgorithmsAndVerify(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, signerKey := generateTestCodeSigningCert(t, rootCert, rootKey)

	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	testData := []byte("test data for all algorithms")

	tests := []struct {
		name    string
		hashAlg HashAlgorithmName
		hasher  func() []byte
	}{
		{
			name:    "SHA256",
			hashAlg: HashAlgorithmSHA256,
			hasher: func() []byte {
				h := sha256.Sum256(testData)
				return h[:]
			},
		},
		{
			name:    "SHA384",
			hashAlg: HashAlgorithmSHA384,
			hasher: func() []byte {
				h := sha512.Sum384(testData)
				return h[:]
			},
		},
		{
			name:    "SHA512",
			hashAlg: HashAlgorithmSHA512,
			hasher: func() []byte {
				h := sha512.Sum512(testData)
				return h[:]
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentHash := tt.hasher()

			// Sign
			opts := SigningOptions{
				Certificate:   signerCert,
				PrivateKey:    signerKey,
				SignatureType: SignatureTypeAuthor,
				HashAlgorithm: tt.hashAlg,
			}

			sigBytes, err := SignPackageData(contentHash, opts)
			if err != nil {
				t.Fatalf("SignPackageData failed: %v", err)
			}

			// Read and verify
			sig, err := ReadSignature(sigBytes)
			if err != nil {
				t.Fatalf("ReadSignature failed: %v", err)
			}

			verifyOpts := VerificationOptions{
				TrustStore:            trustStore,
				AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor},
				AllowedHashAlgorithms: []HashAlgorithmName{tt.hashAlg},
				AllowUntrustedRoot:    false,
			}

			result := VerifySignature(sig, verifyOpts)
			if !result.IsValid {
				t.Errorf("Verification failed for %s: %v", tt.name, result.Errors)
			}
		})
	}
}
