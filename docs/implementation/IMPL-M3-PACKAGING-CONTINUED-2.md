# Milestone 3: Package Operations - Continued 2 (Chunks 8-10)

**Status**: Not Started
**Chunks**: 8-10 (Package Signatures)
**Estimated Time**: 8 hours

---

## M3.8: Package Signature Reader

**Estimated Time**: 2.5 hours
**Dependencies**: M3.1

### Overview

Implement reading and parsing of PKCS#7 package signatures from signed .nupkg files, including signature metadata extraction, certificate chain access, and timestamp information.

### Files to Create/Modify

- `packaging/signatures/reader.go` - Signature reader implementation
- `packaging/signatures/pkcs7.go` - PKCS#7 parsing helpers
- `packaging/signatures/reader_test.go` - Signature reader tests
- `packaging/reader.go` - Add signature reading methods

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging.Signing/PackageArchiveReader.cs` GetPrimarySignatureAsync 
- `NuGet.Packaging.Signing/SignedPackageArchiveUtility.cs` OpenPackageSignatureFileStream 
- `NuGet.Packaging.Signing/PrimarySignature.cs` - Signature structure

**Signature Reading** (SignedPackageArchiveUtility.cs:79-90):
```csharp
public static Stream OpenPackageSignatureFileStream(BinaryReader reader) {
    var metadata = SignedPackageArchiveIOUtility.ReadSignedArchiveMetadata(reader);
    var signatureCentralDirectoryHeader = metadata.GetPackageSignatureFileCentralDirectoryHeaderMetadata();

    return GetPackageSignatureFile(reader, signatureCentralDirectoryHeader);
}
```

### Implementation Details

**1. Signature Structure**:

```go
// packaging/signatures/signature.go

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

    // Parsed PKCS#7 structure
    SignedData *PKCS7SignedData

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
    HashAlgorithmSHA256 HashAlgorithmName = "SHA256"
    HashAlgorithmSHA384 HashAlgorithmName = "SHA384"
    HashAlgorithmSHA512 HashAlgorithmName = "SHA512"
)

// PKCS7SignedData represents parsed PKCS#7 SignedData structure
type PKCS7SignedData struct {
    Version          int
    DigestAlgorithms []asn1.ObjectIdentifier
    ContentInfo      PKCS7ContentInfo
    Certificates     []byte // Raw certificate data
    SignerInfos      []PKCS7SignerInfo
}

// PKCS7ContentInfo represents the signed content
type PKCS7ContentInfo struct {
    ContentType asn1.ObjectIdentifier
    Content     []byte
}

// PKCS7SignerInfo represents signer information
type PKCS7SignerInfo struct {
    Version                   int
    IssuerAndSerialNumber     PKCS7IssuerAndSerialNumber
    DigestAlgorithm           asn1.ObjectIdentifier
    AuthenticatedAttributes   []PKCS7Attribute `asn1:"optional,tag:0"`
    DigestEncryptionAlgorithm asn1.ObjectIdentifier
    EncryptedDigest           []byte
    UnauthenticatedAttributes []PKCS7Attribute `asn1:"optional,tag:1"`
}

// PKCS7IssuerAndSerialNumber identifies the signer's certificate
type PKCS7IssuerAndSerialNumber struct {
    Issuer       asn1.RawValue
    SerialNumber *big.Int
}

// PKCS7Attribute represents a PKCS#7 attribute
type PKCS7Attribute struct {
    Type   asn1.ObjectIdentifier
    Values []asn1.RawValue `asn1:"set"`
}
```

**2. Signature Reader**:

```go
// packaging/signatures/reader.go

package signatures

import (
    "crypto/x509"
    "encoding/asn1"
    "fmt"
)

