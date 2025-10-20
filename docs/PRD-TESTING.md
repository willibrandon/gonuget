# gonuget - Product Requirements Document: Testing

**Version:** 1.0
**Status:** Draft
**Last Updated:** 2025-10-19
**Owner:** Engineering / QA

---

## Table of Contents

1. [Overview](#overview)
2. [Testing Strategy](#testing-strategy)
3. [Unit Testing](#unit-testing)
4. [Integration Testing](#integration-testing)
5. [Compatibility Testing](#compatibility-testing)
6. [Performance Testing](#performance-testing)
7. [Security Testing](#security-testing)
8. [Test Coverage](#test-coverage)
9. [Acceptance Criteria](#acceptance-criteria)

---

## Overview

This document specifies testing requirements to ensure gonuget achieves production-grade quality and 100% compatibility with the C# NuGet.Client.

**Quality Goals:**
- 90%+ code coverage
- 100% compatibility with C# NuGet.Client behavior
- Zero race conditions
- Zero memory leaks
- No panics in library code

**Related Design Documents:**
- DESIGN-TESTING.md - Testing strategy and approaches

---

## Testing Strategy

### Requirement TS-001: Test Pyramid

**Priority:** P0 (Critical)
**Component:** All packages

**Description:**
Follow test pyramid with appropriate distribution of test types.

**Test Distribution:**

1. **Unit tests (70%):**
   - Fast (<1ms per test)
   - Isolated components
   - No external dependencies
   - Mock all I/O

2. **Integration tests (25%):**
   - Real HTTP calls
   - Real file I/O
   - Docker containers for services
   - Slower (<1s per test)

3. **End-to-end tests (5%):**
   - Complete workflows
   - Real NuGet feeds
   - Full stack validation
   - Slowest (<5s per test)

**Acceptance Criteria:**
- ✅ Test distribution follows pyramid
- ✅ Unit tests run fast (<100ms total)
- ✅ Integration tests isolated
- ✅ E2E tests cover critical paths

---

### Requirement TS-002: Test Organization

**Priority:** P0 (Critical)
**Component:** All packages

**Description:**
Organize tests for discoverability and maintenance.

**Directory Structure:**
```
gonuget/
  version/
    version.go
    version_test.go          # Unit tests
    version_integration_test.go  # Integration tests (if needed)
  protocol/v3/
    search.go
    search_test.go
    testdata/                # Test fixtures
      service_index.json
      search_response.json
```

**Test File Naming:**
- `*_test.go` - Unit tests
- `*_integration_test.go` - Integration tests
- `*_benchmark_test.go` - Benchmarks
- `*_fuzz_test.go` - Fuzz tests

**Build Tags:**
- `//go:build integration` - Integration tests (skip by default)
- `//go:build e2e` - E2E tests (skip by default)

**Acceptance Criteria:**
- ✅ Test files properly named
- ✅ Build tags used correctly
- ✅ testdata organized

---

## Unit Testing

### Requirement UT-001: Version Package Tests

**Priority:** P0 (Critical)
**Component:** `version` package

**Description:**
Comprehensive unit tests for version parsing and comparison.

**Test Coverage:**

1. **Parsing:**
   - Valid SemVer 2.0 versions
   - Legacy 4-part versions
   - Invalid formats (error handling)
   - Edge cases (leading zeros, long labels)

2. **Comparison:**
   - Numeric component comparison
   - Prerelease comparison
   - Legacy vs SemVer comparison
   - Nil handling

3. **Version ranges:**
   - All bracket combinations
   - Open-ended ranges
   - Empty ranges
   - Satisfies() logic

4. **Floating versions:**
   - All float patterns
   - Resolution logic

**Test Count Target:** 200+ tests

**Example Test:**
```go
func TestVersionCompare(t *testing.T) {
    tests := []struct{
        v1 string
        v2 string
        expected int // -1, 0, 1
    }{
        {"1.0.0", "2.0.0", -1},
        {"1.0.0", "1.0.0", 0},
        {"2.0.0", "1.0.0", 1},
        {"1.0.0-beta", "1.0.0", -1},
        // ... more cases
    }

    for _, tt := range tests {
        v1 := MustParse(tt.v1)
        v2 := MustParse(tt.v2)
        got := v1.Compare(v2)
        if got != tt.expected {
            t.Errorf("Compare(%s, %s) = %d, want %d", tt.v1, tt.v2, got, tt.expected)
        }
    }
}
```

**Acceptance Criteria:**
- ✅ 200+ test cases
- ✅ 100% branch coverage
- ✅ All edge cases covered
- ✅ Cross-validated against C# NuGet.Versioning

---

### Requirement UT-002: Framework Package Tests

**Priority:** P0 (Critical)
**Component:** `frameworks` package

**Description:**
Comprehensive tests for framework parsing and compatibility.

**Test Coverage:**

1. **TFM parsing:**
   - Modern TFMs (net5.0+)
   - Legacy TFMs (net45, netstandard1.x)
   - Platform-specific TFMs
   - PCL profiles
   - Invalid formats

2. **Compatibility:**
   - .NET Framework compat rules
   - .NET Core compat rules
   - .NET 5+ compat rules
   - .NET Standard compat rules
   - Platform-specific compat

3. **Nearest framework selection:**
   - All selection scenarios
   - Edge cases (no compatible framework)

**Test Count Target:** 500+ tests

**Compatibility Test Matrix:**

| Target | Assets Available | Expected Selection |
|--------|-----------------|-------------------|
| net8.0 | net8.0, net6.0, netstandard2.1 | net8.0 |
| net8.0 | net6.0, netstandard2.1 | net6.0 |
| net48 | net48, net45, netstandard2.0 | net48 |
| net48 | netstandard2.1 | (no compatible) |

**Acceptance Criteria:**
- ✅ 500+ test cases
- ✅ All compatibility rules tested
- ✅ Mapping data validated
- ✅ Cross-validated against C# NuGet.Frameworks

---

### Requirement UT-003: Protocol Package Tests

**Priority:** P0 (Critical)
**Component:** `protocol/v3`, `protocol/v2` packages

**Description:**
Unit tests for protocol parsing and request building.

**Test Coverage:**

1. **Service index:**
   - Valid JSON parsing
   - Resource discovery
   - Invalid JSON handling

2. **Search response:**
   - Parse search results
   - Flexible string/array handling
   - Missing fields handling

3. **Metadata response:**
   - Registration index parsing
   - Dependency group parsing
   - Catalog entry parsing

4. **Request building:**
   - URL construction
   - Query parameter encoding
   - Header injection

**Test Data:**
- Use real service index from nuget.org (saved in testdata/)
- Use real search responses (sanitized)
- Use synthetic data for edge cases

**Acceptance Criteria:**
- ✅ 100+ test cases
- ✅ Real JSON fixtures used
- ✅ All parsing paths tested
- ✅ Error cases covered

---

### Requirement UT-004: Packaging Package Tests

**Priority:** P0 (Critical)
**Component:** `packaging` package

**Description:**
Unit tests for package reading and creation.

**Test Coverage:**

1. **Package reading:**
   - Open valid .nupkg files
   - Parse .nuspec
   - List package contents
   - Extract files
   - Invalid package handling

2. **Nuspec parsing:**
   - All metadata fields
   - Dependency groups
   - Framework assemblies
   - Invalid XML handling

3. **Package creation:**
   - Build valid packages
   - Generate .nuspec
   - OPC structure
   - Validation before build

4. **Asset selection:**
   - Framework-based selection
   - RID-specific selection
   - Nearest framework logic

**Test Packages:**
- Create minimal test .nupkg files
- Use real packages (e.g., Newtonsoft.Json) in testdata/
- Create malformed packages for error testing

**Acceptance Criteria:**
- ✅ 100+ test cases
- ✅ Real .nupkg files tested
- ✅ Package creation validated
- ✅ Asset selection accurate

---

### Requirement UT-005: Mock Testing

**Priority:** P0 (Critical)
**Component:** All packages

**Description:**
Use mocks to isolate units under test.

**Mockable Interfaces:**

1. **HTTP client:**
   ```go
   type HTTPClient interface {
       Do(req *http.Request) (*http.Response, error)
   }

   type MockHTTPClient struct {
       Responses map[string]*http.Response
       Errors map[string]error
   }
   ```

2. **Cache:**
   ```go
   type MockCache struct {
       data map[string][]byte
   }
   ```

3. **Resource providers:**
   ```go
   type MockResourceProvider struct {
       resources map[ResourceType]Resource
   }
   ```

**Mocking Strategy:**
- Use interfaces for dependencies
- Inject mocks via constructors
- Avoid global state

**Acceptance Criteria:**
- ✅ All external dependencies mockable
- ✅ Mocks used in unit tests
- ✅ No real I/O in unit tests

---

## Integration Testing

### Requirement IT-001: HTTP Integration Tests

**Priority:** P1 (High)
**Component:** `protocol/v3`, `protocol/v2` packages

**Description:**
Integration tests with real HTTP calls.

**Test Setup:**

1. **Test server:**
   - Spin up local HTTP server
   - Serve canned responses
   - Simulate error conditions

2. **Test cases:**
   - Service index fetch
   - Package search
   - Metadata fetch
   - Package download
   - Error responses (404, 500, etc.)
   - Timeout simulation
   - Retry logic

**Example:**
```go
func TestSearchIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Start test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Serve canned search response
        w.Header().Set("Content-Type", "application/json")
        w.Write(loadTestData("search_response.json"))
    }))
    defer server.Close()

    // Test search
    client := NewClient(WithSource(server.URL))
    results, err := client.Search(context.Background(), "Newtonsoft")
    if err != nil {
        t.Fatalf("Search failed: %v", err)
    }

    if len(results) == 0 {
        t.Error("Expected search results")
    }
}
```

**Acceptance Criteria:**
- ✅ Local test server used
- ✅ Real HTTP calls made
- ✅ Error scenarios tested
- ✅ Retry logic validated

---

### Requirement IT-002: Real Feed Integration Tests

**Priority:** P1 (High)
**Component:** All packages

**Description:**
Integration tests against real NuGet feeds.

**Test Feeds:**

1. **nuget.org (v3):**
   - Search for well-known packages
   - Fetch metadata
   - Download packages

2. **Legacy v2 feed:**
   - Test v2 protocol implementation
   - Verify OData parsing

**Test Cases:**
```go
func TestRealNuGetOrg(t *testing.T) {
    if os.Getenv("NUGET_INTEGRATION_TESTS") == "" {
        t.Skip("Set NUGET_INTEGRATION_TESTS=1 to run")
    }

    client := NewClient(WithSource("https://api.nuget.org/v3/index.json"))

    // Search for Newtonsoft.Json
    results, err := client.Search(context.Background(), "Newtonsoft.Json")
    if err != nil {
        t.Fatalf("Search failed: %v", err)
    }

    // Verify we found the package
    found := false
    for _, r := range results {
        if r.Identity.ID == "Newtonsoft.Json" {
            found = true
            break
        }
    }
    if !found {
        t.Error("Newtonsoft.Json not found in search results")
    }
}
```

**Rate Limiting:**
- Use rate limiter to avoid overwhelming feeds
- Cache results where possible
- Run sparingly (nightly builds, not on every commit)

**Acceptance Criteria:**
- ✅ Tests against nuget.org work
- ✅ Tests against v2 feeds work
- ✅ Rate limiting respected
- ✅ Gated behind environment variable

---

### Requirement IT-003: Cache Integration Tests

**Priority:** P1 (High)
**Component:** `cache` package

**Description:**
Integration tests for disk cache persistence.

**Test Cases:**

1. **Persistence:**
   - Write to cache
   - Restart process (or close/reopen)
   - Read from cache (should hit)

2. **Eviction:**
   - Fill cache to capacity
   - Add more entries
   - Verify LRU eviction

3. **TTL:**
   - Write with short TTL
   - Wait for expiry
   - Verify cache miss

**Acceptance Criteria:**
- ✅ Disk persistence works
- ✅ Eviction correct
- ✅ TTL honored

---

## Compatibility Testing

### Requirement CT-001: Cross-Validation Against C#

**Priority:** P0 (Critical)
**Component:** All packages

**Description:**
Validate gonuget behavior matches C# NuGet.Client.

**Validation Strategy:**

1. **Generate test cases from C#:**
   - Write C# program to generate test vectors
   - Version comparison results
   - Framework compatibility results
   - Dependency resolution results

2. **Run gonuget against test vectors:**
   - Parse test cases
   - Execute gonuget functions
   - Compare results to C# outputs

3. **Diff any mismatches:**
   - Report discrepancies
   - Fix gonuget implementation
   - Rerun until 100% match

**Test Vector Format (JSON):**
```json
{
  "version_comparison": [
    {"v1": "1.0.0", "v2": "2.0.0", "expected": -1},
    {"v1": "1.0.0-beta", "v2": "1.0.0", "expected": -1}
  ],
  "framework_compatibility": [
    {"target": "net8.0", "package": "netstandard2.1", "compatible": true},
    {"target": "net48", "package": "netstandard2.1", "compatible": false}
  ]
}
```

**Test Vector Generation:**
```csharp
using NuGet.Versioning;
using NuGet.Frameworks;

var tests = new {
    version_comparison = new[] {
        new {
            v1 = "1.0.0",
            v2 = "2.0.0",
            expected = NuGetVersion.Parse("1.0.0").CompareTo(NuGetVersion.Parse("2.0.0"))
        },
        // ... more cases
    }
};

File.WriteAllText("test_vectors.json", JsonSerializer.Serialize(tests));
```

**Acceptance Criteria:**
- ✅ Test vector generator created
- ✅ 1000+ test vectors generated
- ✅ 100% match on version comparison
- ✅ 100% match on framework compatibility
- ✅ 100% match on dependency resolution

---

### Requirement CT-002: Package Compatibility

**Priority:** P0 (Critical)
**Component:** `packaging` package

**Description:**
Verify gonuget can read packages created by NuGet.exe and vice versa.

**Test Cases:**

1. **Read packages created by NuGet.exe:**
   - Download real packages from nuget.org
   - Open with gonuget
   - Verify metadata parsed correctly
   - Compare to NuGet.exe output

2. **Create packages readable by NuGet.exe:**
   - Build package with gonuget
   - Install with NuGet.exe
   - Verify installation succeeds

3. **Round-trip:**
   - Read package with gonuget
   - Modify metadata
   - Rebuild package
   - Verify readable by NuGet.exe

**Acceptance Criteria:**
- ✅ Reads all real packages correctly
- ✅ Created packages installable by NuGet.exe
- ✅ Round-trip succeeds

---

## Performance Testing

### Requirement PT-001: Benchmark Suite

**Priority:** P0 (Critical)
**Component:** All packages

**Description:**
Benchmark critical operations to ensure performance goals.

**Benchmarks:**

1. **Version operations:**
   ```go
   func BenchmarkVersionParse(b *testing.B) {
       for i := 0; i < b.N; i++ {
           _, _ = Parse("1.2.3-beta.1+build.123")
       }
   }

   func BenchmarkVersionCompare(b *testing.B) {
       v1 := MustParse("1.0.0")
       v2 := MustParse("2.0.0")
       b.ResetTimer()
       for i := 0; i < b.N; i++ {
           _ = v1.Compare(v2)
       }
   }
   ```

2. **Framework operations:**
   ```go
   func BenchmarkFrameworkParse(b *testing.B) {
       for i := 0; i < b.N; i++ {
           _, _ = ParseFramework("net8.0")
       }
   }

   func BenchmarkFrameworkCompatibility(b *testing.B) {
       target := MustParseFramework("net8.0")
       pkg := MustParseFramework("netstandard2.1")
       b.ResetTimer()
       for i := 0; i < b.N; i++ {
           _ = target.IsCompatible(pkg)
       }
   }
   ```

3. **Protocol operations:**
   ```go
   func BenchmarkParseServiceIndex(b *testing.B) {
       data := loadTestData("service_index.json")
       b.ResetTimer()
       for i := 0; i < b.N; i++ {
           _, _ = ParseServiceIndex(data)
       }
   }
   ```

**Performance Targets:**

| Operation | Target | Allocations |
|-----------|--------|-------------|
| Version.Parse | <50ns/op | ≤1 alloc |
| Version.Compare | <20ns/op | 0 allocs |
| Framework.Parse | <100ns/op | ≤1 alloc |
| Framework.IsCompatible | <15ns/op | 0 allocs |
| ServiceIndex.Parse | <5ms | Minimal |

**Allocation Benchmarks:**
```go
func BenchmarkVersionParseAllocs(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        _, _ = Parse("1.2.3")
    }
}
```

**Acceptance Criteria:**
- ✅ All targets met
- ✅ Zero allocation hot paths verified
- ✅ Regression tests for performance
- ✅ Benchmarks run in CI

---

### Requirement PT-002: Load Testing

**Priority:** P1 (High)
**Component:** All packages

**Description:**
Load test to verify throughput and resource usage.

**Scenarios:**

1. **Concurrent searches:**
   - 100 concurrent goroutines
   - Each performs 100 searches
   - Measure throughput (searches/sec)
   - Monitor memory usage

2. **Concurrent downloads:**
   - 50 concurrent package downloads
   - 10MB packages
   - Measure throughput (MB/sec)
   - Verify no file descriptor leaks

3. **Long-running operation:**
   - Resolve dependencies for 1000 packages
   - Monitor memory over time
   - Verify no memory leaks

**Acceptance Criteria:**
- ✅ Supports 100 concurrent operations
- ✅ No memory leaks
- ✅ No file descriptor leaks
- ✅ Throughput meets targets

---

## Security Testing

### Requirement ST-001: Vulnerability Scanning

**Priority:** P0 (Critical)
**Component:** All packages

**Description:**
Scan for vulnerabilities in code and dependencies.

**Tools:**

1. **go vet:**
   - Run on every build
   - Catch common mistakes

2. **staticcheck:**
   - Advanced static analysis
   - Detect bugs and inefficiencies

3. **gosec:**
   - Security-focused linter
   - Detect security issues

4. **Dependabot:**
   - Monitor dependency vulnerabilities
   - Auto-create PRs for updates

**CI Integration:**
```yaml
- name: Security scan
  run: |
    go vet ./...
    staticcheck ./...
    gosec ./...
```

**Acceptance Criteria:**
- ✅ All tools pass
- ✅ No high/critical findings
- ✅ Dependency scanning enabled

---

### Requirement ST-002: Fuzzing

**Priority:** P1 (High)
**Component:** Parsers (version, framework, nuspec)

**Description:**
Fuzz test parsers for crashes and panics.

**Fuzz Tests:**

1. **Version parsing:**
   ```go
   func FuzzVersionParse(f *testing.F) {
       // Seed corpus
       f.Add("1.0.0")
       f.Add("1.0.0-beta")
       f.Add("invalid")

       f.Fuzz(func(t *testing.T, input string) {
           // Should never panic
           _, _ = Parse(input)
       })
   }
   ```

2. **Framework parsing:**
   ```go
   func FuzzFrameworkParse(f *testing.F) {
       f.Add("net8.0")
       f.Add("netstandard2.1")
       f.Add("invalid")

       f.Fuzz(func(t *testing.T, input string) {
           _, _ = ParseFramework(input)
       })
   }
   ```

3. **Nuspec parsing:**
   ```go
   func FuzzNuspecParse(f *testing.F) {
       f.Add([]byte(`<package>...</package>`))

       f.Fuzz(func(t *testing.T, data []byte) {
           _, _ = ParseNuspec(bytes.NewReader(data))
       })
   }
   ```

**Fuzzing Duration:**
- Continuous fuzzing in CI (1 minute)
- Extended fuzzing locally (1 hour+)
- Dedicated fuzzing infrastructure (optional)

**Acceptance Criteria:**
- ✅ No crashes in 100K iterations
- ✅ No panics in 100K iterations
- ✅ Parsers handle arbitrary input

---

## Test Coverage

### Requirement COV-001: Code Coverage Tracking

**Priority:** P0 (Critical)
**Component:** All packages

**Description:**
Track and enforce code coverage targets.

**Coverage Targets:**

| Package | Target | Current |
|---------|--------|---------|
| version | 95% | TBD |
| frameworks | 95% | TBD |
| protocol/v3 | 90% | TBD |
| protocol/v2 | 85% | TBD |
| packaging | 90% | TBD |
| cache | 90% | TBD |
| **Overall** | **90%** | **TBD** |

**Coverage Tracking:**
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**CI Enforcement:**
- Fail build if coverage drops below 90%
- Report coverage in PR comments
- Track coverage trends over time

**Acceptance Criteria:**
- ✅ 90%+ overall coverage
- ✅ Critical packages ≥95%
- ✅ Coverage tracked in CI
- ✅ Trends monitored

---

### Requirement COV-002: Branch Coverage

**Priority:** P1 (High)
**Component:** All packages

**Description:**
Ensure all code paths exercised.

**Focus Areas:**

1. **Error paths:**
   - All error returns tested
   - Edge cases covered

2. **Conditional logic:**
   - All if/else branches
   - All switch cases

3. **Loops:**
   - Empty loop iterations
   - Single iteration
   - Multiple iterations

**Acceptance Criteria:**
- ✅ All branches covered
- ✅ Error paths tested
- ✅ Edge cases included

---

## Acceptance Criteria

### Overall Testing

**Test Count:**
- ✅ 1000+ unit tests
- ✅ 100+ integration tests
- ✅ 20+ E2E tests
- ✅ 50+ benchmarks

**Quality:**
- ✅ 90%+ code coverage
- ✅ 100% compatibility with C# NuGet.Client
- ✅ Zero race conditions
- ✅ Zero memory leaks
- ✅ No panics in library code

**Performance:**
- ✅ All benchmark targets met
- ✅ Load tests pass
- ✅ No performance regressions

**Security:**
- ✅ All security scans pass
- ✅ No high/critical vulnerabilities
- ✅ Fuzz testing clean

**CI/CD:**
- ✅ Tests run on every PR
- ✅ Coverage reported
- ✅ Performance benchmarks tracked
- ✅ Security scans automated

---

## Related Documents

- PRD-OVERVIEW.md - Product vision and goals
- PRD-CORE.md - Core library requirements
- PRD-PROTOCOL.md - Protocol implementation
- PRD-PACKAGING.md - Package operations
- PRD-INFRASTRUCTURE.md - HTTP, caching, observability
- PRD-RELEASE.md - Release criteria

---

**END OF PRD-TESTING.md**
