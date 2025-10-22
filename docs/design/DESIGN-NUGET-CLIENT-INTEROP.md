# NuGet.Client Interop Testing - Design Document

**Status**: Draft
**Created**: 2025-01-21
**Author**: gonuget team
**Version**: 1.0

---

## Table of Contents

1. [Overview](#overview)
2. [Goals and Non-Goals](#goals-and-non-goals)
3. [Architecture](#architecture)
4. [Component Design](#component-design)
5. [API Specifications](#api-specifications)
6. [Implementation Details](#implementation-details)
7. [Test Specifications](#test-specifications)
8. [Build and CI Integration](#build-and-ci-integration)
9. [Performance Considerations](#performance-considerations)
10. [Security Considerations](#security-considerations)
11. [Future Enhancements](#future-enhancements)

---

## Overview

### Problem Statement

gonuget needs to ensure 100% compatibility with the official NuGet ecosystem. While unit tests validate individual components, we need **authoritative validation** that gonuget's output is identical to or interoperable with the official NuGet.Client implementation.

### Solution

Create a bidirectional interop test suite using C# projects that consume NuGet.Client libraries (NuGet.Packaging, NuGet.Versioning, NuGet.Frameworks) to validate gonuget's correctness. This provides production-level confidence that gonuget is specification-compliant.

### Key Benefits

1. **Authoritative Validation**: NuGet.Client is the reference implementation
2. **Bidirectional Testing**: Validates both reading and writing
3. **Real-world Scenarios**: Tests actual package workflows
4. **Regression Prevention**: Catches compatibility breaks before release
5. **Polyglot Synergy**: Leverages C# expertise alongside Go
6. **CI/CD Ready**: Integrates cleanly into build pipeline

---

## Goals and Non-Goals

### Goals

- ✅ Validate gonuget-created signatures with NuGet.Client's `PackageSignatureVerifier`
- ✅ Parse NuGet.Client-created signatures with gonuget signature reader
- ✅ Verify identical version comparison semantics (1000+ test cases)
- ✅ Verify identical framework compatibility matrices
- ✅ Validate package structure (OPC compliance, nuspec parsing)
- ✅ Test round-trip compatibility (Go write → C# read → verify)
- ✅ Provide fast feedback (< 5 seconds for full suite)
- ✅ Support local development and CI environments

### Non-Goals

- ❌ Replace existing Go unit tests (complement, not replace)
- ❌ Test NuGet.Client implementation (trust it as ground truth)
- ❌ Test network operations (use local test packages)
- ❌ Performance benchmarking (functional correctness only)
- ❌ UI/CLI testing (API-level testing only)

---

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Interop Test Suite                          │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────────┐   ┌──────────────┐
│   C# Test    │◄──►│  Go CLI Bridge   │◄──►│  gonuget     │
│   Project    │    │  (JSON/Process)  │    │  Library     │
│  (xUnit)     │    └──────────────────┘    └──────────────┘
└──────────────┘              │
        │                     │
        │                     ▼
        │            ┌──────────────────┐
        │            │ Test Artifacts   │
        │            │ (packages, certs)│
        └───────────►└──────────────────┘
                              │
                              ▼
                     ┌──────────────────┐
                     │  NuGet.Client    │
                     │  (Validation)    │
                     └──────────────────┘
```

### Communication Flow

```
C# Test Request
    │
    ├─► 1. Serialize request to JSON
    │
    ├─► 2. Launch Go CLI process
    │
    ├─► 3. Write JSON to stdin
    │
    ├─► 4. Go processes request
    │       └─► Calls gonuget APIs
    │
    ├─► 5. Go returns JSON response via stdout
    │
    └─► 6. C# deserializes and validates
```

### Directory Structure

```
gonuget/
├── cmd/
│   └── nuget-interop-test/        # Go CLI bridge executable
│       ├── main.go                # Entry point
│       ├── handlers.go            # Request handlers
│       └── protocol.go            # JSON protocol definitions
│
├── tests/
│   └── nuget-client-interop/      # C# interop test suite
│       ├── GonugetInterop.Tests/  # Main test project
│       │   ├── GonugetInterop.Tests.csproj
│       │   ├── SignatureTests.cs           # Signature interop tests
│       │   ├── VersionTests.cs              # Version parsing tests
│       │   ├── FrameworkTests.cs            # Framework compat tests
│       │   ├── PackageReaderTests.cs        # Package structure tests
│       │   ├── RoundTripTests.cs            # Bidirectional tests
│       │   └── TestHelpers/
│       │       ├── GonugetBridge.cs         # Go CLI bridge
│       │       ├── TestCertificates.cs      # Test cert management
│       │       └── TestPackageFactory.cs    # Package generation
│       │
│       ├── TestData/               # Static test data
│       │   ├── certificates/       # Test certificates
│       │   └── packages/           # Pre-built test packages
│       │
│       ├── Makefile                # Build orchestration
│       └── README.md               # Setup instructions
│
└── docs/
    └── design/
        └── DESIGN-NUGET-CLIENT-INTEROP.md  # This document
```

---

## Component Design

### 1. Go CLI Bridge (`cmd/nuget-interop-test`)

**Purpose**: Expose gonuget functionality via JSON-RPC over stdin/stdout.

**Key Files**:
- `main.go` (~150 lines): Entry point, request routing
- `handlers.go` (~500 lines): Implementation of all handlers
- `protocol.go` (~200 lines): Request/response types
- `helpers.go` (~150 lines): Shared utilities

**Total Size**: ~1,000 lines

**Responsibilities**:
- Parse JSON requests from stdin
- Route to appropriate handlers
- Call gonuget APIs
- Serialize responses to stdout
- Handle errors gracefully

### 2. C# Test Project (`GonugetInterop.Tests`)

**Purpose**: Comprehensive interop tests using NuGet.Client as validator.

**Key Files**:
- `SignatureTests.cs` (~400 lines): Signature creation/verification
- `VersionTests.cs` (~300 lines): Version parsing/comparison
- `FrameworkTests.cs` (~300 lines): Framework compatibility
- `PackageReaderTests.cs` (~300 lines): Package structure validation
- `RoundTripTests.cs` (~200 lines): Bidirectional workflows

**Total Size**: ~1,500 lines (test code)

**Test Helpers** (~500 lines):
- `GonugetBridge.cs`: Process management and JSON communication
- `TestCertificates.cs`: Certificate generation and management
- `TestPackageFactory.cs`: Package creation utilities

**Total Project Size**: ~2,000 lines

---

## API Specifications

### Go CLI Bridge Protocol

All communication uses JSON over stdin/stdout.

#### Request Format

```json
{
  "action": "sign_package",
  "data": {
    "packageHash": "base64-encoded-hash",
    "certPath": "/path/to/cert.pfx",
    "certPassword": "optional-password",
    "keyPath": "/path/to/key.pem",
    "signatureType": "Author",
    "hashAlgorithm": "SHA256",
    "timestampURL": "http://timestamp.digicert.com"
  }
}
```

#### Response Format

```json
{
  "success": true,
  "data": {
    "signature": "base64-encoded-signature-bytes"
  },
  "error": null
}
```

Or on error:

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "SIGN_001",
    "message": "Failed to load certificate: file not found",
    "details": "/path/to/cert.pfx"
  }
}
```

### Supported Actions

| Action | Purpose | Input | Output |
|--------|---------|-------|--------|
| `sign_package` | Create PKCS#7 signature | Package hash, cert, options | Signature bytes |
| `parse_signature` | Parse signature structure | Signature bytes | Signature metadata |
| `verify_signature` | Verify signature validity | Signature + trust store | Verification result |
| `compare_versions` | Compare two versions | Version strings | Comparison result (-1/0/1) |
| `parse_version` | Parse version string | Version string | Version components |
| `check_framework_compat` | Check framework compatibility | Package FW, Project FW | Boolean + details |
| `parse_framework` | Parse framework identifier | Framework string | Framework components |
| `read_package` | Read package structure | Package bytes | Metadata + files list |
| `build_package` | Create package | Metadata + files | Package bytes |

---

## Implementation Details

### Part 1: Go CLI Bridge

#### File: `cmd/nuget-interop-test/main.go`

```go
// Package main implements a JSON-RPC bridge for gonuget interop testing.
// It receives requests via stdin and returns responses via stdout.
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Request represents an incoming test request
type Request struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

// Response represents the standard response format
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo contains detailed error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func main() {
	// Read request from stdin
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		sendError("REQ_001", "Failed to parse request", err.Error())
		os.Exit(1)
	}

	// Route to handler
	var handler Handler
	switch req.Action {
	case "sign_package":
		handler = &SignPackageHandler{}
	case "parse_signature":
		handler = &ParseSignatureHandler{}
	case "verify_signature":
		handler = &VerifySignatureHandler{}
	case "compare_versions":
		handler = &CompareVersionsHandler{}
	case "parse_version":
		handler = &ParseVersionHandler{}
	case "check_framework_compat":
		handler = &CheckFrameworkCompatHandler{}
	case "parse_framework":
		handler = &ParseFrameworkHandler{}
	case "read_package":
		handler = &ReadPackageHandler{}
	case "build_package":
		handler = &BuildPackageHandler{}
	default:
		sendError("ACT_001", "Unknown action", req.Action)
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

// Handler interface for request handlers
type Handler interface {
	Handle(data json.RawMessage) (interface{}, error)
	ErrorCode() string
}

func sendSuccess(data interface{}) {
	resp := Response{
		Success: true,
		Data:    data,
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}

func sendError(code, message, details string) {
	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}
```

#### File: `cmd/nuget-interop-test/protocol.go`

```go
package main

// SignPackageRequest represents a signature creation request
type SignPackageRequest struct {
	PackageHash    []byte `json:"packageHash"`    // Hash of package content
	CertPath       string `json:"certPath"`       // Path to certificate
	CertPassword   string `json:"certPassword"`   // Optional cert password
	KeyPath        string `json:"keyPath"`        // Path to private key
	SignatureType  string `json:"signatureType"`  // "Author" or "Repository"
	HashAlgorithm  string `json:"hashAlgorithm"`  // "SHA256", "SHA384", "SHA512"
	TimestampURL   string `json:"timestampURL"`   // Optional timestamp URL
}

// SignPackageResponse contains the created signature
type SignPackageResponse struct {
	Signature []byte `json:"signature"` // DER-encoded PKCS#7 signature
}

// ParseSignatureRequest asks gonuget to parse a signature
type ParseSignatureRequest struct {
	Signature []byte `json:"signature"` // DER-encoded PKCS#7 signature
}

// ParseSignatureResponse contains parsed signature metadata
type ParseSignatureResponse struct {
	Type              string   `json:"type"`              // "Author", "Repository", "Unknown"
	HashAlgorithm     string   `json:"hashAlgorithm"`     // "SHA256", etc.
	SignerCertHash    string   `json:"signerCertHash"`    // Hex-encoded cert hash
	TimestampCount    int      `json:"timestampCount"`    // Number of timestamps
	TimestampTimes    []string `json:"timestampTimes"`    // RFC3339 timestamp times
	Certificates      int      `json:"certificates"`      // Cert chain length
}

// VerifySignatureRequest validates a signature
type VerifySignatureRequest struct {
	Signature          []byte   `json:"signature"`          // Signature to verify
	TrustedRoots       [][]byte `json:"trustedRoots"`       // Trusted root certificates
	AllowUntrustedRoot bool     `json:"allowUntrustedRoot"` // Allow untrusted roots
	RequireTimestamp   bool     `json:"requireTimestamp"`   // Require timestamp
}

// VerifySignatureResponse contains verification results
type VerifySignatureResponse struct {
	Valid         bool     `json:"valid"`         // Overall validity
	Errors        []string `json:"errors"`        // Verification errors
	Warnings      []string `json:"warnings"`      // Verification warnings
	SignerSubject string   `json:"signerSubject"` // Signer cert subject DN
}

// CompareVersionsRequest compares two version strings
type CompareVersionsRequest struct {
	Version1 string `json:"version1"`
	Version2 string `json:"version2"`
}

// CompareVersionsResponse returns comparison result
type CompareVersionsResponse struct {
	Result int `json:"result"` // -1 (v1 < v2), 0 (equal), 1 (v1 > v2)
}

// ParseVersionRequest parses a version string
type ParseVersionRequest struct {
	Version string `json:"version"`
}

// ParseVersionResponse contains parsed version components
type ParseVersionResponse struct {
	Major         int    `json:"major"`
	Minor         int    `json:"minor"`
	Patch         int    `json:"patch"`
	Revision      int    `json:"revision"`
	Release       string `json:"release"`       // Pre-release label
	Metadata      string `json:"metadata"`      // Build metadata
	IsPrerelease  bool   `json:"isPrerelease"`
	IsLegacyForm  bool   `json:"isLegacyForm"`
}

// CheckFrameworkCompatRequest checks framework compatibility
type CheckFrameworkCompatRequest struct {
	PackageFramework string `json:"packageFramework"` // e.g., "net6.0"
	ProjectFramework string `json:"projectFramework"` // e.g., "net8.0"
}

// CheckFrameworkCompatResponse returns compatibility result
type CheckFrameworkCompatResponse struct {
	Compatible bool   `json:"compatible"`
	Reason     string `json:"reason"` // Explanation if not compatible
}

// ParseFrameworkRequest parses a framework identifier
type ParseFrameworkRequest struct {
	Framework string `json:"framework"`
}

// ParseFrameworkResponse contains parsed framework components
type ParseFrameworkResponse struct {
	Identifier string `json:"identifier"` // e.g., ".NETCoreApp"
	Version    string `json:"version"`    // e.g., "6.0"
	Profile    string `json:"profile"`    // Optional profile
	Platform   string `json:"platform"`   // Optional platform
}

// ReadPackageRequest reads package structure
type ReadPackageRequest struct {
	PackageBytes []byte `json:"packageBytes"` // ZIP package content
}

// ReadPackageResponse contains package metadata
type ReadPackageResponse struct {
	ID              string   `json:"id"`
	Version         string   `json:"version"`
	Authors         []string `json:"authors"`
	Description     string   `json:"description"`
	Dependencies    []string `json:"dependencies"`
	FileCount       int      `json:"fileCount"`
	HasSignature    bool     `json:"hasSignature"`
	SignatureType   string   `json:"signatureType"`
}

// BuildPackageRequest creates a package
type BuildPackageRequest struct {
	ID          string            `json:"id"`
	Version     string            `json:"version"`
	Authors     []string          `json:"authors"`
	Description string            `json:"description"`
	Files       map[string][]byte `json:"files"` // path -> content
}

// BuildPackageResponse contains the built package
type BuildPackageResponse struct {
	PackageBytes []byte `json:"packageBytes"` // ZIP package content
}
```

#### File: `cmd/nuget-interop-test/handlers.go` (Part 1: Signature Handlers)

```go
package main

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/willibrandon/gonuget/packaging/signatures"
)

// SignPackageHandler creates a package signature
type SignPackageHandler struct{}

func (h *SignPackageHandler) ErrorCode() string { return "SIGN_001" }

func (h *SignPackageHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req SignPackageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Load certificate
	cert, err := loadCertificate(req.CertPath, req.CertPassword)
	if err != nil {
		return nil, fmt.Errorf("load certificate: %w", err)
	}

	// Load private key
	key, err := loadPrivateKey(req.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	// Parse signature type
	var sigType signatures.SignatureType
	switch req.SignatureType {
	case "Author":
		sigType = signatures.SignatureTypeAuthor
	case "Repository":
		sigType = signatures.SignatureTypeRepository
	default:
		return nil, fmt.Errorf("invalid signature type: %s", req.SignatureType)
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
		return nil, fmt.Errorf("invalid hash algorithm: %s", req.HashAlgorithm)
	}

	// Create signing options
	opts := signatures.SigningOptions{
		Certificate:   cert,
		PrivateKey:    key,
		SignatureType: sigType,
		HashAlgorithm: hashAlg,
		TimestampURL:  req.TimestampURL,
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

// ParseSignatureHandler parses a signature structure
type ParseSignatureHandler struct{}

func (h *ParseSignatureHandler) ErrorCode() string { return "PARSE_001" }

func (h *ParseSignatureHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req ParseSignatureRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Parse signature
	sig, err := signatures.ReadSignature(req.Signature)
	if err != nil {
		return nil, fmt.Errorf("read signature: %w", err)
	}

	// Extract timestamp times
	var timestampTimes []string
	for _, ts := range sig.Timestamps {
		timestampTimes = append(timestampTimes, ts.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Compute signer cert hash
	signerCertHash := ""
	if sig.SignerCertificate != nil {
		signerCertHash = fmt.Sprintf("%x", sig.SignerCertificate.SubjectKeyId)
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

// VerifySignatureHandler verifies a signature
type VerifySignatureHandler struct{}

func (h *VerifySignatureHandler) ErrorCode() string { return "VERIFY_001" }

func (h *VerifySignatureHandler) Handle(data json.RawMessage) (interface{}, error) {
	var req VerifySignatureRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	// Parse signature
	sig, err := signatures.ReadSignature(req.Signature)
	if err != nil {
		return nil, fmt.Errorf("read signature: %w", err)
	}

	// Build trust store
	trustStore := signatures.NewTrustStore()
	for _, rootBytes := range req.TrustedRoots {
		cert, err := x509.ParseCertificate(rootBytes)
		if err != nil {
			return nil, fmt.Errorf("parse trusted root: %w", err)
		}
		trustStore.AddCertificate(cert)
	}

	// Create verification options
	opts := signatures.VerificationOptions{
		TrustStore:         trustStore,
		AllowUntrustedRoot: req.AllowUntrustedRoot,
		RequireTimestamp:   req.RequireTimestamp,
	}

	// Verify signature
	result, err := signatures.VerifySignature(sig, opts)

	var errors, warnings []string
	if err != nil {
		errors = append(errors, err.Error())
	}
	if result != nil {
		for _, warning := range result.Warnings {
			warnings = append(warnings, warning)
		}
	}

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

// Helper functions

func loadCertificate(path, password string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Try PEM format first
	block, _ := pem.Decode(data)
	if block != nil {
		return x509.ParseCertificate(block.Bytes)
	}

	// Try DER format
	return x509.ParseCertificate(data)
}

func loadPrivateKey(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	return x509.ParsePKCS8PrivateKey(block.Bytes)
}
```

**[Continued in Part 2...]**

---

## Test Specifications

### Test Categories

#### 1. Signature Interop Tests (`SignatureTests.cs`)

**Coverage**:
- Go creates Author signature → C# verifies with `PackageSignatureVerifier`
- Go creates Repository signature → C# verifies
- C# creates signature → Go parses and validates structure
- Timestamp validation (both directions)
- Certificate chain validation
- Hash algorithm variations (SHA256/384/512)

**Key Test Cases**:
```csharp
[Fact] // 40 tests total in this category
public void GonugetAuthorSignature_VerifiesWithNuGetClient()
public void GonugetRepositorySignature_VerifiesWithNuGetClient()
public void GonugetTimestampedSignature_HasValidTimestamp()
public void NuGetClientSignature_ParsesCorrectlyInGonuget()
public void SignatureHashAlgorithms_MatchBetweenImplementations()
```

#### 2. Version Interop Tests (`VersionTests.cs`)

**Coverage**:
- Version parsing (1000+ test cases from NuGet.Client test suite)
- Version comparison semantics
- SemVer 2.0 compliance
- Legacy version format support
- Prerelease version ordering

**Key Test Cases**:
```csharp
[Theory] // 80 tests total
[InlineData("1.0.0", "2.0.0", -1)]
[InlineData("2.0.0-beta", "2.0.0", -1)]
[InlineData("1.0.0+build", "1.0.0", 0)]
public void VersionComparison_MatchesNuGetSemantics(v1, v2, expected)

[Theory] // 100 tests
[InlineData("1.0.0")]
[InlineData("1.0.0-beta.1")]
public void VersionParsing_MatchesNuGetParsing(versionString)
```

#### 3. Framework Compatibility Tests (`FrameworkTests.cs`)

**Coverage**:
- .NET Framework/Standard/Core compatibility
- Platform-specific frameworks (Xamarin, Mono, etc.)
- Framework equivalence
- Portable framework profiles

**Key Test Cases**:
```csharp
[Theory] // 60 tests
[InlineData("net8.0", "net6.0", true)]
[InlineData("netstandard2.0", "net461", true)]
public void FrameworkCompatibility_MatchesNuGetLogic(pkg, proj, expected)
```

#### 4. Package Structure Tests (`PackageReaderTests.cs`)

**Coverage**:
- Package reading (nuspec, files, metadata)
- OPC compliance validation
- Signature file detection
- Content extraction

**Key Test Cases**:
```csharp
[Fact] // 30 tests
public void GonugetPackage_ReadsCorrectlyWithNuGetClient()
public void PackageHasCorrectOPCStructure()
public void NuspecMetadata_MatchesBetweenReaders()
```

#### 5. Round-Trip Tests (`RoundTripTests.cs`)

**Coverage**:
- Go write → C# read → compare
- C# write → Go read → compare
- Signature preservation
- Metadata preservation

**Key Test Cases**:
```csharp
[Fact] // 20 tests
public void RoundTrip_GonugetToNuGetClient_PreservesData()
public void RoundTrip_NuGetClientToGonuget_PreservesData()
```

### Test Data Requirements

**Certificates**:
- Self-signed test root CA
- Code signing certificate (RSA 2048-bit)
- Expired certificate (for negative tests)
- Weak key certificate (1024-bit, should fail)

**Packages**:
- Minimal package (ID + version only)
- Complex package (dependencies, multiple frameworks)
- Signed package (author signature)
- Countersigned package (repository signature)
- Legacy format package

---

## Build and CI Integration

### Makefile Targets

```makefile
# Build Go CLI bridge
.PHONY: build-interop
build-interop:
	@echo "Building gonuget interop CLI..."
	@go build -o tests/nuget-client-interop/gonuget-interop-test ./cmd/nuget-interop-test

# Run C# interop tests
.PHONY: test-interop
test-interop: build-interop
	@echo "Running NuGet.Client interop tests..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet test --verbosity normal

# Run all tests (Go + C# interop)
.PHONY: test-all
test-all: test test-interop

# Clean interop artifacts
.PHONY: clean-interop
clean-interop:
	@rm -f tests/nuget-client-interop/gonuget-interop-test
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet clean
```

### CI Pipeline Integration

```yaml
# .github/workflows/ci.yml (excerpt)
jobs:
  interop-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Setup .NET
        uses: actions/setup-dotnet@v3
        with:
          dotnet-version: '8.0'

      - name: Build Go interop CLI
        run: make build-interop

      - name: Run interop tests
        run: make test-interop

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: interop-test-results
          path: tests/nuget-client-interop/GonugetInterop.Tests/TestResults/
```

---

## Performance Considerations

### Expected Performance

| Test Category | Test Count | Target Time | Notes |
|---------------|-----------|-------------|-------|
| Signature Tests | 40 | < 2s | May include TSA calls |
| Version Tests | 180 | < 500ms | Pure computation |
| Framework Tests | 60 | < 300ms | Pure computation |
| Package Tests | 30 | < 1s | ZIP operations |
| Round-Trip Tests | 20 | < 1s | Full workflows |
| **TOTAL** | **330** | **< 5s** | Excluding TSA timeouts |

### Optimization Strategies

1. **Parallel Test Execution**: xUnit runs tests in parallel by default
2. **Test Data Caching**: Reuse certificates and test packages
3. **Process Pooling**: Keep Go CLI process warm between tests (future)
4. **Selective TSA Testing**: Only test timestamps in dedicated tests

---

## Security Considerations

### Certificate Management

- ✅ Use self-signed test certificates only
- ✅ Never commit private keys to repository
- ✅ Generate test certs at build time
- ✅ Use weak passwords for test certs (documented as test-only)

### Test Isolation

- ✅ Each test runs in isolated directory
- ✅ Cleanup test artifacts after completion
- ✅ No shared mutable state between tests

### Timestamp Authority Usage

- ✅ Use public TSAs for tests (DigiCert, etc.)
- ✅ Mark TSA tests as `[Trait("Category", "RequiresNetwork")]`
- ✅ Allow skipping TSA tests in offline environments

---

## Future Enhancements

### Phase 2 (Post-MVP)

1. **Performance Benchmarking**
   - Compare gonuget vs NuGet.Client performance
   - Identify optimization opportunities

2. **Fuzzing Integration**
   - Generate random valid/invalid packages
   - Feed to both implementations, compare results

3. **Package Repository Testing**
   - Test against real NuGet.org packages
   - Validate feed protocol compatibility

4. **Countersignature Testing**
   - Repository countersignatures
   - Multiple timestamp validation

### Phase 3 (Advanced)

1. **Cross-Platform Validation**
   - Test on Windows/Linux/macOS
   - Validate certificate store integration

2. **Offline Package Cache**
   - Build corpus of real-world packages
   - Regression test against entire corpus

3. **Automated Compatibility Reports**
   - Generate compatibility matrix
   - Track version-to-version changes

---

## Appendix A: File Size Budget

| Component | File | Lines | Notes |
|-----------|------|-------|-------|
| **Go CLI** | `main.go` | 150 | Entry point |
| | `protocol.go` | 200 | Request/response types |
| | `handlers.go` | 500 | All handlers |
| | `helpers.go` | 150 | Utilities |
| | **Subtotal** | **1,000** | |
| **C# Tests** | `SignatureTests.cs` | 400 | Signature interop |
| | `VersionTests.cs` | 300 | Version parsing |
| | `FrameworkTests.cs` | 300 | Framework compat |
| | `PackageReaderTests.cs` | 300 | Package structure |
| | `RoundTripTests.cs` | 200 | Bidirectional tests |
| | **Test Subtotal** | **1,500** | |
| **C# Helpers** | `GonugetBridge.cs` | 200 | Process bridge |
| | `TestCertificates.cs` | 150 | Cert management |
| | `TestPackageFactory.cs` | 150 | Package utilities |
| | **Helper Subtotal** | **500** | |
| **TOTAL** | | **3,000** | All code |

All files stay well under 1,500 lines as required.

---

## Appendix B: NuGet.Client API Reference

### Key APIs Used

```csharp
// Signature verification
using NuGet.Packaging.Signing;

var verifier = new PackageSignatureVerifier();
var settings = new SignedPackageVerifierSettings(...);
var result = await verifier.VerifySignaturesAsync(package, settings, token);

// Signature parsing
var signature = PrimarySignature.Load(signatureBytes);

// Package reading
using var reader = new PackageArchiveReader(stream);
var identity = reader.GetIdentity();
var nuspec = reader.NuspecReader;

// Version comparison
using NuGet.Versioning;

var v1 = NuGetVersion.Parse("1.0.0");
var v2 = NuGetVersion.Parse("2.0.0");
int result = v1.CompareTo(v2);

// Framework compatibility
using NuGet.Frameworks;

var reducer = new FrameworkReducer();
var compatible = reducer.IsCompatible(projectFw, packageFw);
```

---

## Appendix C: Implementation Checklist

### Phase 1: Foundation
- [ ] Create `cmd/nuget-interop-test/` directory structure
- [ ] Implement `main.go` with request routing
- [ ] Implement `protocol.go` with all request/response types
- [ ] Implement signature handlers in `handlers.go`
- [ ] Create `tests/nuget-client-interop/` directory structure
- [ ] Create C# test project with NuGet.Client dependencies
- [ ] Implement `GonugetBridge.cs` process communication

### Phase 2: Signature Tests
- [ ] Implement `SignatureTests.cs` (40 tests)
- [ ] Generate test certificates
- [ ] Test Go → C# signature validation
- [ ] Test C# → Go signature parsing
- [ ] Test timestamp validation

### Phase 3: Version & Framework Tests
- [ ] Implement version handlers in `handlers.go`
- [ ] Implement `VersionTests.cs` (180 tests)
- [ ] Implement framework handlers in `handlers.go`
- [ ] Implement `FrameworkTests.cs` (60 tests)

### Phase 4: Package Tests
- [ ] Implement package handlers in `handlers.go`
- [ ] Implement `PackageReaderTests.cs` (30 tests)
- [ ] Implement `RoundTripTests.cs` (20 tests)

### Phase 5: Polish & CI
- [ ] Add Makefile targets
- [ ] Integrate with GitHub Actions
- [ ] Add documentation
- [ ] Performance tuning
- [ ] Security review

---

**End of Design Document**