// ReadSignature reads and parses a PKCS#7 signature
func ReadSignature(signatureData []byte) (*PrimarySignature, error) {
    if len(signatureData) == 0 {
        return nil, fmt.Errorf("signature data is empty")
    }

    sig := &PrimarySignature{
        RawData: signatureData,
    }

    // Parse PKCS#7 structure
    signedData, err := parsePKCS7SignedData(signatureData)
    if err != nil {
        return nil, fmt.Errorf("parse PKCS#7: %w", err)
    }
    sig.SignedData = signedData

    // Extract certificates
    certs, err := extractCertificates(signedData.Certificates)
    if err != nil {
        return nil, fmt.Errorf("extract certificates: %w", err)
    }
    sig.Certificates = certs

    // Find signer certificate
    if len(signedData.SignerInfos) > 0 {
        signerCert, err := findSignerCertificate(signedData.SignerInfos[0], certs)
        if err != nil {
            return nil, fmt.Errorf("find signer certificate: %w", err)
        }
        sig.SignerCertificate = signerCert
    }

    // Determine signature type from authenticated attributes
    sig.Type = determineSignatureType(signedData)

    // Extract hash algorithm
    sig.HashAlgorithm = extractHashAlgorithm(signedData)

    // Extract timestamps (RFC 3161)
    timestamps, err := extractTimestamps(signedData)
    if err != nil {
        // Timestamps are optional, don't fail
        timestamps = []Timestamp{}
    }
    sig.Timestamps = timestamps

    return sig, nil
}

func parsePKCS7SignedData(data []byte) (*PKCS7SignedData, error) {
    var contentInfo struct {
        ContentType asn1.ObjectIdentifier
        Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
    }

    if _, err := asn1.Unmarshal(data, &contentInfo); err != nil {
        return nil, fmt.Errorf("unmarshal content info: %w", err)
    }

    // Verify this is SignedData (OID 1.2.840.113549.1.7.2)
    signedDataOID := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
    if !contentInfo.ContentType.Equal(signedDataOID) {
        return nil, fmt.Errorf("not a PKCS#7 SignedData structure")
    }

    var signedData PKCS7SignedData
    if _, err := asn1.Unmarshal(contentInfo.Content.Bytes, &signedData); err != nil {
        return nil, fmt.Errorf("unmarshal signed data: %w", err)
    }

    return &signedData, nil
}

func extractCertificates(certData []byte) ([]*x509.Certificate, error) {
    if len(certData) == 0 {
        return []*x509.Certificate{}, nil
    }

    // Certificates are in a SET
    var certSet []asn1.RawValue
    if _, err := asn1.Unmarshal(certData, &certSet); err != nil {
        // Try parsing as a single certificate
        cert, err := x509.ParseCertificate(certData)
        if err != nil {
            return nil, fmt.Errorf("parse certificates: %w", err)
        }
        return []*x509.Certificate{cert}, nil
    }

    var certs []*x509.Certificate
    for _, certBytes := range certSet {
        cert, err := x509.ParseCertificate(certBytes.FullBytes)
        if err != nil {
            return nil, fmt.Errorf("parse certificate: %w", err)
        }
        certs = append(certs, cert)
    }

    return certs, nil
}

func findSignerCertificate(signerInfo PKCS7SignerInfo, certs []*x509.Certificate) (*x509.Certificate, error) {
    // Match by issuer and serial number
    for _, cert := range certs {
        if cert.SerialNumber.Cmp(signerInfo.IssuerAndSerialNumber.SerialNumber) == 0 {
            // Note: In production, should also verify issuer matches
            return cert, nil
        }
    }

    return nil, fmt.Errorf("signer certificate not found in certificate chain")
}

