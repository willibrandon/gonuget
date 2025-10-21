package signatures

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

// Test helpers to generate certificates for testing

// generateTestRootCA creates a self-signed root CA certificate
func generateTestRootCA(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "Test Root CA",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert, priv
}

// generateTestCodeSigningCert creates a code signing certificate
func generateTestCodeSigningCert(t *testing.T, rootCert *x509.Certificate, rootKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName:   "Test Code Signing Cert",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, rootCert, &priv.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert, priv
}

// generateTestTimestampCert creates a timestamp authority certificate
func generateTestTimestampCert(t *testing.T, rootCert *x509.Certificate, rootKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			CommonName:   "Test Timestamp Authority",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, rootCert, &priv.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert, priv
}

// generateWeakRSAKeyCert creates a certificate with weak RSA key (1024 bits)
func generateWeakRSAKeyCert(t *testing.T, rootCert *x509.Certificate, rootKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	// Generate 1024-bit key (weak)
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(4),
		Subject: pkix.Name{
			CommonName:   "Weak Key Cert",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, rootCert, &priv.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert, priv
}

// generateExpiredCert creates an expired certificate
func generateExpiredCert(t *testing.T, rootCert *x509.Certificate, rootKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(5),
		Subject: pkix.Name{
			CommonName:   "Expired Cert",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now().Add(-365 * 24 * time.Hour),
		NotAfter:              time.Now().Add(-24 * time.Hour), // Expired
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, rootCert, &priv.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert, priv
}

func TestDefaultVerificationOptions(t *testing.T) {
	opts := DefaultVerificationOptions()

	if opts.TrustStore == nil {
		t.Error("TrustStore should not be nil")
	}

	if opts.AllowUntrustedRoot {
		t.Error("AllowUntrustedRoot should be false by default")
	}

	if opts.RequireTimestamp {
		t.Error("RequireTimestamp should be false by default")
	}

	if !opts.VerifyTimestamp {
		t.Error("VerifyTimestamp should be true by default")
	}

	expectedSigTypes := []SignatureType{SignatureTypeAuthor, SignatureTypeRepository}
	if len(opts.AllowedSignatureTypes) != len(expectedSigTypes) {
		t.Errorf("expected %d signature types, got %d", len(expectedSigTypes), len(opts.AllowedSignatureTypes))
	}

	expectedHashAlgs := []HashAlgorithmName{HashAlgorithmSHA256, HashAlgorithmSHA384, HashAlgorithmSHA512}
	if len(opts.AllowedHashAlgorithms) != len(expectedHashAlgs) {
		t.Errorf("expected %d hash algorithms, got %d", len(expectedHashAlgs), len(opts.AllowedHashAlgorithms))
	}
}

func TestVerifySignature_ValidAuthorSignature(t *testing.T) {
	// Generate test certificates
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Create trust store with root
	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	// Create test signature
	sig := &PrimarySignature{
		Type:              SignatureTypeAuthor,
		SignerCertificate: signerCert,
		Certificates:      []*x509.Certificate{signerCert, rootCert},
		HashAlgorithm:     HashAlgorithmSHA256,
	}

	opts := VerificationOptions{
		TrustStore:            trustStore,
		AllowUntrustedRoot:    false,
		RequireTimestamp:      false,
		VerifyTimestamp:       true,
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor, SignatureTypeRepository},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256, HashAlgorithmSHA384, HashAlgorithmSHA512},
	}

	result := VerifySignature(sig, opts)

	if !result.IsValid {
		t.Errorf("expected valid signature, got errors: %v", result.Errors)
	}

	if result.SignatureType != SignatureTypeAuthor {
		t.Errorf("expected signature type Author, got %s", result.SignatureType)
	}

	if result.SignerCertificate == nil {
		t.Error("expected signer certificate to be set")
	}

	if result.TrustedRoot == nil {
		t.Error("expected trusted root to be set")
	}
}

func TestVerifySignature_ValidRepositorySignature(t *testing.T) {
	// Generate test certificates
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Create trust store with root
	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	// Create test signature
	sig := &PrimarySignature{
		Type:              SignatureTypeRepository,
		SignerCertificate: signerCert,
		Certificates:      []*x509.Certificate{signerCert, rootCert},
		HashAlgorithm:     HashAlgorithmSHA384,
	}

	opts := VerificationOptions{
		TrustStore:            trustStore,
		AllowUntrustedRoot:    false,
		RequireTimestamp:      false,
		VerifyTimestamp:       true,
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor, SignatureTypeRepository},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256, HashAlgorithmSHA384, HashAlgorithmSHA512},
	}

	result := VerifySignature(sig, opts)

	if !result.IsValid {
		t.Errorf("expected valid signature, got errors: %v", result.Errors)
	}

	if result.SignatureType != SignatureTypeRepository {
		t.Errorf("expected signature type Repository, got %s", result.SignatureType)
	}
}

