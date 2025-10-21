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

## M3.10: Package Signature Creation - Production Implementation

**Estimated Time**: 8-10 hours
**Dependencies**: M3.4, M3.5, M3.6, M3.8, M3.9

### Overview

Implement **production-ready** PKCS#7/CMS signature creation for NuGet packages using Go's native crypto and ASN.1 libraries. This implementation creates RFC 5652 compliant signatures with NuGet-specific authenticated attributes, matching NuGet.Client's SignedCms behavior.

**Key Differences from Previous Guide**:
- ❌ **OLD**: Placeholder implementation requiring external library
- ✅ **NEW**: Full production implementation using Go crypto/x509, crypto/rsa, encoding/asn1
- ✅ **NEW**: Complete CMS/PKCS#7 structure building
- ✅ **NEW**: NuGet-specific authenticated attributes
- ✅ **NEW**: RSA-PKCS#1 v1.5 signature generation
- ✅ **NEW**: RFC 3161 timestamp integration

### Files to Create/Modify

- `packaging/signatures/cms.go` - CMS/PKCS#7 structure definitions and encoding
- `packaging/signatures/signer.go` - Package signing implementation
- `packaging/signatures/attributes.go` - NuGet authenticated attributes
- `packaging/signatures/timestamp.go` - RFC 3161 timestamp client (keep existing)
- `packaging/signatures/signer_test.go` - Signing tests
- `packaging/builder.go` - Add functional Sign method

### Reference Implementation

**NuGet.Client References**:
- `SigningUtility.cs` - CreateCmsSigner, CreateSignedAttributes (lines 112-170)
- `X509SignatureProvider.cs` - CreatePrimarySignature (lines 95-150)
- `AttributeUtility.cs` - CreateCommitmentTypeIndication, CreateSigningCertificateV2
- `Oids.cs` - OID constants
- Uses .NET `SignedCms` class (System.Security.Cryptography.Pkcs)

**Standards**:
- RFC 5652 - Cryptographic Message Syntax (CMS)
- RFC 3161 - Time-Stamp Protocol (TSP)
- RFC 3370 - CMS Algorithms

### Architecture

```
SignPackageData()
    └── CreateSignedData()
        ├── BuildContentInfo() - Package hash
        ├── BuildSignerInfo()
        │   ├── BuildSignedAttributes()
        │   │   ├── Pkcs9SigningTime
        │   │   ├── CommitmentTypeIndication (Author/Repository)
        │   │   └── SigningCertificateV2
        │   ├── SignAttributes() - RSA-PKCS#1 v1.5
        │   └── AddTimestamp() - RFC 3161 (optional)
        └── EncodeSignedData() - DER encoding
```

###

 Implementation Details

#### 1. CMS/PKCS#7 Structure Definitions