func determineSignatureType(signedData *PKCS7SignedData) SignatureType {
    if len(signedData.SignerInfos) == 0 {
        return SignatureTypeUnknown
    }

    // Check authenticated attributes for nuget-specific signature type
    // NuGet uses OID 1.3.6.1.4.1.311.2.4.1 for commitment-type-indication
    // Author: "1.3.6.1.4.1.311.2.4.1.1"
    // Repository: "1.3.6.1.4.1.311.2.4.1.2"

    for _, attr := range signedData.SignerInfos[0].AuthenticatedAttributes {
        // Commitment type indication OID
        commitmentTypeOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 2, 4, 1}
        if attr.Type.Equal(commitmentTypeOID) {
            // Parse commitment type from attribute value
            if len(attr.Values) > 0 {
                var commitmentType asn1.ObjectIdentifier
                if _, err := asn1.Unmarshal(attr.Values[0].FullBytes, &commitmentType); err == nil {
                    authorOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 2, 4, 1, 1}
                    repoOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 2, 4, 1, 2}

                    if commitmentType.Equal(authorOID) {
                        return SignatureTypeAuthor
                    } else if commitmentType.Equal(repoOID) {
                        return SignatureTypeRepository
                    }
                }
            }
        }
    }

    return SignatureTypeUnknown
}

func extractHashAlgorithm(signedData *PKCS7SignedData) HashAlgorithmName {
    if len(signedData.DigestAlgorithms) == 0 {
        return ""
    }

    // Map OIDs to hash algorithm names
    // SHA256: 2.16.840.1.101.3.4.2.1
    // SHA384: 2.16.840.1.101.3.4.2.2
    // SHA512: 2.16.840.1.101.3.4.2.3

    oid := signedData.DigestAlgorithms[0]

    sha256OID := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
    sha384OID := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
    sha512OID := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}

    switch {
    case oid.Equal(sha256OID):
        return HashAlgorithmSHA256
    case oid.Equal(sha384OID):
        return HashAlgorithmSHA384
    case oid.Equal(sha512OID):
        return HashAlgorithmSHA512
    default:
        return ""
    }
}

func extractTimestamps(signedData *PKCS7SignedData) ([]Timestamp, error) {
    // Timestamps are in unauthenticated attributes
    // RFC 3161 Timestamp OID: 1.2.840.113549.1.9.16.2.14

    var timestamps []Timestamp

    if len(signedData.SignerInfos) == 0 {
        return timestamps, nil
    }

    timestampOID := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 14}

    for _, attr := range signedData.SignerInfos[0].UnauthenticatedAttributes {
        if attr.Type.Equal(timestampOID) {
            // Parse timestamp token (another PKCS#7 SignedData)
            for _, value := range attr.Values {
                ts, err := parseTimestampToken(value.FullBytes)
                if err != nil {
                    continue // Skip invalid timestamps
                }
                timestamps = append(timestamps, ts)
            }
        }
    }

    return timestamps, nil
}

func parseTimestampToken(data []byte) (Timestamp, error) {
    // Simplified timestamp parsing
    // Full implementation would parse TSTInfo structure
    var ts Timestamp

    // Parse as PKCS#7 SignedData
    tsSignedData, err := parsePKCS7SignedData(data)
    if err != nil {
        return ts, err
    }

    // Extract timestamp from content
    // TSTInfo is in the encapsulated content
    var tstInfo struct {
        Version        int
        Policy         asn1.ObjectIdentifier
        MessageImprint struct {
            HashAlgorithm asn1.ObjectIdentifier
            HashedMessage []byte
        }
        SerialNumber *big.Int
        GenTime      time.Time
        Accuracy     asn1.RawValue `asn1:"optional"`
    }

    if _, err := asn1.Unmarshal(tsSignedData.ContentInfo.Content, &tstInfo); err != nil {
        return ts, fmt.Errorf("parse TSTInfo: %w", err)
    }

    ts.Time = tstInfo.GenTime

    // Extract TSA certificate
    certs, err := extractCertificates(tsSignedData.Certificates)
    if err != nil {
        return ts, err
    }

    if len(certs) > 0 {
        ts.SignerCertificate = certs[0]
    }

    // Map hash algorithm
    ts.HashAlgorithm = mapOIDToHashAlgorithm(tstInfo.MessageImprint.HashAlgorithm)

    return ts, nil
}

