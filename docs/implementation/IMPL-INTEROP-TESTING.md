# NuGet.Client Interop Testing - Implementation Guide

**Status**: Ready for Implementation
**Created**: 2025-10-21
**Dependencies**: M3.10 (Package Signature Creation)
**Estimated Time**: 12-16 hours
**Chunks**: I1-I4

---

## Overview

This guide implements bidirectional interop testing between gonuget and the official NuGet.Client library. The implementation validates that gonuget produces specification-compliant output by testing against Microsoft's reference implementation.

**Key Components**:
1. **Go CLI Bridge** - JSON-RPC interface exposing gonuget functionality
2. **C# Test Project** - xUnit tests using NuGet.Client for validation
3. **Test Infrastructure** - Certificate management, package generation, test data

**Architecture**: Process-based communication (Go CLI ↔ C# via JSON over stdin/stdout)

---

## Chunk I1: Go CLI Bridge - Foundation

**Estimated Time**: 3-4 hours
**Dependencies**: None (uses existing gonuget packages)

### Overview

Create the Go CLI bridge executable that exposes gonuget functionality via JSON-RPC over stdin/stdout. This chunk implements the core request/response handling and protocol definitions.

### Files to Create

1. `cmd/nuget-interop-test/main.go` (~150 lines)
2. `cmd/nuget-interop-test/protocol.go` (~200 lines)
3. `cmd/nuget-interop-test/helpers.go` (~150 lines)

### Implementation Details

#### File: `cmd/nuget-interop-test/main.go`

```go
// Package main implements a JSON-RPC bridge for gonuget interop testing.
// It receives requests via stdin and returns responses via stdout.
// This enables C# test projects to validate gonuget against NuGet.Client.
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Request represents an incoming test request from C# tests.
// Action specifies which gonuget operation to perform.
// Data contains action-specific parameters in JSON format.
type Request struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

// Response represents the standard response format sent back to C#.
// Success indicates whether the operation completed without errors.
// Data contains action-specific results (only present on success).
// Error contains detailed error information (only present on failure).
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo contains structured error information for debugging.
// Code is a machine-readable error code (e.g., "SIGN_001").
// Message is a human-readable error description.
// Details contains additional context (e.g., file paths, stack traces).
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func main() {
	// Disable log output to avoid contaminating JSON response
	// (tests may need to enable logging to files for debugging)

	// Read request from stdin
	var req Request
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&req); err != nil {
		sendError("REQ_001", "Failed to parse request JSON", err.Error())
		os.Exit(1)
	}

	// Route to appropriate handler based on action
	var handler Handler
	switch req.Action {
	// Signature operations
	case "sign_package":
		handler = &SignPackageHandler{}
	case "parse_signature":
		handler = &ParseSignatureHandler{}
	case "verify_signature":
		handler = &VerifySignatureHandler{}

	// Version operations
	case "compare_versions":
		handler = &CompareVersionsHandler{}
	case "parse_version":
		handler = &ParseVersionHandler{}

	// Framework operations
	case "check_framework_compat":
		handler = &CheckFrameworkCompatHandler{}
	case "parse_framework":
		handler = &ParseFrameworkHandler{}

	// Package operations
	case "read_package":
		handler = &ReadPackageHandler{}
	case "build_package":
		handler = &BuildPackageHandler{}

	default:
		sendError("ACT_001", "Unknown action", fmt.Sprintf("action=%s", req.Action))
		os.Exit(1)
	}

	// Execute handler
	result, err := handler.Handle(req.Data)
	if err != nil {
		sendError(handler.ErrorCode(), err.Error(), "")
		os.Exit(1)
	}

	// Send success response
	sendSuccess(result)
}

// Handler interface for all request handlers.
// Handle processes the request data and returns a result or error.
// ErrorCode returns the error code prefix for this handler.
type Handler interface {
	Handle(data json.RawMessage) (interface{}, error)
	ErrorCode() string
}

// sendSuccess writes a successful response to stdout.
func sendSuccess(data interface{}) {
	resp := Response{
		Success: true,
		Data:    data,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ") // Pretty print for debugging
	_ = encoder.Encode(resp)
}

// sendError writes an error response to stdout.
func sendError(code, message, details string) {
	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ") // Pretty print for debugging
	_ = encoder.Encode(resp)
}
```

#### File: `cmd/nuget-interop-test/protocol.go`

```go
package main

// SignPackageRequest represents a request to create a package signature.
// This matches the NuGet.Client SignPackageRequest semantics.
type SignPackageRequest struct {
	// PackageHash is the hash of the package content (ZIP archive).
	// This is the value that gets signed (detached signature).
	PackageHash []byte `json:"packageHash"`

	// CertPath is the filesystem path to the signing certificate.
	// Supports PEM, DER, or PFX formats.
	CertPath string `json:"certPath"`

	// CertPassword is optional password for encrypted certificates (PFX).
	CertPassword string `json:"certPassword,omitempty"`

	// KeyPath is the filesystem path to the private key (PEM format).
	// Not needed if CertPath points to PFX with embedded key.
	KeyPath string `json:"keyPath,omitempty"`

	// SignatureType is "Author" or "Repository".
	SignatureType string `json:"signatureType"`

	// HashAlgorithm is "SHA256", "SHA384", or "SHA512".
	HashAlgorithm string `json:"hashAlgorithm"`

	// TimestampURL is optional RFC 3161 timestamp authority URL.
	// If empty, signature is created without timestamp.
	TimestampURL string `json:"timestampURL,omitempty"`
}

// SignPackageResponse contains the created signature bytes.
type SignPackageResponse struct {
	// Signature is the DER-encoded PKCS#7 SignedData structure.
	// This can be loaded by NuGet.Client's PrimarySignature.Load().
	Signature []byte `json:"signature"`
}

// ParseSignatureRequest asks gonuget to parse a signature.
type ParseSignatureRequest struct {
	// Signature is DER-encoded PKCS#7 SignedData to parse.
	Signature []byte `json:"signature"`
}

// ParseSignatureResponse contains parsed signature metadata.
// This allows C# tests to verify gonuget extracts the same data as NuGet.Client.
type ParseSignatureResponse struct {
	// Type is "Author", "Repository", or "Unknown".
	Type string `json:"type"`

	// HashAlgorithm is "SHA256", "SHA384", "SHA512", or "Unknown".
	HashAlgorithm string `json:"hashAlgorithm"`

	// SignerCertHash is hex-encoded SubjectKeyId or certificate hash.
	SignerCertHash string `json:"signerCertHash"`

	// TimestampCount is the number of timestamps in the signature.
	TimestampCount int `json:"timestampCount"`

	// TimestampTimes are RFC3339-formatted timestamp generation times.
	TimestampTimes []string `json:"timestampTimes,omitempty"`

	// Certificates is the number of certificates in the chain.
	Certificates int `json:"certificates"`
}

// VerifySignatureRequest validates a signature.
type VerifySignatureRequest struct {
	// Signature is the signature to verify.
	Signature []byte `json:"signature"`

	// TrustedRoots are DER-encoded root CA certificates.
	TrustedRoots [][]byte `json:"trustedRoots,omitempty"`

	// AllowUntrustedRoot allows signatures with untrusted roots (for testing).
	AllowUntrustedRoot bool `json:"allowUntrustedRoot"`

	// RequireTimestamp requires the signature to be timestamped.
	RequireTimestamp bool `json:"requireTimestamp"`
}

// VerifySignatureResponse contains verification results.
type VerifySignatureResponse struct {
	// Valid is true if signature verification succeeded.
	Valid bool `json:"valid"`

	// Errors contains verification failure messages.
	Errors []string `json:"errors,omitempty"`

	// Warnings contains non-fatal verification warnings.
	Warnings []string `json:"warnings,omitempty"`

	// SignerSubject is the signer certificate's subject DN.
	SignerSubject string `json:"signerSubject,omitempty"`
}

// CompareVersionsRequest compares two NuGet version strings.
type CompareVersionsRequest struct {
	Version1 string `json:"version1"`
	Version2 string `json:"version2"`
}

// CompareVersionsResponse returns version comparison result.
type CompareVersionsResponse struct {
	// Result is -1 (v1 < v2), 0 (equal), or 1 (v1 > v2).
	Result int `json:"result"`
}

// ParseVersionRequest parses a NuGet version string.
type ParseVersionRequest struct {
	Version string `json:"version"`
}

// ParseVersionResponse contains parsed version components.
// This matches NuGet.Versioning.NuGetVersion structure.
type ParseVersionResponse struct {
	Major        int    `json:"major"`
	Minor        int    `json:"minor"`
	Patch        int    `json:"patch"`
	Revision     int    `json:"revision"`
	Release      string `json:"release"`      // Pre-release label (e.g., "beta.1")
	Metadata     string `json:"metadata"`     // Build metadata (e.g., "20130313144700")
	IsPrerelease bool   `json:"isPrerelease"` // True if has pre-release label
	IsLegacy     bool   `json:"isLegacy"`     // True if uses legacy 4-part format
}

// CheckFrameworkCompatRequest checks framework compatibility.
type CheckFrameworkCompatRequest struct {
	// PackageFramework is the framework the package supports (e.g., "net6.0").
	PackageFramework string `json:"packageFramework"`

	// ProjectFramework is the project's target framework (e.g., "net8.0").
	ProjectFramework string `json:"projectFramework"`
}

// CheckFrameworkCompatResponse returns compatibility result.
type CheckFrameworkCompatResponse struct {
	// Compatible is true if project can use package.
	Compatible bool `json:"compatible"`

	// Reason explains why incompatible (empty if compatible).
	Reason string `json:"reason,omitempty"`
}

// ParseFrameworkRequest parses a framework identifier.
type ParseFrameworkRequest struct {
	Framework string `json:"framework"`
}

// ParseFrameworkResponse contains parsed framework components.
type ParseFrameworkResponse struct {
	// Identifier is the framework identifier (e.g., ".NETCoreApp", ".NETFramework").
	Identifier string `json:"identifier"`

	// Version is the framework version (e.g., "6.0", "4.7.2").
	Version string `json:"version"`

	// Profile is optional profile (e.g., "Client" for ".NETFramework,Profile=Client").
	Profile string `json:"profile,omitempty"`

	// Platform is optional platform (e.g., "windows7.0").
	Platform string `json:"platform,omitempty"`
}

// ReadPackageRequest reads package structure and metadata.
type ReadPackageRequest struct {
	// PackageBytes is the ZIP package content.
	PackageBytes []byte `json:"packageBytes"`
}

// ReadPackageResponse contains package metadata.
type ReadPackageResponse struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Authors      []string `json:"authors,omitempty"`
	Description  string   `json:"description,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"` // Formatted as "id:version"
	FileCount    int      `json:"fileCount"`
	HasSignature bool     `json:"hasSignature"`
	SignatureType string  `json:"signatureType,omitempty"` // "Author", "Repository", or empty
}

// BuildPackageRequest creates a minimal NuGet package.
type BuildPackageRequest struct {
	ID          string            `json:"id"`
	Version     string            `json:"version"`
	Authors     []string          `json:"authors,omitempty"`
	Description string            `json:"description,omitempty"`
	Files       map[string][]byte `json:"files"` // Relative path -> content
}

// BuildPackageResponse contains the built package.
type BuildPackageResponse struct {
	// PackageBytes is the ZIP package content.
	PackageBytes []byte `json:"packageBytes"`
}
```

#### File: `cmd/nuget-interop-test/helpers.go`

```go
package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"software.sslmate.com/src/go-pkcs12"
)

// loadCertificate loads a certificate from PEM, DER, or PFX format.
// For PFX files, password may be required.
func loadCertificate(path, password string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Try PEM format first
	block, _ := pem.Decode(data)
	if block != nil && block.Type == "CERTIFICATE" {
		return x509.ParseCertificate(block.Bytes)
	}

	// Try PFX format
	if password != "" || isPFX(data) {
		_, cert, _, err := pkcs12.DecodeChain(data, password)
		if err != nil {
			return nil, fmt.Errorf("decode PFX: %w", err)
		}
		return cert, nil
	}

	// Try DER format
	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, fmt.Errorf("parse certificate (tried PEM, PFX, DER): %w", err)
	}
	return cert, nil
}

// loadPrivateKey loads a private key from PEM format.
// Supports RSA, ECDSA, and Ed25519 keys in PKCS#8 format.
func loadPrivateKey(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Parse PEM block
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	// Try PKCS#8 format (most common)
	if block.Type == "PRIVATE KEY" {
		return x509.ParsePKCS8PrivateKey(block.Bytes)
	}

	// Try legacy RSA format
	if block.Type == "RSA PRIVATE KEY" {
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}

	// Try EC format
	if block.Type == "EC PRIVATE KEY" {
		return x509.ParseECPrivateKey(block.Bytes)
	}

	return nil, fmt.Errorf("unsupported key type: %s", block.Type)
}

// loadPrivateKeyFromPFX loads private key from PFX file.
func loadPrivateKeyFromPFX(path, password string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	key, _, _, err := pkcs12.DecodeChain(data, password)
	if err != nil {
		return nil, fmt.Errorf("decode PFX: %w", err)
	}

	return key, nil
}

// isPFX checks if data looks like a PFX file.
// PFX files start with the PKCS#12 magic bytes.
func isPFX(data []byte) bool {
	// PKCS#12 magic: 0x30 (SEQUENCE)
	if len(data) < 4 {
		return false
	}
	return data[0] == 0x30
}

// formatError formats an error with context for debugging.
func formatError(operation string, err error) error {
	return fmt.Errorf("%s: %w", operation, err)
}
```

### Testing

Create a simple test to verify the CLI works:

```bash
# Test the help/error case
echo '{"action":"unknown"}' | go run ./cmd/nuget-interop-test
# Should return error response with ACT_001

# Test signature parsing (will fail without real signature, but tests routing)
echo '{"action":"parse_signature","data":{"signature":""}}' | go run ./cmd/nuget-interop-test
```

### Acceptance Criteria

- ✅ CLI accepts JSON requests from stdin
- ✅ CLI returns JSON responses to stdout
- ✅ Unknown actions return ACT_001 error
- ✅ Invalid JSON returns REQ_001 error
- ✅ All protocol types are properly defined
- ✅ Helper functions support PEM, DER, and PFX formats

---

## Chunk I2: Go CLI Bridge - Signature Handlers

**Estimated Time**: 3-4 hours
**Dependencies**: I1, M3.10 (Signature Implementation)

### Overview

Implement signature-related handlers (sign, parse, verify) that bridge gonuget signature functionality to the C# test suite.

### Files to Modify/Create

1. `cmd/nuget-interop-test/handlers_signature.go` (~500 lines) - NEW

### Implementation Details

#### File: `cmd/nuget-interop-test/handlers_signature.go`

```go
package main

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/willibrandon/gonuget/packaging/signatures"
)

// SignPackageHandler creates a PKCS#7 package signature.
type SignPackageHandler struct{}

func (h *SignPackageHandler) ErrorCode() string { return "SIGN_001" }

func (h *SignPackageHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req SignPackageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate required fields
	if len(req.PackageHash) == 0 {
		return nil, fmt.Errorf("packageHash is required")
	}
	if req.CertPath == "" {
		return nil, fmt.Errorf("certPath is required")
	}

	// Load certificate
	cert, err := loadCertificate(req.CertPath, req.CertPassword)
	if err != nil {
		return nil, fmt.Errorf("load certificate: %w", err)
	}

	// Load private key
	var key interface{}
	if req.KeyPath != "" {
		// Separate key file
		key, err = loadPrivateKey(req.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("load private key: %w", err)
		}
	} else {
		// Key embedded in PFX
		key, err = loadPrivateKeyFromPFX(req.CertPath, req.CertPassword)
		if err != nil {
			return nil, fmt.Errorf("load private key from PFX: %w", err)
		}
	}

	// Parse signature type
	var sigType signatures.SignatureType
	switch req.SignatureType {
	case "Author":
		sigType = signatures.SignatureTypeAuthor
	case "Repository":
		sigType = signatures.SignatureTypeRepository
	default:
		return nil, fmt.Errorf("invalid signature type: %s (must be 'Author' or 'Repository')", req.SignatureType)
	}

	// Parse hash algorithm
	var hashAlg signatures.HashAlgorithmName
	switch req.HashAlgorithm {
	case "SHA256":
		hashAlg = signatures.HashAlgorithmSHA256
	case "SHA384":
		hashAlg = signatures.HashAlgorithmSHA384
	case "SHA512":
		hashAlg = signatures.HashAlgorithmSHA512
	default:
		return nil, fmt.Errorf("invalid hash algorithm: %s (must be 'SHA256', 'SHA384', or 'SHA512')", req.HashAlgorithm)
	}

	// Create signing options
	opts := signatures.SigningOptions{
		Certificate:   cert,
		PrivateKey:    key,
		SignatureType: sigType,
		HashAlgorithm: hashAlg,
		TimestampURL:  req.TimestampURL,
	}

	// Validate options
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid signing options: %w", err)
	}

	// Sign package
	signature, err := signatures.SignPackageData(req.PackageHash, opts)
	if err != nil {
		return nil, fmt.Errorf("sign package: %w", err)
	}

	return SignPackageResponse{
		Signature: signature,
	}, nil
}

// ParseSignatureHandler parses a PKCS#7 signature structure.
type ParseSignatureHandler struct{}

func (h *ParseSignatureHandler) ErrorCode() string { return "PARSE_001" }

func (h *ParseSignatureHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ParseSignatureRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate signature data
	if len(req.Signature) == 0 {
		return nil, fmt.Errorf("signature is required")
	}

	// Parse signature using gonuget
	sig, err := signatures.ReadSignature(req.Signature)
	if err != nil {
		return nil, fmt.Errorf("read signature: %w", err)
	}

	// Extract timestamp information
	var timestampTimes []string
	for _, ts := range sig.Timestamps {
		// Format as RFC3339 for JSON
		timestampTimes = append(timestampTimes, ts.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Compute signer certificate hash
	// Use SubjectKeyId if available, otherwise hash the cert
	signerCertHash := ""
	if sig.SignerCertificate != nil {
		if len(sig.SignerCertificate.SubjectKeyId) > 0 {
			signerCertHash = hex.EncodeToString(sig.SignerCertificate.SubjectKeyId)
		} else {
			// Fallback: SHA256 of cert
			h := sha256.Sum256(sig.SignerCertificate.Raw)
			signerCertHash = hex.EncodeToString(h[:])
		}
	}

	return ParseSignatureResponse{
		Type:           sig.Type.String(),
		HashAlgorithm:  sig.HashAlgorithm.String(),
		SignerCertHash: signerCertHash,
		TimestampCount: len(sig.Timestamps),
		TimestampTimes: timestampTimes,
		Certificates:   len(sig.Certificates),
	}, nil
}

// VerifySignatureHandler verifies a signature.
type VerifySignatureHandler struct{}

func (h *VerifySignatureHandler) ErrorCode() string { return "VERIFY_001" }

func (h *VerifySignatureHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req VerifySignatureRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Validate signature data
	if len(req.Signature) == 0 {
		return nil, fmt.Errorf("signature is required")
	}

	// Parse signature
	sig, err := signatures.ReadSignature(req.Signature)
	if err != nil {
		return nil, fmt.Errorf("read signature: %w", err)
	}

	// Build trust store from provided roots
	trustStore := signatures.NewTrustStore()
	for i, rootBytes := range req.TrustedRoots {
		cert, err := x509.ParseCertificate(rootBytes)
		if err != nil {
			return nil, fmt.Errorf("parse trusted root %d: %w", i, err)
		}
		trustStore.AddCertificate(cert)
	}

	// Create verification options
	opts := signatures.VerificationOptions{
		TrustStore:         trustStore,
		AllowUntrustedRoot: req.AllowUntrustedRoot,
		RequireTimestamp:   req.RequireTimestamp,
		AllowedSignatureTypes: []signatures.SignatureType{
			signatures.SignatureTypeAuthor,
			signatures.SignatureTypeRepository,
		},
		AllowedHashAlgorithms: []signatures.HashAlgorithmName{
			signatures.HashAlgorithmSHA256,
			signatures.HashAlgorithmSHA384,
			signatures.HashAlgorithmSHA512,
		},
	}

	// Verify signature
	result, err := signatures.VerifySignature(sig, opts)

	// Collect errors and warnings
	var errors, warnings []string
	if err != nil {
		errors = append(errors, err.Error())
	}
	if result != nil {
		warnings = result.Warnings
	}

	// Extract signer subject
	signerSubject := ""
	if sig.SignerCertificate != nil {
		signerSubject = sig.SignerCertificate.Subject.String()
	}

	return VerifySignatureResponse{
		Valid:         err == nil,
		Errors:        errors,
		Warnings:      warnings,
		SignerSubject: signerSubject,
	}, nil
}
```

### Testing

```bash
# Create test certificate and key (for manual testing)
openssl req -x509 -newkey rsa:2048 -keyout test_key.pem -out test_cert.pem \
  -days 365 -nodes -subj "/CN=Test Signer"

# Test signing (requires valid cert/key)
cat > sign_request.json <<EOF
{
  "action": "sign_package",
  "data": {
    "packageHash": "$(echo -n 'test content' | sha256sum | cut -d' ' -f1 | xxd -r -p | base64)",
    "certPath": "test_cert.pem",
    "keyPath": "test_key.pem",
    "signatureType": "Author",
    "hashAlgorithm": "SHA256"
  }
}
EOF

cat sign_request.json | go run ./cmd/nuget-interop-test
```

### Acceptance Criteria

- ✅ `sign_package` creates valid PKCS#7 signatures
- ✅ `parse_signature` extracts correct metadata
- ✅ `verify_signature` validates signatures correctly
- ✅ Supports PEM, DER, and PFX certificate formats
- ✅ Handles missing/invalid certificates gracefully
- ✅ Returns detailed error messages

---

## Chunk I3: C# Test Project - Foundation & Signature Tests

**Estimated Time**: 4-5 hours
**Dependencies**: I2

### Overview

Create the C# xUnit test project with NuGet.Client dependencies, implement the Go CLI bridge helper, and create signature interop tests.

### Files to Create

1. `tests/nuget-client-interop/GonugetInterop.Tests/GonugetInterop.Tests.csproj` (~50 lines)
2. `tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs` (~200 lines)
3. `tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/TestCertificates.cs` (~150 lines)
4. `tests/nuget-client-interop/GonugetInterop.Tests/SignatureTests.cs` (~400 lines)
5. `tests/nuget-client-interop/Makefile` (~50 lines)

### Implementation Details

#### File: `tests/nuget-client-interop/GonugetInterop.Tests/GonugetInterop.Tests.csproj`

```xml
<Project Sdk="Microsoft.NET.Sdk">

  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <IsPackable>false</IsPackable>
    <IsTestProject>true</IsTestProject>
    <LangVersion>latest</LangVersion>
    <Nullable>enable</Nullable>
  </PropertyGroup>

  <ItemGroup>
    <!-- Test framework -->
    <PackageReference Include="Microsoft.NET.Test.Sdk" Version="17.8.0" />
    <PackageReference Include="xunit" Version="2.6.1" />
    <PackageReference Include="xunit.runner.visualstudio" Version="2.5.3">
      <IncludeAssets>runtime; build; native; contentfiles; analyzers; buildtransitive</IncludeAssets>
      <PrivateAssets>all</PrivateAssets>
    </PackageReference>

    <!-- NuGet.Client libraries -->
    <PackageReference Include="NuGet.Packaging" Version="6.8.0" />
    <PackageReference Include="NuGet.Versioning" Version="6.8.0" />
    <PackageReference Include="NuGet.Frameworks" Version="6.8.0" />

    <!-- JSON serialization -->
    <PackageReference Include="System.Text.Json" Version="8.0.0" />

    <!-- Test utilities -->
    <PackageReference Include="FluentAssertions" Version="6.12.0" />
  </ItemGroup>

  <ItemGroup>
    <!-- Copy gonuget CLI to output directory -->
    <None Include="../../../cmd/nuget-interop-test/gonuget-interop-test"
          CopyToOutputDirectory="PreserveNewest"
          Condition="Exists('../../../cmd/nuget-interop-test/gonuget-interop-test')" />
  </ItemGroup>

</Project>
```

#### File: `tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs`

```csharp
using System.Diagnostics;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Bridge to communicate with the gonuget CLI bridge process.
/// Sends JSON requests via stdin and receives JSON responses via stdout.
/// </summary>
public static class GonugetBridge
{
    private static readonly string GonugetPath = FindGonugetExecutable();

    /// <summary>
    /// Signs a package hash using gonuget and returns the PKCS#7 signature.
    /// </summary>
    public static byte[] SignPackage(
        byte[] packageHash,
        string certPath,
        string? certPassword = null,
        string? keyPath = null,
        string signatureType = "Author",
        string hashAlgorithm = "SHA256",
        string? timestampURL = null)
    {
        var request = new
        {
            action = "sign_package",
            data = new
            {
                packageHash,
                certPath,
                certPassword,
                keyPath,
                signatureType,
                hashAlgorithm,
                timestampURL
            }
        };

        var response = Execute<SignPackageResponse>(request);
        return response.Signature;
    }

    /// <summary>
    /// Parses a signature using gonuget and returns metadata.
    /// </summary>
    public static ParseSignatureResponse ParseSignature(byte[] signature)
    {
        var request = new
        {
            action = "parse_signature",
            data = new { signature }
        };

        return Execute<ParseSignatureResponse>(request);
    }

    /// <summary>
    /// Verifies a signature using gonuget.
    /// </summary>
    public static VerifySignatureResponse VerifySignature(
        byte[] signature,
        byte[][]? trustedRoots = null,
        bool allowUntrustedRoot = false,
        bool requireTimestamp = false)
    {
        var request = new
        {
            action = "verify_signature",
            data = new
            {
                signature,
                trustedRoots = trustedRoots ?? Array.Empty<byte[]>(),
                allowUntrustedRoot,
                requireTimestamp
            }
        };

        return Execute<VerifySignatureResponse>(request);
    }

    /// <summary>
    /// Executes a request against the gonuget CLI and deserializes the response.
    /// </summary>
    private static TResponse Execute<TResponse>(object request)
    {
        var psi = new ProcessStartInfo
        {
            FileName = GonugetPath,
            RedirectStandardInput = true,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
            CreateNoWindow = true
        };

        using var process = Process.Start(psi)
            ?? throw new InvalidOperationException("Failed to start gonuget process");

        // Send request as JSON
        var requestJson = JsonSerializer.Serialize(request, new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
        });

        process.StandardInput.WriteLine(requestJson);
        process.StandardInput.Close();

        // Read response
        var outputJson = process.StandardOutput.ReadToEnd();
        var errorOutput = process.StandardError.ReadToEnd();

        process.WaitForExit(timeoutMilliseconds: 30000); // 30 second timeout

        if (!string.IsNullOrEmpty(errorOutput))
        {
            throw new Exception($"gonuget stderr: {errorOutput}");
        }

        // Parse response envelope
        var envelope = JsonSerializer.Deserialize<ResponseEnvelope>(outputJson, new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase
        }) ?? throw new Exception("Failed to deserialize response");

        if (!envelope.Success)
        {
            var error = envelope.Error ?? new ErrorInfo { Message = "Unknown error" };
            throw new GonugetException(error.Code, error.Message, error.Details);
        }

        // Deserialize data payload
        return JsonSerializer.Deserialize<TResponse>(
            JsonSerializer.Serialize(envelope.Data),
            new JsonSerializerOptions { PropertyNamingPolicy = JsonNamingPolicy.CamelCase }
        ) ?? throw new Exception("Failed to deserialize response data");
    }

    /// <summary>
    /// Finds the gonuget executable in the test output directory or build location.
    /// </summary>
    private static string FindGonugetExecutable()
    {
        // Check test output directory first
        var testDir = AppContext.BaseDirectory;
        var exePath = Path.Combine(testDir, "gonuget-interop-test");
        if (File.Exists(exePath))
            return exePath;

        // Check relative to repository root (for local development)
        var repoRoot = Path.GetFullPath(Path.Combine(testDir, "../../../../../"));
        exePath = Path.Combine(repoRoot, "cmd/nuget-interop-test/gonuget-interop-test");
        if (File.Exists(exePath))
            return exePath;

        throw new FileNotFoundException(
            "gonuget-interop-test executable not found. " +
            "Run 'make build-interop' before running tests.");
    }

    // Response types
    private class ResponseEnvelope
    {
        public bool Success { get; set; }
        public object? Data { get; set; }
        public ErrorInfo? Error { get; set; }
    }

    private class ErrorInfo
    {
        public string Code { get; set; } = "";
        public string Message { get; set; } = "";
        public string? Details { get; set; }
    }

    public class SignPackageResponse
    {
        public byte[] Signature { get; set; } = Array.Empty<byte>();
    }

    public class ParseSignatureResponse
    {
        public string Type { get; set; } = "";
        public string HashAlgorithm { get; set; } = "";
        public string SignerCertHash { get; set; } = "";
        public int TimestampCount { get; set; }
        public string[]? TimestampTimes { get; set; }
        public int Certificates { get; set; }
    }

    public class VerifySignatureResponse
    {
        public bool Valid { get; set; }
        public string[]? Errors { get; set; }
        public string[]? Warnings { get; set; }
        public string? SignerSubject { get; set; }
    }
}

/// <summary>
/// Exception thrown when gonuget returns an error.
/// </summary>
public class GonugetException : Exception
{
    public string Code { get; }
    public string? Details { get; }

    public GonugetException(string code, string message, string? details = null)
        : base($"[{code}] {message}")
    {
        Code = code;
        Details = details;
    }
}
```

#### File: `tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/TestCertificates.cs`

```csharp
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;

namespace GonugetInterop.Tests.TestHelpers;

/// <summary>
/// Manages test certificates for signing tests.
/// Generates self-signed certificates in memory or exports to files.
/// </summary>
public static class TestCertificates
{
    /// <summary>
    /// Creates a self-signed code signing certificate for testing.
    /// </summary>
    public static X509Certificate2 CreateTestCodeSigningCertificate(
        string subjectName = "CN=Test Code Signing",
        int keySize = 2048,
        int validDays = 365)
    {
        using var rsa = RSA.Create(keySize);

        var request = new CertificateRequest(
            subjectName,
            rsa,
            HashAlgorithmName.SHA256,
            RSASignaturePadding.Pkcs1);

        // Add code signing extended key usage
        request.CertificateExtensions.Add(
            new X509EnhancedKeyUsageExtension(
                new OidCollection
                {
                    new Oid("1.3.6.1.5.5.7.3.3") // Code signing
                },
                critical: true));

        // Add key usage
        request.CertificateExtensions.Add(
            new X509KeyUsageExtension(
                X509KeyUsageFlags.DigitalSignature,
                critical: true));

        // Add subject key identifier
        request.CertificateExtensions.Add(
            new X509SubjectKeyIdentifierExtension(
                request.PublicKey,
                critical: false));

        var notBefore = DateTimeOffset.UtcNow.AddDays(-1);
        var notAfter = DateTimeOffset.UtcNow.AddDays(validDays);

        var cert = request.CreateSelfSigned(notBefore, notAfter);

        // Export and re-import to ensure private key is included
        var pfxBytes = cert.Export(X509ContentType.Pfx, "test");
        return new X509Certificate2(pfxBytes, "test", X509KeyStorageFlags.Exportable);
    }

    /// <summary>
    /// Exports a certificate to PEM format (certificate only, no private key).
    /// </summary>
    public static void ExportCertificateToPem(X509Certificate2 cert, string path)
    {
        var pem = PemEncoding.Write("CERTIFICATE", cert.RawData);
        File.WriteAllText(path, pem);
    }

    /// <summary>
    /// Exports a certificate's private key to PEM format (PKCS#8).
    /// </summary>
    public static void ExportPrivateKeyToPem(X509Certificate2 cert, string path)
    {
        if (cert.PrivateKey == null)
            throw new InvalidOperationException("Certificate has no private key");

        var rsa = cert.GetRSAPrivateKey()
            ?? throw new InvalidOperationException("Not an RSA certificate");

        var pkcs8 = rsa.ExportPkcs8PrivateKey();
        var pem = PemEncoding.Write("PRIVATE KEY", pkcs8);
        File.WriteAllText(path, pem);
    }

    /// <summary>
    /// Exports certificate to PFX format with password.
    /// </summary>
    public static void ExportToPfx(X509Certificate2 cert, string path, string password)
    {
        var pfxBytes = cert.Export(X509ContentType.Pfx, password);
        File.WriteAllBytes(path, pfxBytes);
    }

    /// <summary>
    /// Creates an expired certificate for negative testing.
    /// </summary>
    public static X509Certificate2 CreateExpiredCertificate(string subjectName = "CN=Expired Test Cert")
    {
        using var rsa = RSA.Create(2048);

        var request = new CertificateRequest(
            subjectName,
            rsa,
            HashAlgorithmName.SHA256,
            RSASignaturePadding.Pkcs1);

        // Certificate expired 30 days ago
        var notBefore = DateTimeOffset.UtcNow.AddDays(-60);
        var notAfter = DateTimeOffset.UtcNow.AddDays(-30);

        var cert = request.CreateSelfSigned(notBefore, notAfter);

        var pfxBytes = cert.Export(X509ContentType.Pfx, "test");
        return new X509Certificate2(pfxBytes, "test", X509KeyStorageFlags.Exportable);
    }
}
```

**[Continued in next part due to length...]**

### Acceptance Criteria for Chunk I3

- ✅ C# project builds successfully with NuGet.Client dependencies
- ✅ GonugetBridge successfully communicates with Go CLI
- ✅ Test certificates can be generated and exported
- ✅ At least 20 signature tests pass
- ✅ Tests validate gonuget signatures with NuGet.Client
- ✅ Tests parse NuGet.Client signatures with gonuget

---

## Chunk I4: Remaining Handlers & Tests

**Estimated Time**: 3-4 hours
**Dependencies**: I3

### Overview

Implement the remaining handlers (version, framework, package) and their corresponding C# tests.

### Files to Create/Modify

1. `cmd/nuget-interop-test/handlers_version.go` (~150 lines)
2. `cmd/nuget-interop-test/handlers_framework.go` (~150 lines)
3. `cmd/nuget-interop-test/handlers_package.go` (~200 lines)
4. `tests/nuget-client-interop/GonugetInterop.Tests/VersionTests.cs` (~300 lines)
5. `tests/nuget-client-interop/GonugetInterop.Tests/FrameworkTests.cs` (~300 lines)
6. `tests/nuget-client-interop/GonugetInterop.Tests/PackageReaderTests.cs` (~300 lines)

### Total Lines: ~1,400 lines (all files under 500 lines)

### Testing

Run the complete test suite:

```bash
make test-interop
```

Expected output:
```
Building gonuget interop CLI...
Running NuGet.Client interop tests...
  Passed SignatureTests.GonugetAuthorSignature_VerifiesWithNuGetClient [42 ms]
  Passed SignatureTests.GonugetRepositorySignature_VerifiesWithNuGetClient [38 ms]
  ... (40 signature tests)
  ... (180 version tests)
  ... (60 framework tests)
  ... (30 package tests)

Total tests: 330
  Passed: 330
  Failed: 0
  Skipped: 0

Time: 4.2s
```

### Acceptance Criteria

- ✅ All 330 interop tests pass
- ✅ Test suite completes in < 5 seconds
- ✅ No flaky tests (100% pass rate across 10 runs)
- ✅ Makefile targets work correctly
- ✅ CI pipeline integration ready

---

## Build and CI Integration

### Makefile

Create `Makefile` in repository root (if not exists) or add targets:

```makefile
.PHONY: build-interop test-interop test-all clean-interop

# Build the Go CLI bridge for interop testing
build-interop:
	@echo "Building gonuget interop CLI..."
	@go build -o cmd/nuget-interop-test/gonuget-interop-test ./cmd/nuget-interop-test

# Run interop tests (requires .NET SDK)
test-interop: build-interop
	@echo "Running NuGet.Client interop tests..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet test --verbosity normal

# Run all tests (Go unit tests + interop tests)
test-all: test test-interop

# Clean interop build artifacts
clean-interop:
	@rm -f cmd/nuget-interop-test/gonuget-interop-test
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet clean
```

### GitHub Actions

Add to `.github/workflows/ci.yml`:

```yaml
jobs:
  interop-tests:
    name: Interop Tests (NuGet.Client)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Setup .NET
        uses: actions/setup-dotnet@v4
        with:
          dotnet-version: '8.0'

      - name: Build Go interop CLI
        run: make build-interop

      - name: Run interop tests
        run: make test-interop

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: interop-test-results
          path: tests/nuget-client-interop/GonugetInterop.Tests/TestResults/
```

---

## Success Metrics

### Functional Metrics

- ✅ 100% of gonuget-created signatures verify with `PackageSignatureVerifier`
- ✅ 100% of NuGet.Client signatures parse correctly in gonuget
- ✅ Version comparison: 1000+ test cases, 100% match
- ✅ Framework compatibility: 100% match with FrameworkReducer
- ✅ Packages built by gonuget install with `dotnet add package`

### Performance Metrics

- ✅ Full test suite (330 tests) completes in < 5 seconds
- ✅ No test timeouts or flakiness
- ✅ Parallel execution works correctly

### Code Quality Metrics

- ✅ All files under 500 lines (target: under 400)
- ✅ No duplicated code between handlers
- ✅ Proper error handling with specific error codes
- ✅ Clear documentation in all public APIs

---

## Implementation Checklist

### Chunk I1: Foundation
- [ ] Create `cmd/nuget-interop-test/` directory
- [ ] Implement `main.go` (request routing)
- [ ] Implement `protocol.go` (all request/response types)
- [ ] Implement `helpers.go` (certificate loading)
- [ ] Test CLI accepts JSON and returns responses

### Chunk I2: Signature Handlers
- [ ] Implement `handlers_signature.go`
  - [ ] SignPackageHandler
  - [ ] ParseSignatureHandler
  - [ ] VerifySignatureHandler
- [ ] Manual test with real certificates
- [ ] Verify error handling

### Chunk I3: C# Foundation & Signature Tests
- [ ] Create C# test project structure
- [ ] Add NuGet.Client dependencies
- [ ] Implement `GonugetBridge.cs`
- [ ] Implement `TestCertificates.cs`
- [ ] Implement `SignatureTests.cs` (40 tests)
- [ ] All signature tests pass

### Chunk I4: Remaining Handlers & Tests
- [ ] Implement `handlers_version.go`
- [ ] Implement `handlers_framework.go`
- [ ] Implement `handlers_package.go`
- [ ] Implement `VersionTests.cs` (180 tests)
- [ ] Implement `FrameworkTests.cs` (60 tests)
- [ ] Implement `PackageReaderTests.cs` (30 tests)
- [ ] All 330 tests pass
- [ ] CI integration complete

---

## Troubleshooting

### Issue: "gonuget-interop-test not found"

**Solution**: Run `make build-interop` before running tests.

### Issue: Tests timeout

**Solution**:
- Check if Go CLI is hanging (add debug logging)
- Increase timeout in `GonugetBridge.Execute()` (currently 30s)
- For timestamp tests, allow more time for TSA network calls

### Issue: Certificate format errors

**Solution**:
- Ensure test certificates include private keys
- Use `X509KeyStorageFlags.Exportable` when creating certificates
- Export to PFX format with password for maximum compatibility

### Issue: JSON serialization errors

**Solution**:
- Ensure property names match between C# and Go (use camelCase)
- Check for null values in optional fields
- Verify byte arrays are base64-encoded in JSON

---

## Future Enhancements

### Phase 2 (Post-MVP)

1. **Additional Test Categories**
   - Dependency resolution tests
   - Package installation tests
   - Feed protocol tests

2. **Performance Benchmarking**
   - Compare gonuget vs NuGet.Client performance
   - Identify optimization opportunities

3. **Fuzzing Integration**
   - Generate random packages
   - Feed to both implementations
   - Compare results

### Phase 3 (Advanced)

1. **Real Package Corpus**
   - Test against NuGet.org packages
   - Build compatibility matrix
   - Regression testing

2. **Cross-Platform Testing**
   - Windows, Linux, macOS
   - Different .NET SDK versions
   - Certificate store integration

---

**End of Implementation Guide**

Total estimated implementation time: **12-16 hours**
Total code to write: **~3,000 lines** (all files < 500 lines)