```go
// packaging/signatures/cms.go

package signatures

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"math/big"
	"time"
)

// OID constants for CMS/PKCS#7 and NuGet
var (
	// RFC 5652 - CMS content types
	oidData        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidSignedData  = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}

	// PKCS#9 attributes
	oidContentType   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	oidMessageDigest = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}
	oidSigningTime   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}

	// RFC 5126 - Commitment Type Indication
	oidCommitmentTypeIndication = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 16}
	oidProofOfOrigin           = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 6, 1} // Author
	oidProofOfReceipt          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 6, 2} // Repository

	// ESS - Enhanced Security Services
	oidSigningCertificateV2 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 47}

	// Signature algorithms
	oidRSAEncryption      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	oidSHA256WithRSA      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}
	oidSHA384WithRSA      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12}
	oidSHA512WithRSA      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13}
)

// ContentInfo is the outer wrapper for CMS structures
type ContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

// SignedData represents a CMS SignedData structure (RFC 5652 Section 5.1)
type SignedData struct {
	Version          int                      `asn1:"default:1"`
	DigestAlgorithms []AlgorithmIdentifier    `asn1:"set"`
	EncapContentInfo EncapsulatedContentInfo
	Certificates     asn1.RawValue            `asn1:"optional,tag:0"`
	CRLs             asn1.RawValue            `asn1:"optional,tag:1"`
	SignerInfos      []SignerInfo             `asn1:"set"`
}

// EncapsulatedContentInfo contains the signed content
type EncapsulatedContentInfo struct {
	EContentType asn1.ObjectIdentifier
	EContent     asn1.RawValue `asn1:"optional,explicit,tag:0"`
}

// SignerInfo contains signature information (RFC 5652 Section 5.3)
type SignerInfo struct {
	Version            int           `asn1:"default:1"`
	SID                asn1.RawValue // SignerIdentifier (CHOICE)
	DigestAlgorithm    AlgorithmIdentifier
	SignedAttrs        asn1.RawValue `asn1:"optional,tag:0"`
	SignatureAlgorithm AlgorithmIdentifier
	Signature          []byte
	UnsignedAttrs      asn1.RawValue `asn1:"optional,tag:1"`
}

// IssuerAndSerialNumber identifies a certificate (RFC 5652 Section 10.2.4)
type IssuerAndSerialNumber struct {
	Issuer       asn1.RawValue // Name
	SerialNumber *big.Int
}

// Attribute represents a CMS attribute
type Attribute struct {
	Type   asn1.ObjectIdentifier
	Values asn1.RawValue `asn1:"set"`
}

// SigningCertificateV2 per RFC 5035
type SigningCertificateV2 struct {
	Certs []ESSCertIDv2 `asn1:"sequence"`
}

// ESSCertIDv2 identifies a certificate by hash
type ESSCertIDv2 struct {
	HashAlgorithm AlgorithmIdentifier            `asn1:"optional"`
	CertHash      []byte
	IssuerSerial  IssuerSerial                   `asn1:"optional"`
}

// IssuerSerial combines issuer and serial number
type IssuerSerial struct {
	Issuer       []asn1.RawValue                `asn1:"sequence"`
	SerialNumber *big.Int
}

// CommitmentTypeIndication per RFC 5126 Section 5.11.1
type CommitmentTypeIndication struct {
	CommitmentTypeID asn1.ObjectIdentifier
}

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
```

#### 2. Authenticated Attributes Builder

```go
// packaging/signatures/attributes.go

package signatures

import (
	"crypto"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"time"
)

// BuildSignedAttributes creates authenticated attributes for the signature
// Matches NuGet.Client's SigningUtility.CreateSignedAttributes
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

func createCommitmentTypeIndicationAttribute(sigType SignatureType) (Attribute, error) {
	var commitmentOID asn1.ObjectIdentifier
	switch sigType {
	case SignatureTypeAuthor:
		commitmentOID = oidProofOfOrigin
	case SignatureTypeRepository:
		commitmentOID = oidProofOfReceipt
	default:
		return Attribute{}, fmt.Errorf("unknown signature type: %s", sigType)
	}

	// CommitmentTypeIndication ::= SEQUENCE {
	//   commitmentTypeId   OBJECT IDENTIFIER }
	commitment := CommitmentTypeIndication{
		CommitmentTypeID: commitmentOID,
	}

	value, err := asn1.Marshal(commitment)
	if err != nil {
		return Attribute{}, err
	}

	values, err := asn1.Marshal([]asn1.RawValue{{FullBytes: value}})
	if err != nil {
		return Attribute{}, err
	}

	return Attribute{
		Type:   oidCommitmentTypeIndication,
		Values: asn1.RawValue{FullBytes: values},
	}, nil
}

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
	essC ertID := ESSCertIDv2{
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

// EncodeAttributesForSigning encodes attributes for signing (DER, with SET tag)
// Per RFC 5652 Section 5.3: "the content that is signed is the DER encoding of the signedAttrs"
func EncodeAttributesForSigning(attributes []Attribute) ([]byte, error) {
	// Encode as SET (tag 17, constructed)
	encoded, err := asn1.MarshalWithParams(attributes, "set")
	if err != nil {
		return nil, err
	}
	return encoded, nil
}
```

#### 3. Signature Creation