func mapOIDToHashAlgorithm(oid asn1.ObjectIdentifier) HashAlgorithmName {
    sha256OID := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
    sha384OID := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
    sha512OID := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}

    switch {
    case oid.Equal(sha256OID):
        return HashAlgorithmSHA256
    case oid.Equal(sha384OID):
        return HashAlgorithmSHA384
    case oid.Equal(sha512OID):
        return HashAlgorithmSHA512
    default:
        return ""
    }
}
```

**3. Add to PackageReader**:

```go
// packaging/reader.go additions

import (
    "github.com/willibrandon/gonuget/packaging/signatures"
)

// GetPrimarySignature returns the primary signature if package is signed
func (r *PackageReader) GetPrimarySignature() (*signatures.PrimarySignature, error) {
    if !r.IsSigned() {
        return nil, ErrPackageNotSigned
    }

    // Get signature file
    sigFile, err := r.GetSignatureFile()
    if err != nil {
        return nil, err
    }

    // Open and read signature data
    reader, err := sigFile.Open()
    if err != nil {
        return nil, fmt.Errorf("open signature file: %w", err)
    }
    defer reader.Close()

    sigData, err := io.ReadAll(reader)
    if err != nil {
        return nil, fmt.Errorf("read signature data: %w", err)
    }

    // Parse signature
    return signatures.ReadSignature(sigData)
}

// IsRepositorySigned checks if package has a repository signature
func (r *PackageReader) IsRepositorySigned() (bool, error) {
    sig, err := r.GetPrimarySignature()
    if err != nil {
        if err == ErrPackageNotSigned {
            return false, nil
        }
        return false, err
    }

    return sig.Type == signatures.SignatureTypeRepository, nil
}

// IsAuthorSigned checks if package has an author signature
func (r *PackageReader) IsAuthorSigned() (bool, error) {
    sig, err := r.GetPrimarySignature()
    if err != nil {
        if err == ErrPackageNotSigned {
            return false, nil
        }
        return false, err
    }

    return sig.Type == signatures.SignatureTypeAuthor, nil
}
```

### Verification Steps

```bash
# 1. Run signature reader tests
go test ./packaging/signatures -v -run TestSignatureReader

# 2. Test PKCS#7 parsing
go test ./packaging/signatures -v -run TestParsePKCS7

# 3. Test with real signed packages
go test ./packaging/signatures -v -run TestReadRealSignature

# 4. Test certificate extraction
go test ./packaging/signatures -v -run TestExtractCertificates

# 5. Check test coverage
go test ./packaging/signatures -cover
```

### Acceptance Criteria

- [ ] Read .signature.p7s file from package
- [ ] Parse PKCS#7 SignedData structure
- [ ] Extract certificate chain
- [ ] Identify signer certificate
- [ ] Determine signature type (Author/Repository)
- [ ] Extract hash algorithm
- [ ] Parse RFC 3161 timestamps
- [ ] Extract timestamp authority certificates
- [ ] Handle packages without signatures gracefully
- [ ] Expose signature via PackageReader
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement package signature reader

Add PKCS#7 signature reading:
- Parse .signature.p7s files
- Extract certificate chains
- Identify signature type (Author/Repository)
- Parse RFC 3161 timestamps
- Hash algorithm detection (SHA256/384/512)
- Integration with PackageReader

Reference: NuGet.Packaging.Signing/PrimarySignature.cs
```

---

## M3.9: Package Signature Verification

**Estimated Time**: 3 hours
**Dependencies**: M3.8

### Overview

Implement package signature verification including certificate chain validation, trust store verification, timestamp verification, and package integrity checking.

### Files to Create/Modify