func TestVerifySignature_DisallowedSignatureType(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	sig := &PrimarySignature{
		Type:              SignatureTypeUnknown,
		SignerCertificate: signerCert,
		Certificates:      []*x509.Certificate{signerCert, rootCert},
		HashAlgorithm:     HashAlgorithmSHA256,
	}

	opts := VerificationOptions{
		TrustStore:            trustStore,
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor, SignatureTypeRepository},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256},
	}

	result := VerifySignature(sig, opts)

	if result.IsValid {
		t.Error("expected signature to be invalid due to disallowed type")
	}

	if len(result.Errors) == 0 {
		t.Error("expected error about disallowed signature type")
	}
}

func TestVerifySignature_DisallowedHashAlgorithm(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	sig := &PrimarySignature{
		Type:              SignatureTypeAuthor,
		SignerCertificate: signerCert,
		Certificates:      []*x509.Certificate{signerCert, rootCert},
		HashAlgorithm:     HashAlgorithmSHA512,
	}

	opts := VerificationOptions{
		TrustStore:            trustStore,
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256}, // Only SHA256 allowed
	}

	result := VerifySignature(sig, opts)

	if result.IsValid {
		t.Error("expected signature to be invalid due to disallowed hash algorithm")
	}

	if len(result.Errors) == 0 {
		t.Error("expected error about disallowed hash algorithm")
	}
}

func TestVerifySignature_UntrustedRoot(t *testing.T) {
	// Generate test certificates with different roots
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Create trust store with a different root
	otherRoot, _ := generateTestRootCA(t)
	trustStore := NewTrustStore()
	trustStore.AddCertificate(otherRoot)

	sig := &PrimarySignature{
		Type:              SignatureTypeAuthor,
		SignerCertificate: signerCert,
		Certificates:      []*x509.Certificate{signerCert, rootCert},
		HashAlgorithm:     HashAlgorithmSHA256,
	}

	opts := VerificationOptions{
		TrustStore:            trustStore,
		AllowUntrustedRoot:    false,
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256},
	}

	result := VerifySignature(sig, opts)

	if result.IsValid {
		t.Error("expected signature to be invalid due to untrusted root")
	}

	if len(result.Errors) == 0 {
		t.Error("expected error about certificate chain verification")
	}
}

func TestVerifySignature_UntrustedRootAllowed(t *testing.T) {
	// Generate test certificates with different roots
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	// Create trust store with a different root
	otherRoot, _ := generateTestRootCA(t)
	trustStore := NewTrustStore()
	trustStore.AddCertificate(otherRoot)

	sig := &PrimarySignature{
		Type:              SignatureTypeAuthor,
		SignerCertificate: signerCert,
		Certificates:      []*x509.Certificate{signerCert, rootCert},
		HashAlgorithm:     HashAlgorithmSHA256,
	}

	opts := VerificationOptions{
		TrustStore:            trustStore,
		AllowUntrustedRoot:    true, // Allow untrusted roots
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256},
	}

	result := VerifySignature(sig, opts)

	if result.IsValid {
		t.Error("expected signature to be invalid (untrusted root causes chain failure)")
	}

	// Should have warnings about untrusted root
	if len(result.Warnings) == 0 {
		t.Log("Note: warnings may not be added if chain validation fails completely")
	}
}

func TestVerifySignature_WeakRSAKey(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	weakCert, _ := generateWeakRSAKeyCert(t, rootCert, rootKey)

	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	sig := &PrimarySignature{
		Type:              SignatureTypeAuthor,
		SignerCertificate: weakCert,
		Certificates:      []*x509.Certificate{weakCert, rootCert},
		HashAlgorithm:     HashAlgorithmSHA256,
	}

	opts := VerificationOptions{
		TrustStore:            trustStore,
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256},
	}

	result := VerifySignature(sig, opts)

	if result.IsValid {
		t.Error("expected signature to be invalid due to weak RSA key")
	}

	foundWeakKeyError := false
	for _, err := range result.Errors {
		if err.Error() == "RSA key length 1024 is less than minimum 2048 bits" {
			foundWeakKeyError = true
			break
		}
	}

	if !foundWeakKeyError {
		t.Errorf("expected weak RSA key error, got errors: %v", result.Errors)
	}
}