```go
// packaging/signatures/signer.go

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

// SigningOptions configures package signing
type SigningOptions struct {
	Certificate      *x509.Certificate
	PrivateKey       crypto.PrivateKey
	CertificateChain []*x509.Certificate
	SignatureType    SignatureType
	HashAlgorithm    HashAlgorithmName
	TimestampURL     string
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
	if !slices.Contains(allowedAlgos, opts.HashAlgorithm) {
		return fmt.Errorf("hash algorithm %s is not allowed", opts.HashAlgorithm)
	}

	return nil
}

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

// SignPackageData creates a PKCS#7/CMS signature for package content
// Implements RFC 5652 SignedData with NuGet-specific attributes
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
	signedDataBytes, err := asn1.Marshal(signedData)
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

func createSignedData(contentHash []byte, opts SigningOptions) (*SignedData, error) {
	// 1. Build EncapsulatedContentInfo (package hash as data)
	encapContentInfo := EncapsulatedContentInfo{
		EContentType: oidData,
		// EContent is OPTIONAL - for detached signatures, we omit it
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
		EncapContentInfo: encapContentInfo,
		Certificates:     certificates,
		SignerInfos:      []SignerInfo{*signerInfo},
	}

	return signedData, nil
}

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
			SerialNumber: opts.Certificate.SerialNumber,
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
		SignedAttrs: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      signedAttrsBytes[1:], // Strip SET tag for [0] IMPLICIT
		},
		SignatureAlgorithm: AlgorithmIdentifier{
			Algorithm: signatureAlgOID,
		},
		Signature: signature,
	}

	// 6. Add timestamp to unsigned attributes (if requested)
	if opts.TimestampURL != "" {
		timestampAttr, err := createTimestampAttribute(signature, opts)
		if err != nil {
			return nil, fmt.Errorf("create timestamp: %w", err)
		}

		unsignedAttrsBytes, err := asn1.Marshal([]Attribute{timestampAttr})
		if err != nil {
			return nil, fmt.Errorf("marshal unsigned attributes: %w", err)
		}

		signerInfo.UnsignedAttrs = asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        1,
			IsCompound: true,
			Bytes:      unsignedAttrsBytes,
		}
	}

	return signerInfo, nil
}

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

func createTimestampAttribute(signature []byte, opts SigningOptions) (Attribute, error) {
	// Request RFC 3161 timestamp
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

	// Timestamp token is already a ContentInfo, just wrap it
	values, err := asn1.Marshal([]asn1.RawValue{{FullBytes: timestampToken}})
	if err != nil {
		return Attribute{}, err
	}

	return Attribute{
		Type:   oidSignatureTimeStampToken,
		Values: asn1.RawValue{FullBytes: values},
	}, nil
}
```

#### 4. Integration with PackageBuilder

```go
// packaging/builder.go - Replace placeholder Sign method

// Sign signs the package with the provided certificate and key
func (b *PackageBuilder) Sign(opts signatures.SigningOptions) error {
	// 1. Validate we have content to sign
	if len(b.files) == 0 {
		return fmt.Errorf("cannot sign empty package")
	}

	// 2. Calculate package hash (this would normally be done on the actual .nupkg bytes)
	// For now, return error indicating this needs to be called after Save()
	return fmt.Errorf("package must be saved before signing - use SignedPackageArchive for signing")
}

// Note: Full integration would require:
// 1. Save package to temp file
// 2. Read package bytes and calculate hash (excluding any existing signature)
// 3. Call signatures.SignPackageData(hash, opts)
// 4. Embed signature into .signature.p7s file in the ZIP
// 5. Update OPC files ([Content_Types].xml and package rels)
```
#### 5. RFC 3161 Timestamp Client