- `packaging/signatures/verifier.go` - Signature verification implementation
- `packaging/signatures/trust.go` - Trust store management
- `packaging/signatures/verifier_test.go` - Verification tests

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging.Signing/SignedPackageArchiveUtility.cs` VerifySignedPackageIntegrity 
- `NuGet.Packaging.Signing/SignatureVerificationProvider.cs`
- NuGet uses Windows certificate store or custom trust policies

**Integrity Verification** (SignedPackageArchiveUtility.cs:447-533):
```csharp
internal static bool VerifySignedPackageIntegrity(BinaryReader reader,
                                                   HashAlgorithm hashAlgorithm,
                                                   byte[] expectedHash) {
    if (!IsSigned(reader)) {
        throw new SignatureException("Package is not signed");
    }

    var metadata = ReadSignedArchiveMetadata(reader);
    var signatureCDH = metadata.GetPackageSignatureFileCentralDirectoryHeaderMetadata();
    var centralDirectoryRecords = RemoveSignatureAndOrderByOffset(metadata);

    // Hash everything except the signature file
    // ... (detailed hashing logic)

    return CompareHash(expectedHash, hashAlgorithm.Hash);
}
```

### Implementation Details

**1. Verification Options**:

```go
// packaging/signatures/verifier.go

package signatures

import (
    "crypto"
    "crypto/x509"
    "fmt"
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
```

**2. Signature Verifier**:

```go
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
    for _, a := range allowed {
        if a == sigType {
            return true
        }
    }
    return false
}