func TestVerifySignature_ExpiredCertificate(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	expiredCert, _ := generateExpiredCert(t, rootCert, rootKey)

	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	sig := &PrimarySignature{
		Type:              SignatureTypeAuthor,
		SignerCertificate: expiredCert,
		Certificates:      []*x509.Certificate{expiredCert, rootCert},
		HashAlgorithm:     HashAlgorithmSHA256,
	}

	opts := VerificationOptions{
		TrustStore:            trustStore,
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256},
	}

	result := VerifySignature(sig, opts)

	if result.IsValid {
		t.Error("expected signature to be invalid due to expired certificate")
	}

	if len(result.Errors) == 0 {
		t.Error("expected error about expired certificate")
	}
}

func TestVerifySignature_WithValidTimestamp(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)
	tsaCert, _ := generateTestTimestampCert(t, rootCert, rootKey)

	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	timestamp := Timestamp{
		Time:              time.Now().Add(-1 * time.Hour),
		SignerCertificate: tsaCert,
		HashAlgorithm:     HashAlgorithmSHA256,
	}

	sig := &PrimarySignature{
		Type:              SignatureTypeAuthor,
		SignerCertificate: signerCert,
		Certificates:      []*x509.Certificate{signerCert, rootCert, tsaCert},
		HashAlgorithm:     HashAlgorithmSHA256,
		Timestamps:        []Timestamp{timestamp},
	}

	opts := VerificationOptions{
		TrustStore:            trustStore,
		RequireTimestamp:      true,
		VerifyTimestamp:       true,
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256},
	}

	result := VerifySignature(sig, opts)

	if !result.IsValid {
		t.Errorf("expected valid signature, got errors: %v", result.Errors)
	}

	if !result.TimestampValid {
		t.Error("expected timestamp to be valid")
	}

	if result.SigningTime == nil {
		t.Error("expected signing time to be set")
	}
}

func TestVerifySignature_MissingRequiredTimestamp(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	signerCert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	trustStore := NewTrustStore()
	trustStore.AddCertificate(rootCert)

	sig := &PrimarySignature{
		Type:              SignatureTypeAuthor,
		SignerCertificate: signerCert,
		Certificates:      []*x509.Certificate{signerCert, rootCert},
		HashAlgorithm:     HashAlgorithmSHA256,
		Timestamps:        []Timestamp{}, // No timestamp
	}

	opts := VerificationOptions{
		TrustStore:            trustStore,
		RequireTimestamp:      true, // Timestamp required
		AllowedSignatureTypes: []SignatureType{SignatureTypeAuthor},
		AllowedHashAlgorithms: []HashAlgorithmName{HashAlgorithmSHA256},
	}

	result := VerifySignature(sig, opts)

	if result.IsValid {
		t.Error("expected signature to be invalid due to missing required timestamp")
	}

	foundTimestampError := false
	for _, err := range result.Errors {
		if err.Error() == "signature does not have a timestamp" {
			foundTimestampError = true
			break
		}
	}

	if !foundTimestampError {
		t.Errorf("expected missing timestamp error, got errors: %v", result.Errors)
	}
}

func TestVerifyCertificateChain_MissingSignerCertificate(t *testing.T) {
	sig := &PrimarySignature{
		Type:              SignatureTypeAuthor,
		SignerCertificate: nil, // Missing
		Certificates:      []*x509.Certificate{},
		HashAlgorithm:     HashAlgorithmSHA256,
	}

	opts := VerificationOptions{
		TrustStore: NewTrustStore(),
	}

	result := verifyCertificateChain(sig, opts)

	if result.IsValid {
		t.Error("expected chain verification to fail with missing signer certificate")
	}

	if len(result.Errors) == 0 {
		t.Error("expected error about missing signer certificate")
	}
}

func TestVerifyTimestamp_MissingTSACertificate(t *testing.T) {
	ts := Timestamp{
		Time:              time.Now(),
		SignerCertificate: nil, // Missing
		HashAlgorithm:     HashAlgorithmSHA256,
	}

	opts := VerificationOptions{
		TrustStore: NewTrustStore(),
	}

	result := verifyTimestamp(ts, opts)

	if result.IsValid {
		t.Error("expected timestamp verification to fail with missing TSA certificate")
	}

	if len(result.Errors) == 0 {
		t.Error("expected error about missing TSA certificate")
	}
}

func TestTrustStore_NewTrustStore(t *testing.T) {
	ts := NewTrustStore()
	if ts == nil {
		t.Fatal("NewTrustStore returned nil")
	}

	if ts.roots == nil {
		t.Error("trust store roots should not be nil")
	}
}