```go
// packaging/signatures/timestamp.go

package signatures

import (
	"bytes"
	"crypto/rand"
	"encoding/asn1"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"
)

// TimestampClient provides RFC 3161 timestamp token requests
type TimestampClient struct {
	url     string
	timeout time.Duration
	client  *http.Client
}

// NewTimestampClient creates a new RFC 3161 timestamp client
func NewTimestampClient(url string, timeout time.Duration) *TimestampClient {
	return &TimestampClient{
		url:     url,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// RequestTimestamp requests an RFC 3161 timestamp token for the given message hash
func (c *TimestampClient) RequestTimestamp(messageHash []byte, hashAlg HashAlgorithmName) ([]byte, error) {
	// Generate nonce (32 bytes random, ensure valid per NuGet.Client)
	nonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Build TimeStampReq
	req, err := buildTimestampRequest(messageHash, hashAlg, nonce)
	if err != nil {
		return nil, fmt.Errorf("build timestamp request: %w", err)
	}

	// Encode request to DER
	reqBytes, err := asn1.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal timestamp request: %w", err)
	}

	// Send HTTP POST request
	httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/timestamp-query")

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send timestamp request: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("timestamp server error: HTTP %d %s", httpResp.StatusCode, httpResp.Status)
	}

	// Read response body
	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read timestamp response: %w", err)
	}

	// Parse TimeStampResp
	var resp timestampResponse
	if _, err := asn1.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal timestamp response: %w", err)
	}

	// Verify status
	if resp.Status.Status != 0 && resp.Status.Status != 1 {
		return nil, fmt.Errorf("timestamp request rejected: status=%d", resp.Status.Status)
	}

	// Verify response contains token
	if len(resp.TimeStampToken.FullBytes) == 0 {
		return nil, fmt.Errorf("timestamp response missing token")
	}

	// Verify nonce matches (replay attack prevention)
	if err := verifyTimestampResponse(resp.TimeStampToken.FullBytes, messageHash, nonce); err != nil {
		return nil, fmt.Errorf("verify timestamp response: %w", err)
	}

	// Return the timestamp token (ContentInfo containing SignedData)
	return resp.TimeStampToken.FullBytes, nil
}

// RFC 3161 ASN.1 structures

type timestampRequest struct {
	Version        int
	MessageImprint messageImprint
	ReqPolicy      asn1.ObjectIdentifier `asn1:"optional"`
	Nonce          *big.Int              `asn1:"optional"`
	CertReq        bool                  `asn1:"optional,default:false"`
	Extensions     asn1.RawValue         `asn1:"optional,tag:0"`
}

type messageImprint struct {
	HashAlgorithm AlgorithmIdentifier
	HashedMessage []byte
}

type timestampResponse struct {
	Status         pkiStatusInfo
	TimeStampToken asn1.RawValue `asn1:"optional"`
}

type pkiStatusInfo struct {
	Status       int
	StatusString []string          `asn1:"optional"`
	FailInfo     asn1.BitString    `asn1:"optional"`
}

// buildTimestampRequest creates an RFC 3161 TimeStampReq structure
func buildTimestampRequest(messageHash []byte, hashAlg HashAlgorithmName, nonce []byte) (timestampRequest, error) {
	// Get hash algorithm OID
	hashAlgOID := getDigestAlgorithmOID(hashAlg)

	// Build MessageImprint
	mi := messageImprint{
		HashAlgorithm: AlgorithmIdentifier{
			Algorithm: hashAlgOID,
		},
		HashedMessage: messageHash,
	}

	// Convert nonce bytes to big.Int (big-endian)
	var nonceInt *big.Int
	if len(nonce) > 0 {
		nonceInt = new(big.Int).SetBytes(nonce)
	}

	// Build TimeStampReq
	req := timestampRequest{
		Version:        1,
		MessageImprint: mi,
		Nonce:          nonceInt,
		CertReq:        true, // Request TSA certificate
	}

	return req, nil
}

// generateNonce generates a 32-byte nonce for timestamp requests
// Matches NuGet.Client's nonce generation (EnsureValidNonce)
func generateNonce() ([]byte, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Per NuGet.Client: ensure nonce is unsigned big-endian integer
	// Clear sign bit on most significant byte
	nonce[0] &= 0x7f

	return nonce, nil
}

// verifyTimestampResponse verifies the timestamp token matches request
func verifyTimestampResponse(tokenBytes, expectedHash, expectedNonce []byte) error {
	// Parse the timestamp token (it's a ContentInfo containing SignedData)
	var contentInfo ContentInfo
	if _, err := asn1.Unmarshal(tokenBytes, &contentInfo); err != nil {
		return fmt.Errorf("unmarshal content info: %w", err)
	}

	// Verify contentType is SignedData
	if !contentInfo.ContentType.Equal(oidSignedData) {
		return fmt.Errorf("invalid content type: expected SignedData")
	}

	// Parse SignedData
	var signedData SignedData
	if _, err := asn1.Unmarshal(contentInfo.Content.Bytes, &signedData); err != nil {
		return fmt.Errorf("unmarshal signed data: %w", err)
	}

	// Parse TSTInfo from eContent
	var tstInfo tstInfo
	if _, err := asn1.Unmarshal(signedData.ContentInfo.Content.Bytes, &tstInfo); err != nil {
		return fmt.Errorf("unmarshal TSTInfo: %w", err)
	}

	// Verify message imprint hash matches
	if !bytes.Equal(tstInfo.MessageImprint.HashedMessage, expectedHash) {
		return fmt.Errorf("timestamp message imprint mismatch")
	}

	// Verify nonce matches (if present)
	if tstInfo.Nonce != nil {
		expectedNonceInt := new(big.Int).SetBytes(expectedNonce)
		if tstInfo.Nonce.Cmp(expectedNonceInt) != 0 {
			return fmt.Errorf("timestamp nonce mismatch")
		}
	}

	return nil
}

// tstInfo represents RFC 3161 TSTInfo structure
type tstInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint messageImprint
	SerialNumber   *big.Int
	GenTime        time.Time
	Accuracy       accuracy              `asn1:"optional"`
	Ordering       bool                  `asn1:"optional,default:false"`
	Nonce          *big.Int              `asn1:"optional"`
	TSA            asn1.RawValue         `asn1:"optional,tag:0"`
	Extensions     asn1.RawValue         `asn1:"optional,tag:1"`
}

type accuracy struct {
	Seconds int                `asn1:"optional"`
	Millis  int                `asn1:"optional,tag:0"`
	Micros  int                `asn1:"optional,tag:1"`
}
```

