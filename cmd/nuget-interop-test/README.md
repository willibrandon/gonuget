# nuget-interop-test

A JSON-RPC bridge that enables cross-language interoperability testing between gonuget (Go) and NuGet.Client (C#).

## Overview

This executable provides a command-line interface for invoking gonuget functionality via JSON-RPC over stdin/stdout. It acts as a bridge to validate that gonuget's implementation matches the official NuGet.Client behavior across all major subsystems.

## Architecture

The bridge uses a simple JSON-RPC protocol:

**Request Format:**
```json
{
  "action": "action_name",
  "data": { /* action-specific parameters */ }
}
```

**Response Format:**
```json
{
  "success": true,
  "data": { /* action-specific results */ }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "error_code",
    "message": "Human-readable message",
    "details": "Additional context (optional)"
  }
}
```

## Supported Actions

The bridge exposes 15 actions across 5 functional categories:

### Signature Operations
- **`sign_package`** - Create PKCS#7 package signatures
  - Inputs: packageHash, certPath, certPassword (optional), keyPath (optional), signatureType, hashAlgorithm, timestampURL (optional)
  - Output: signature (base64-encoded PKCS#7)

- **`parse_signature`** - Parse signature metadata
  - Input: signature (base64-encoded)
  - Output: signerSubject, signerIssuer, signatureType, hashAlgorithm, signingTime, timestampTime (optional)

- **`verify_signature`** - Verify signature validity
  - Inputs: signature, trustedRoots (optional), allowUntrustedRoot, requireTimestamp
  - Output: valid, signerSubject, errors[], warnings[]

### Version Operations
- **`compare_versions`** - Compare two NuGet version strings
  - Inputs: version1, version2
  - Output: result (-1, 0, or 1)

- **`parse_version`** - Parse version string into components
  - Input: version
  - Output: major, minor, patch, revision, release, metadata, isPrerelease

### Framework Operations
- **`check_framework_compat`** - Check framework compatibility
  - Inputs: packageFramework, projectFramework
  - Output: compatible (boolean)

- **`parse_framework`** - Parse framework identifier
  - Input: framework
  - Output: identifier, version, profile, platform

### Package Operations
- **`read_package`** - Read .nupkg metadata and structure
  - Input: packageBytes (base64-encoded ZIP)
  - Output: id, version, authors[], description, dependencies[], files[], isSigned, signatureType (optional)

- **`build_package`** - Build minimal .nupkg from metadata
  - Inputs: id, version, authors[], description, files (map of path -> base64 content)
  - Output: packageBytes (base64-encoded ZIP)

### Asset Selection Operations
- **`find_runtime_assemblies`** - Find runtime assemblies for target framework
  - Inputs: paths[], targetFramework (optional)
  - Output: items[] with path and properties (tfm, assembly, rid, etc.)

- **`find_compile_assemblies`** - Find compile reference assemblies
  - Inputs: paths[], targetFramework (optional)
  - Output: items[] with path and properties

- **`parse_asset_path`** - Parse single asset path
  - Input: path
  - Output: properties map (tfm, assembly, rid, locale, etc.)

### RID Resolution Operations
- **`expand_runtime`** - Expand RID to compatibility chain
  - Input: rid
  - Output: expandedRuntimes[] (nearest first)

- **`are_runtimes_compatible`** - Check RID compatibility
  - Inputs: targetRid, packageRid
  - Output: compatible (boolean)

## Building

### Using Make (Recommended)

```bash
# From repository root
make build-interop

# Or build everything (Go + interop + .NET tests)
make build
```

### Manual Build

```bash
# From repository root
go build -o gonuget-interop-test ./cmd/nuget-interop-test

# Or from cmd/nuget-interop-test directory
cd cmd/nuget-interop-test
go build -o ../../gonuget-interop-test .
```

## Usage

The bridge is designed to be invoked by automated tests, not used interactively. However, for manual testing:

```bash
# Echo JSON request to stdin
echo '{"action":"parse_version","data":{"version":"1.2.3-beta.1"}}' | ./gonuget-interop-test

# Output:
# {"success":true,"data":{"major":1,"minor":2,"patch":3,"revision":0,"release":"beta.1","metadata":"","isPrerelease":true}}
```

## Integration with C# Tests

The C# test suite (`tests/nuget-client-interop`) uses `GonugetBridge.cs` to invoke this executable via `System.Diagnostics.Process`:

```csharp
// C# test helper spawns the bridge process
var result = GonugetBridge.ParseVersion("1.2.3-beta.1");
Assert.Equal(1, result.Major);
Assert.Equal("beta.1", result.Release);
```

The bridge automatically finds the executable in:
1. Test output directory (`bin/Debug/net9.0/`)
2. Repository root (for local development)

## Error Handling

All errors are returned as structured JSON with error codes:

| Error Code | Meaning |
|------------|---------|
| `invalid_action` | Unknown action name |
| `invalid_input` | Malformed request data |
| `parse_error` | Failed to parse input (version, framework, etc.) |
| `file_not_found` | Certificate or package file not found |
| `signature_error` | Signature creation/verification failed |
| `package_error` | Package read/build failed |

Example error response:
```json
{
  "success": false,
  "error": {
    "code": "parse_error",
    "message": "invalid version string",
    "details": "version must follow semver format"
  }
}
```

## Implementation Notes

- **Handler Pattern**: Each action is implemented by a handler that implements the `Handler` interface
- **JSON Serialization**: Uses Go's `encoding/json` with camelCase property names
- **Base64 Encoding**: Binary data (signatures, packages, certificates) is base64-encoded in JSON
- **Process Isolation**: Each request spawns a new process (C# side controls lifecycle)
- **Timeout**: C# bridge enforces 30-second timeout per request

## Related Components

- **C# Test Suite**: `tests/nuget-client-interop/GonugetInterop.Tests/`
- **Bridge Client**: `tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs`
- **Test Coverage**: 327 tests across 8 test classes validating all 15 actions