func isHashAlgorithmAllowed(hashAlg HashAlgorithmName, allowed []HashAlgorithmName) bool {
    for _, a := range allowed {
        if a == hashAlg {
            return true
        }
    }
    return false
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
```

**3. Certificate Chain Verification**:

```go
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
```

**4. Timestamp Verification**:

```go
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
```

**5. Trust Store**:

```go
// packaging/signatures/trust.go

package signatures

import (
    "crypto/x509"
)

// TrustStore manages trusted root certificates
type TrustStore struct {
    roots *x509.CertPool
}

// NewTrustStore creates a new trust store
func NewTrustStore() *TrustStore {
    return &TrustStore{
        roots: x509.NewCertPool(),
    }
}

// NewTrustStoreFromSystem creates a trust store from system roots
func NewTrustStoreFromSystem() (*TrustStore, error) {
    roots, err := x509.SystemCertPool()
    if err != nil {
        return nil, err
    }

    return &TrustStore{
        roots: roots,
    }, nil
}

// AddCertificate adds a trusted root certificate
func (ts *TrustStore) AddCertificate(cert *x509.Certificate) {
    ts.roots.AddCert(cert)
}

// AddCertificatePEM adds a trusted root from PEM data
func (ts *TrustStore) AddCertificatePEM(pemData []byte) error {
    if !ts.roots.AppendCertsFromPEM(pemData) {
        return fmt.Errorf("failed to parse PEM certificate")
    }
    return nil
}

// GetRootPool returns the certificate pool
func (ts *TrustStore) GetRootPool() *x509.CertPool {
    return ts.roots
}
```

### Verification Steps

```bash
# 1. Run verification tests
go test ./packaging/signatures -v -run TestVerifySignature

# 2. Test certificate chain verification
go test ./packaging/signatures -v -run TestVerifyCertificateChain

# 3. Test timestamp verification
go test ./packaging/signatures -v -run TestVerifyTimestamp

# 4. Test trust store
go test ./packaging/signatures -v -run TestTrustStore

# 5. Test with real signed packages
go test ./packaging/signatures -v -run TestVerifyRealSignature

# 6. Check test coverage
go test ./packaging/signatures -cover
```

### Acceptance Criteria

- [ ] Verify certificate chain to trusted root
- [ ] Verify signer certificate validity
- [ ] Check RSA minimum key length (2048 bits)
- [ ] Verify allowed signature types
- [ ] Verify allowed hash algorithms
- [ ] Verify RFC 3161 timestamps
- [ ] Verify timestamp authority certificates
- [ ] Support system trust store
- [ ] Support custom trust store
- [ ] Return detailed verification results
- [ ] Handle untrusted roots with option
- [ ] 90%+ test coverage

### Commit Message

```
feat(packaging): implement signature verification

Add signature verification with:
- Certificate chain validation
- Trust store management (system and custom)
- Timestamp verification (RFC 3161)
- RSA key length enforcement (2048+ bits)
- Hash algorithm validation (SHA256/384/512)
- Detailed verification results

Reference: SignedPackageArchiveUtility.cs
Reference: SigningSpecificationsV1.cs
```

---

## M3.10: Package Signature Creation

**Estimated Time**: 2.5 hours
**Dependencies**: M3.4, M3.5, M3.6

### Overview

Implement package signing with PKCS#7 signature generation, timestamp request support, and signature embedding into .nupkg files.

### Files to Create/Modify

- `packaging/signatures/signer.go` - Package signing implementation
- `packaging/signatures/timestamp.go` - RFC 3161 timestamp client
- `packaging/signatures/signer_test.go` - Signing tests
- `packaging/builder.go` - Add Sign method

### Reference Implementation

**NuGet.Client Reference**:
- `NuGet.Packaging.Signing/SignedPackageArchive.cs` AddSignatureAsync
- `NuGet.Packaging.Signing/Rfc3161TimestampProvider.cs`
- Uses .NET PKCS#7 SignedCms class

**Note**: Full PKCS#7 signature creation is complex. This chunk provides a simplified interface that wraps Go's crypto libraries. Production use should leverage existing libraries like `github.com/digitorus/pkcs7`.

### Implementation Details

**1. Signing Options**:

```go
// packaging/signatures/signer.go

package signatures

import (
    "crypto"
    "crypto/rsa"
    "crypto/x509"
    "fmt"
    "time"
)

// SigningOptions configures package signing
type SigningOptions struct {
    // Certificate is the signing certificate
    Certificate *x509.Certificate

    // PrivateKey is the private key for signing
    PrivateKey crypto.PrivateKey

    // CertificateChain contains intermediate certificates
    CertificateChain []*x509.Certificate

    // SignatureType is the type of signature to create
    SignatureType SignatureType

    // HashAlgorithm is the hash algorithm to use
    HashAlgorithm HashAlgorithmName

    // TimestampURL is the RFC 3161 timestamp authority URL (optional)
    TimestampURL string

    // TimestampTimeout is the timeout for timestamp requests
    TimestampTimeout time.Duration
}

// DefaultSigningOptions returns default signing options
func DefaultSigningOptions(cert *x509.Certificate, key crypto.PrivateKey) SigningOptions {
    return SigningOptions{
        Certificate:      cert,
        PrivateKey:       key,
        SignatureType:    SignatureTypeAuthor,
        HashAlgorithm:    HashAlgorithmSHA256,
        TimestampTimeout: 30 * time.Second,
    }
}

// Validate validates signing options
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
    if !contains(allowedAlgos, opts.HashAlgorithm) {
        return fmt.Errorf("hash algorithm %s is not allowed", opts.HashAlgorithm)
    }

    return nil
}