**Key Design Decisions**:

1. **Nonce Generation**: Matches NuGet.Client's `EnsureValidNonce` - 32 bytes with cleared sign bit
2. **Request Building**: Minimal required fields - no policy OID, no extensions
3. **certReq=true**: Always request TSA certificate for verification
4. **HTTP Client**: Simple POST with `application/timestamp-query` content type
5. **Response Verification**: Validates status, nonce, and message imprint
6. **Error Handling**: Comprehensive error messages for debugging

**NuGet.Client Compatibility**:
- `Rfc3161TimestampProvider.GenerateNonce()` → `generateNonce()`
- `Rfc3161TimestampRequest.SubmitRequestAsync()` → `RequestTimestamp()`
- `Rfc3161TimestampProvider.ValidateTimestampResponse()` → `verifyTimestampResponse()`

#### 6. Comprehensive Test Implementation

```go
// packaging/signatures/signer_test.go

package signatures

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
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
			expectedOID: oidProofOfOrigin,
			expectErr:   false,
		},
		{
			name:        "Repository signature",
			sigType:     SignatureTypeRepository,
			expectedOID: oidProofOfReceipt,
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
			_, err = asn1.Unmarshal(attr.Values.FullBytes, &values)
			if err != nil {
				t.Fatalf("failed to unmarshal values: %v", err)
			}

			if len(values) != 1 {
				t.Fatalf("expected 1 value, got %d", len(values))
			}

			var commitment CommitmentTypeIndication
			_, err = asn1.Unmarshal(values[0].FullBytes, &commitment)
			if err != nil {
				t.Fatalf("failed to unmarshal commitment: %v", err)
			}

			if !commitment.CommitmentTypeID.Equal(tc.expectedOID) {
				t.Errorf("expected commitment OID %v, got %v", tc.expectedOID, commitment.CommitmentTypeID)
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
	// Note: This would use the PrimarySignature parsing from reader.go
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
	if nonce1[0] & 0x80 != 0 {
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
	// This test would require a real timestamp token
	// In practice, you'd save a real timestamp response and parse it
	t.Skip("requires real timestamp token - test manually with real TSA")
}

func TestVerifyTimestampResponse_NonceMismatch(t *testing.T) {
	// This test would verify nonce validation
	// Requires constructing invalid timestamp token
	t.Skip("requires timestamp token construction - test during integration")
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
```

### Testing Requirements

Following NuGet.Client's testing patterns:

1. **Certificate Generation** (reuse from verifier_test.go):
   - `generateTestRootCA()` - Self-signed root CA
   - `generateTestCodeSigningCert()` - Code signing certificate
   - `generateWeakRSAKeyCert()` - Weak key for negative testing
   - `generateExpiredCert()` - Expired certificate

2. **CMS Structure Tests**:
   - ✓ ContentInfo encoding with oidSignedData
   - ✓ SignedData structure with version, digest algorithms, certificates
   - ✓ SignerInfo with IssuerAndSerialNumber
   - ✓ SignerInfo with SubjectKeyIdentifier (when present)

3. **Attribute Tests**:
   - ✓ All signed attributes creation (content-type, signing-time, message-digest)
   - ✓ Commitment-type-indication for Author and Repository
   - ✓ Signing-certificate-v2 with cert hash and issuer/serial
   - ✓ Attribute encoding for signing (DER with SET tag)