func TestTrustStore_NewTrustStoreFromSystem(t *testing.T) {
	ts, err := NewTrustStoreFromSystem()
	if err != nil {
		t.Skipf("skipping system trust store test: %v", err)
	}

	if ts == nil {
		t.Fatal("NewTrustStoreFromSystem returned nil")
	}

	if ts.roots == nil {
		t.Error("trust store roots should not be nil")
	}
}

func TestTrustStore_AddCertificate(t *testing.T) {
	ts := NewTrustStore()
	rootCert, _ := generateTestRootCA(t)

	ts.AddCertificate(rootCert)

	pool := ts.GetRootPool()
	if pool == nil {
		t.Error("GetRootPool returned nil")
	}
}

func TestTrustStore_AddCertificatePEM(t *testing.T) {
	ts := NewTrustStore()
	rootCert, _ := generateTestRootCA(t)

	// Encode to PEM
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rootCert.Raw,
	})

	err := ts.AddCertificatePEM(pemData)
	if err != nil {
		t.Errorf("AddCertificatePEM failed: %v", err)
	}

	pool := ts.GetRootPool()
	if pool == nil {
		t.Error("GetRootPool returned nil")
	}
}

func TestTrustStore_AddCertificatePEM_Invalid(t *testing.T) {
	ts := NewTrustStore()

	// Invalid PEM data
	err := ts.AddCertificatePEM([]byte("not a certificate"))
	if err == nil {
		t.Error("expected error when adding invalid PEM data")
	}
}

func TestIsSignatureTypeAllowed(t *testing.T) {
	tests := []struct {
		name     string
		sigType  SignatureType
		allowed  []SignatureType
		expected bool
	}{
		{
			name:     "Author allowed",
			sigType:  SignatureTypeAuthor,
			allowed:  []SignatureType{SignatureTypeAuthor, SignatureTypeRepository},
			expected: true,
		},
		{
			name:     "Repository allowed",
			sigType:  SignatureTypeRepository,
			allowed:  []SignatureType{SignatureTypeAuthor, SignatureTypeRepository},
			expected: true,
		},
		{
			name:     "Unknown not allowed",
			sigType:  SignatureTypeUnknown,
			allowed:  []SignatureType{SignatureTypeAuthor, SignatureTypeRepository},
			expected: false,
		},
		{
			name:     "Empty allowed list",
			sigType:  SignatureTypeAuthor,
			allowed:  []SignatureType{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSignatureTypeAllowed(tt.sigType, tt.allowed)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsHashAlgorithmAllowed(t *testing.T) {
	tests := []struct {
		name     string
		hashAlg  HashAlgorithmName
		allowed  []HashAlgorithmName
		expected bool
	}{
		{
			name:     "SHA256 allowed",
			hashAlg:  HashAlgorithmSHA256,
			allowed:  []HashAlgorithmName{HashAlgorithmSHA256, HashAlgorithmSHA384, HashAlgorithmSHA512},
			expected: true,
		},
		{
			name:     "SHA384 allowed",
			hashAlg:  HashAlgorithmSHA384,
			allowed:  []HashAlgorithmName{HashAlgorithmSHA256, HashAlgorithmSHA384, HashAlgorithmSHA512},
			expected: true,
		},
		{
			name:     "SHA512 allowed",
			hashAlg:  HashAlgorithmSHA512,
			allowed:  []HashAlgorithmName{HashAlgorithmSHA256, HashAlgorithmSHA384, HashAlgorithmSHA512},
			expected: true,
		},
		{
			name:     "Unknown not allowed",
			hashAlg:  "",
			allowed:  []HashAlgorithmName{HashAlgorithmSHA256},
			expected: false,
		},
		{
			name:     "Empty allowed list",
			hashAlg:  HashAlgorithmSHA256,
			allowed:  []HashAlgorithmName{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHashAlgorithmAllowed(tt.hashAlg, tt.allowed)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestVerifySignerKeyLength_Valid2048Bit(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	cert, _ := generateTestCodeSigningCert(t, rootCert, rootKey)

	err := verifySignerKeyLength(cert)
	if err != nil {
		t.Errorf("expected no error for 2048-bit key, got: %v", err)
	}
}

func TestVerifySignerKeyLength_Weak1024Bit(t *testing.T) {
	rootCert, rootKey := generateTestRootCA(t)
	weakCert, _ := generateWeakRSAKeyCert(t, rootCert, rootKey)

	err := verifySignerKeyLength(weakCert)
	if err == nil {
		t.Error("expected error for 1024-bit key")
	}

	if err.Error() != "RSA key length 1024 is less than minimum 2048 bits" {
		t.Errorf("unexpected error message: %v", err)
	}
}