func verifyKeyMatchesCertificate(cert *x509.Certificate, key crypto.PrivateKey) error {
    // Simplified check - in production use more robust verification
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

func contains(slice []HashAlgorithmName, item HashAlgorithmName) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

**2. Package Signing**:

```go
// SignPackageData creates a PKCS#7 signature for package content
// Note: This is a simplified implementation. Production code should use
// a robust PKCS#7 library like github.com/digitorus/pkcs7
func SignPackageData(contentHash []byte, opts SigningOptions) ([]byte, error) {
    if err := opts.Validate(); err != nil {
        return nil, fmt.Errorf("invalid signing options: %w", err)
    }

    // In production, use a PKCS#7 library:
    // 1. Create SignedData structure
    // 2. Add content hash
    // 3. Add certificates
    // 4. Add authenticated attributes (signature type, signing time)
    // 5. Sign with private key
    // 6. Optionally add timestamp

    return nil, fmt.Errorf("PKCS#7 signing requires external library (github.com/digitorus/pkcs7)")
}

// Note: Full implementation outline (requires pkcs7 library):
/*
import "github.com/digitorus/pkcs7"

func SignPackageData(contentHash []byte, opts SigningOptions) ([]byte, error) {
    // Create signed data
    signedData, err := pkcs7.NewSignedData(contentHash)
    if err != nil {
        return nil, err
    }

    // Add signer
    if err := signedData.AddSigner(opts.Certificate, opts.PrivateKey, pkcs7.SignerInfoConfig{
        DigestAlgorithm: mapHashAlgorithm(opts.HashAlgorithm),
    }); err != nil {
        return nil, err
    }

    // Add certificate chain
    for _, cert := range opts.CertificateChain {
        signedData.AddCertificate(cert)
    }

    // Add NuGet-specific authenticated attributes
    // - Signature type (Author/Repository)
    // - Signing time
    // - Package hash

    // Sign
    signature, err := signedData.Finish()
    if err != nil {
        return nil, err
    }

    // Add timestamp if requested
    if opts.TimestampURL != "" {
        signature, err = addTimestamp(signature, opts.TimestampURL, opts.TimestampTimeout)
        if err != nil {
            return nil, fmt.Errorf("add timestamp: %w", err)
        }
    }

    return signature, nil
}
*/
```

**3. RFC 3161 Timestamp Client**:

```go
// packaging/signatures/timestamp.go

package signatures

import (
    "bytes"
    "crypto/sha256"
    "encoding/asn1"
    "fmt"
    "io"
    "net/http"
    "time"
)

// TimestampClient requests RFC 3161 timestamps
type TimestampClient struct {
    URL     string
    Timeout time.Duration
    client  *http.Client
}

// NewTimestampClient creates a new timestamp client
func NewTimestampClient(url string, timeout time.Duration) *TimestampClient {
    return &TimestampClient{
        URL:     url,
        Timeout: timeout,
        client: &http.Client{
            Timeout: timeout,
        },
    }
}

// RequestTimestamp requests a timestamp for a message hash
// Reference: RFC 3161 Time-Stamp Protocol
func (tc *TimestampClient) RequestTimestamp(messageHash []byte, hashAlgorithm HashAlgorithmName) ([]byte, error) {
    // Create TimeStampReq
    tsReq, err := createTimestampRequest(messageHash, hashAlgorithm)
    if err != nil {
        return nil, fmt.Errorf("create timestamp request: %w", err)
    }

    // Send HTTP POST request
    resp, err := tc.client.Post(tc.URL, "application/timestamp-query", bytes.NewReader(tsReq))
    if err != nil {
        return nil, fmt.Errorf("send timestamp request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("timestamp server returned status %d", resp.StatusCode)
    }

    // Read response
    tsResp, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read timestamp response: %w", err)
    }

    // Verify and extract timestamp token
    token, err := extractTimestampToken(tsResp)
    if err != nil {
        return nil, fmt.Errorf("extract timestamp token: %w", err)
    }

    return token, nil
}

func createTimestampRequest(messageHash []byte, hashAlgorithm HashAlgorithmName) ([]byte, error) {
    // TimeStampReq ::= SEQUENCE {
    //   version INTEGER { v1(1) },
    //   messageImprint MessageImprint,
    //   reqPolicy TSAPolicyId OPTIONAL,
    //   nonce INTEGER OPTIONAL,
    //   certReq BOOLEAN DEFAULT FALSE,
    //   extensions [0] IMPLICIT Extensions OPTIONAL }

    hashAlgOID := mapHashAlgorithmToOID(hashAlgorithm)

    messageImprint := struct {
        HashAlgorithm asn1.ObjectIdentifier
        HashedMessage []byte
    }{
        HashAlgorithm: hashAlgOID,
        HashedMessage: messageHash,
    }

    tsReq := struct {
        Version        int
        MessageImprint interface{}
        CertReq        bool
    }{
        Version:        1,
        MessageImprint: messageImprint,
        CertReq:        true,
    }

    return asn1.Marshal(tsReq)
}

func extractTimestampToken(responseData []byte) ([]byte, error) {
    // TimeStampResp ::= SEQUENCE {
    //   status PKIStatusInfo,
    //   timeStampToken TimeStampToken OPTIONAL }

    var tsResp struct {
        Status struct {
            Status int
        }
        TimeStampToken asn1.RawValue `asn1:"optional"`
    }

    if _, err := asn1.Unmarshal(responseData, &tsResp); err != nil {
        return nil, fmt.Errorf("unmarshal timestamp response: %w", err)
    }

    // Status 0 = granted, 1 = granted with modifications
    if tsResp.Status.Status != 0 && tsResp.Status.Status != 1 {
        return nil, fmt.Errorf("timestamp request failed with status %d", tsResp.Status.Status)
    }

    return tsResp.TimeStampToken.FullBytes, nil
}

func mapHashAlgorithmToOID(hashAlg HashAlgorithmName) asn1.ObjectIdentifier {
    switch hashAlg {
    case HashAlgorithmSHA256:
        return asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
    case HashAlgorithmSHA384:
        return asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
    case HashAlgorithmSHA512:
        return asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}
    default:
        return asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1} // Default SHA256
    }
}
```

**4. Integration with PackageBuilder** (Placeholder):

```go
// packaging/builder.go

// Sign signs the package (placeholder - requires PKCS#7 library)
func (b *PackageBuilder) Sign(opts signatures.SigningOptions) error {
    return fmt.Errorf("package signing requires github.com/digitorus/pkcs7 library")
}

// Note: Full implementation would:
// 1. Save package to temp location
// 2. Calculate package hash (excluding signature)
// 3. Create PKCS#7 signature
// 4. Add timestamp if requested
// 5. Embed signature into .signature.p7s
// 6. Update OPC files
```

### Verification Steps

```bash
# 1. Run signing tests (with mock PKCS#7)
go test ./packaging/signatures -v -run TestSigning

# 2. Test timestamp client
go test ./packaging/signatures -v -run TestTimestampClient

# 3. Test signing options validation
go test ./packaging/signatures -v -run TestSigningOptions

# 4. Check test coverage
go test ./packaging/signatures -cover
```

### Acceptance Criteria

- [ ] Define signing options structure
- [ ] Validate certificate and key
- [ ] Validate RSA key length (2048+ bits)
- [ ] RFC 3161 timestamp client implementation
- [ ] Timestamp request/response parsing
- [ ] Integration point with PackageBuilder
- [ ] Documentation for PKCS#7 library requirement
- [ ] 90%+ test coverage (with mocks)

### Commit Message

```
feat(packaging): add package signing infrastructure

Add package signing framework:
- Signing options with validation
- Certificate and key verification
- RFC 3161 timestamp client
- Timestamp request/response handling
- Integration points for PKCS#7 signing

Note: Full PKCS#7 signing requires github.com/digitorus/pkcs7

Reference: SignedPackageArchive.cs
Reference: Rfc3161TimestampProvider.cs
```

---

## Summary - Chunks 8-10 Complete

**Total Time for This File**: 8 hours
**Files Created**: 9
**Lines of Code**: ~1,300

**Next File**: IMPL-M3-PACKAGING-CONTINUED-3.md (Chunks 11-14: Asset Selection & Extraction)

**Dependencies for Next Chunks**:
- M3.11 requires M1 (frameworks package)
- M3.12 requires M3.11 (pattern engine)
- M3.13 requires M3.12 (framework resolution)
- M3.14 requires M3.1, M3.2, M3.3 (package reader)