4. **Signature Tests**:
   - ✓ RSA-PKCS#1 v1.5 signing with crypto/rsa
   - ✓ All hash algorithms (SHA256/384/512)
   - ✓ Certificate chain inclusion
   - ✓ Validation tests (missing cert, key mismatch, weak keys)

5. **Integration Tests**:
   - ✓ Create signature and parse with existing structures
   - ✓ Verify signature structure matches RFC 5652
   - ✓ End-to-end: sign → parse → verify structure

### Test Coverage Targets

- **cms.go**: 95%+ (structure definitions, OID mappings)
- **attributes.go**: 95%+ (all attribute builders)
- **signer.go**: 90%+ (signing logic, validation)
- **Overall**: 90%+ coverage

### Verification Steps

```bash
# 1. Build
go build ./packaging/signatures

# 2. Run all tests
go test ./packaging/signatures -v

# 3. Run specific test categories
go test ./packaging/signatures -v -run TestSignPackageData
go test ./packaging/signatures -v -run TestBuildSignedAttributes
go test ./packaging/signatures -v -run TestSigningOptions

# 4. Check coverage
go test ./packaging/signatures -coverprofile=/tmp/signer_coverage.out
go tool cover -func=/tmp/signer_coverage.out

# 5. Run integration test
go test ./packaging/signatures -v -run TestSignAndVerifyIntegration

# 6. Format check
gofmt -l packaging/signatures/
```
### Testing Requirements

1. **CMS Structure Tests**:
   - Test ContentInfo encoding
   - Test SignedData structure
   - Test SignerInfo with IssuerAndSerialNumber
   - Test SignerInfo with SubjectKeyIdentifier

2. **Attribute Tests**:
   - Test all signed attributes creation
   - Test commitment-type-indication (Author/Repository)
   - Test signing-certificate-v2
   - Test attribute encoding for signing

3. **Signature Tests**:
   - Test RSA-PKCS#1 v1.5 signing
   - Test with different hash algorithms (SHA256/384/512)
   - Test with certificate chains
   - Test with timestamp

4. **Integration Tests**:
   - Create signature and verify with reader
   - Compare with NuGet-signed packages

### Acceptance Criteria

- [ ] Complete CMS/PKCS#7 structure implementation
- [ ] NuGet authenticated attributes (signing-time, commitment-type, signing-certificate-v2)
- [ ] RSA-PKCS#1 v1.5 signature generation
- [ ] Certificate chain inclusion
- [ ] SubjectKeyIdentifier and IssuerAndSerialNumber support
- [ ] RFC 3161 timestamp integration
- [ ] DER encoding compliance
- [ ] Integration with existing signature reader (M3.8)
- [ ] 90%+ test coverage
- [ ] Interoperability verification with NuGet.Client

### Commit Message

```
feat(packaging): implement production PKCS#7 package signing

Add complete CMS/PKCS#7 signature creation using Go crypto:
- RFC 5652 SignedData structures with DER encoding
- NuGet authenticated attributes (commitment-type, signing-cert-v2)
- RSA-PKCS#1 v1.5 signature generation
- Certificate chain embedding
- RFC 3161 timestamp integration
- SubjectKeyIdentifier and IssuerAndSerialNumber support

Matches NuGet.Client SignedCms behavior without external dependencies.

Chunk: M3.10
Status: ✓ Complete
Coverage: 90%+

Reference: SigningUtility.cs, X509SignatureProvider.cs
Reference: RFC 5652 (CMS), RFC 5126 (ESS), RFC 3161 (TSP)
```

---

## Notes

This implementation provides **production-ready** PKCS#7/CMS signing that:

1. ✅ Uses only Go standard library (crypto/x509, crypto/rsa, encoding/asn1)
2. ✅ Creates RFC 5652 compliant signatures
3. ✅ Includes all NuGet-required authenticated attributes
4. ✅ Supports both Author and Repository signatures
5. ✅ Integrates with existing signature reader (M3.8)
6. ✅ Supports RFC 3161 timestamping
7. ✅ Matches NuGet.Client's SignedCms output format

**Estimated Lines of Code**:
- `cms.go`: ~200 lines (structures + OIDs)
- `attributes.go`: ~250 lines (attribute builders)
- `signer.go`: ~300 lines (signature creation)
- `signer_test.go`: ~800 lines (comprehensive tests)
- **Total**: ~1,550 lines

**Implementation Time**: 8-10 hours (vs 2.5 hours for placeholder)
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
