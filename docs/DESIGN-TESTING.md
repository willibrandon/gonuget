# gonuget Testing Design

**Component**: All packages + `internal/testutil/`
**Version**: 1.0.0
**Status**: Draft

---

## Table of Contents

1. [Overview](#overview)
2. [Testing Philosophy](#testing-philosophy)
3. [Test Organization](#test-organization)
4. [Unit Testing](#unit-testing)
5. [Integration Testing](#integration-testing)
6. [Benchmark Testing](#benchmark-testing)
7. [Test Utilities](#test-utilities)
8. [Golden File Testing](#golden-file-testing)
9. [Coverage Targets](#coverage-targets)
10. [CI/CD Integration](#cicd-integration)

---

## Overview

Comprehensive testing is critical for gonuget's reliability. The testing strategy ensures:

- **Correctness**: All features work as specified
- **Compatibility**: Behavior matches C# NuGet.Client
- **Performance**: Meets or exceeds performance targets
- **Regression prevention**: Tests catch breaking changes

---

## Testing Philosophy

### Testing Pyramid

```
        ┌─────────────┐
        │   E2E Tests │  5% - Full workflow tests
        │   (10-20)   │
        └─────────────┘
       ┌───────────────┐
       │Integration Tests│  15% - Component integration
       │    (100-200)   │
       └───────────────┘
      ┌──────────────────┐
      │   Unit Tests     │  80% - Individual functions
      │   (1000-2000)    │
      └──────────────────┘
```

### Principles

1. **Fast feedback**: Unit tests run in <1s, full suite in <30s
2. **Deterministic**: No flaky tests, no test order dependencies
3. **Isolated**: Tests don't affect each other
4. **Maintainable**: Clear test names, minimal duplication
5. **Comprehensive**: Cover happy paths, edge cases, errors

---

## Test Organization

### Directory Structure

```
gonuget/
├── pkg/gonuget/
│   ├── version/
│   │   ├── version.go
│   │   ├── version_test.go         # Unit tests
│   │   └── version_bench_test.go   # Benchmarks
│   ├── framework/
│   │   ├── framework.go
│   │   ├── framework_test.go
│   │   └── compat_test.go
│   └── http/
│       ├── client.go
│       ├── client_test.go
│       └── integration_test.go     # +build integration
│
├── internal/testutil/
│   ├── mock.go                     # Mock HTTP client
│   ├── fixtures.go                 # Test fixtures
│   ├── assert.go                   # Custom assertions
│   └── server.go                   # Test HTTP server
│
├── testdata/
│   ├── fixtures/
│   │   ├── service-index.json      # Real API responses
│   │   ├── search-newtonsoft.json
│   │   └── metadata-serilog.json
│   ├── packages/
│   │   ├── test-package-1.0.0.nupkg
│   │   └── signed-package-2.0.0.nupkg
│   └── certificates/
│       ├── test-cert.pem
│       └── trusted-certs.pem
│
└── tests/
    ├── integration/                # Integration tests
    │   ├── search_test.go
    │   ├── download_test.go
    │   └── install_test.go
    └── e2e/                        # End-to-end tests
        └── workflow_test.go
```

### Build Tags

```go
// Unit tests (default)
// No build tag required

// Integration tests (require network)
// +build integration

// End-to-end tests (slow, require full setup)
// +build e2e
```

---

## Unit Testing

### Test Naming Convention

```go
// Format: Test<Function>_<Scenario>_<ExpectedResult>

func TestParse_ValidVersion_Success(t *testing.T)
func TestParse_InvalidVersion_ReturnsError(t *testing.T)
func TestCompare_EqualVersions_ReturnsZero(t *testing.T)
func TestCompare_GreaterVersion_ReturnsPositive(t *testing.T)
```

### Table-Driven Tests

**File**: `pkg/gonuget/version/version_test.go`

```go
package version

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestParse_ValidVersions(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected *NuGetVersion
    }{
        {
            name:  "simple version",
            input: "1.0.0",
            expected: &NuGetVersion{
                Major: 1,
                Minor: 0,
                Patch: 0,
            },
        },
        {
            name:  "version with prerelease",
            input: "1.0.0-beta",
            expected: &NuGetVersion{
                Major:         1,
                Minor:         0,
                Patch:         0,
                ReleaseLabels: []string{"beta"},
            },
        },
        {
            name:  "version with metadata",
            input: "1.0.0+build.123",
            expected: &NuGetVersion{
                Major:    1,
                Minor:    0,
                Patch:    0,
                Metadata: "build.123",
            },
        },
        {
            name:  "4-part legacy version",
            input: "1.2.3.4",
            expected: &NuGetVersion{
                Major:           1,
                Minor:           2,
                Patch:           3,
                Revision:        4,
                IsLegacyVersion: true,
            },
        },
        {
            name:  "version with leading zeros",
            input: "1.01.1",
            expected: &NuGetVersion{
                Major: 1,
                Minor: 1,
                Patch: 1,
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Parse(tt.input)
            require.NoError(t, err)
            assert.Equal(t, tt.expected.Major, result.Major)
            assert.Equal(t, tt.expected.Minor, result.Minor)
            assert.Equal(t, tt.expected.Patch, result.Patch)
            assert.Equal(t, tt.expected.ReleaseLabels, result.ReleaseLabels)
            assert.Equal(t, tt.expected.Metadata, result.Metadata)
        })
    }
}

func TestParse_InvalidVersions(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"empty string", ""},
        {"invalid format", "not-a-version"},
        {"negative numbers", "-1.0.0"},
        {"too many parts", "1.2.3.4.5"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := Parse(tt.input)
            assert.Error(t, err)
        })
    }
}

func TestCompare_VersionPairs(t *testing.T) {
    tests := []struct {
        name     string
        v1       string
        v2       string
        expected int
    }{
        {"equal versions", "1.0.0", "1.0.0", 0},
        {"v1 > v2", "2.0.0", "1.0.0", 1},
        {"v1 < v2", "1.0.0", "2.0.0", -1},
        {"stable > prerelease", "1.0.0", "1.0.0-beta", 1},
        {"prerelease comparison", "1.0.0-beta.10", "1.0.0-beta.2", 1},
        {"metadata ignored", "1.0.0+build1", "1.0.0+build2", 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            v1, err := Parse(tt.v1)
            require.NoError(t, err)
            v2, err := Parse(tt.v2)
            require.NoError(t, err)

            result := v1.Compare(v2)

            // Normalize result to -1, 0, or 1
            if result < 0 {
                result = -1
            } else if result > 0 {
                result = 1
            }

            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Subtests

```go
func TestFrameworkParsing(t *testing.T) {
    t.Run("modern TFMs", func(t *testing.T) {
        t.Run("net8.0", func(t *testing.T) {
            fw, err := Parse("net8.0")
            require.NoError(t, err)
            assert.Equal(t, ".NETCoreApp", fw.Framework)
            assert.Equal(t, 8, fw.Version.Major)
        })

        t.Run("netstandard2.1", func(t *testing.T) {
            fw, err := Parse("netstandard2.1")
            require.NoError(t, err)
            assert.Equal(t, ".NETStandard", fw.Framework)
        })
    })

    t.Run("legacy TFMs", func(t *testing.T) {
        t.Run("net45", func(t *testing.T) {
            fw, err := Parse("net45")
            require.NoError(t, err)
            assert.Equal(t, ".NETFramework", fw.Framework)
            assert.Equal(t, 4, fw.Version.Major)
            assert.Equal(t, 5, fw.Version.Minor)
        })
    })
}
```

---

## Integration Testing

### Mock HTTP Client

**File**: `internal/testutil/mock.go`

```go
package testutil

import (
    "bytes"
    "io"
    "io/ioutil"
    "net/http"
)

// MockHTTPClient is a mock HTTP client for testing
type MockHTTPClient struct {
    Responses map[string]*http.Response
}

// NewMockHTTPClient creates a new mock HTTP client
func NewMockHTTPClient() *MockHTTPClient {
    return &MockHTTPClient{
        Responses: make(map[string]*http.Response),
    }
}

// Do executes a mock HTTP request
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    key := req.Method + " " + req.URL.String()

    resp, ok := m.Responses[key]
    if !ok {
        return &http.Response{
            StatusCode: 404,
            Body:       io.NopCloser(bytes.NewReader([]byte{})),
        }, nil
    }

    return resp, nil
}

// AddResponse adds a mock response
func (m *MockHTTPClient) AddResponse(method, url string, statusCode int, body string) {
    key := method + " " + url
    m.Responses[key] = &http.Response{
        StatusCode: statusCode,
        Body:       io.NopCloser(bytes.NewReader([]byte(body))),
        Header:     make(http.Header),
    }
}

// AddJSONResponse adds a mock JSON response
func (m *MockHTTPClient) AddJSONResponse(method, url string, statusCode int, body string) {
    key := method + " " + url
    resp := &http.Response{
        StatusCode: statusCode,
        Body:       io.NopCloser(bytes.NewReader([]byte(body))),
        Header:     make(http.Header),
    }
    resp.Header.Set("Content-Type", "application/json")
    m.Responses[key] = resp
}
```

### Integration Test Example

**File**: `pkg/gonuget/api/v3/search_integration_test.go`

```go
// +build integration

package v3

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/willibrandon/gonuget/internal/testutil"
)

func TestSearchClient_Search_RealAPI(t *testing.T) {
    // Use real nuget.org API (with caching to avoid rate limits)
    httpClient := &http.Client{}
    indexClient := NewServiceIndexClient(httpClient, "https://api.nuget.org/v3/index.json", testutil.NullLogger())
    searchClient := NewSearchClient(httpClient, indexClient, testutil.NullLogger())

    ctx := context.Background()
    result, err := searchClient.Search(ctx, "Newtonsoft.Json", &SearchOptions{
        Take: 20,
    })

    require.NoError(t, err)
    require.NotNil(t, result)
    assert.Greater(t, result.TotalHits, int64(0))
    assert.Greater(t, len(result.Data), 0)

    // First result should be Newtonsoft.Json
    assert.Equal(t, "Newtonsoft.Json", result.Data[0].ID)
}

func TestSearchClient_Search_WithMock(t *testing.T) {
    // Mock HTTP client with fixture
    mock := testutil.NewMockHTTPClient()

    // Load fixture
    indexJSON := testutil.LoadFixture(t, "service-index.json")
    searchJSON := testutil.LoadFixture(t, "search-newtonsoft.json")

    // Add mock responses
    mock.AddJSONResponse("GET", "https://api.nuget.org/v3/index.json", 200, indexJSON)
    mock.AddJSONResponse("GET", "https://azuresearch-usnc.nuget.org/query?q=Newtonsoft&skip=0&take=20&prerelease=false&semVerLevel=2.0.0", 200, searchJSON)

    indexClient := NewServiceIndexClient(mock, "https://api.nuget.org/v3/index.json", testutil.NullLogger())
    searchClient := NewSearchClient(mock, indexClient, testutil.NullLogger())

    ctx := context.Background()
    result, err := searchClient.Search(ctx, "Newtonsoft", &SearchOptions{
        Take: 20,
    })

    require.NoError(t, err)
    assert.Equal(t, int64(247), result.TotalHits)
    assert.Equal(t, "Newtonsoft.Json", result.Data[0].ID)
}
```

---

## Benchmark Testing

### Benchmark Example

**File**: `pkg/gonuget/version/version_bench_test.go`

```go
package version

import (
    "testing"
)

func BenchmarkParse(b *testing.B) {
    versions := []string{
        "1.0.0",
        "1.0.0-beta",
        "1.0.0-beta.1",
        "1.0.0+build.123",
        "1.2.3.4",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        Parse(versions[i%len(versions)])
    }
}

func BenchmarkCompare(b *testing.B) {
    v1 := MustParse("1.0.0")
    v2 := MustParse("2.0.0")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        v1.Compare(v2)
    }
}

func BenchmarkParse_Parallel(b *testing.B) {
    versions := []string{
        "1.0.0",
        "1.0.0-beta",
        "1.2.3.4",
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            Parse(versions[i%len(versions)])
            i++
        }
    })
}

// Benchmark vs C# baseline
func BenchmarkVersionComparison(b *testing.B) {
    v1 := MustParse("1.0.0")
    v2 := MustParse("2.0.0")

    b.Run("Compare", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            v1.Compare(v2)
        }
    })

    b.Run("Equals", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            v1.Equals(v2)
        }
    })

    b.Run("LessThan", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            v1.LessThan(v2)
        }
    })
}
```

### Performance Targets

| Operation | Target | Baseline (C#) |
|-----------|--------|---------------|
| Version parse | <250ns/op | ~500ns/op |
| Version compare | <20ns/op | ~100ns/op |
| Framework parse | <300ns/op | ~600ns/op |
| Framework compat | <50ns/op | ~150ns/op |

---

## Test Utilities

### Fixture Loading

**File**: `internal/testutil/fixtures.go`

```go
package testutil

import (
    "io/ioutil"
    "path/filepath"
    "testing"
)

// LoadFixture loads a test fixture file
func LoadFixture(t *testing.T, filename string) string {
    t.Helper()

    path := filepath.Join("testdata", "fixtures", filename)
    data, err := ioutil.ReadFile(path)
    if err != nil {
        t.Fatalf("failed to load fixture %s: %v", filename, err)
    }

    return string(data)
}

// LoadPackageFixture loads a test package file
func LoadPackageFixture(t *testing.T, filename string) string {
    t.Helper()

    path := filepath.Join("testdata", "packages", filename)
    return path
}

// LoadCertificateFixture loads a test certificate
func LoadCertificateFixture(t *testing.T, filename string) string {
    t.Helper()

    path := filepath.Join("testdata", "certificates", filename)
    data, err := ioutil.ReadFile(path)
    if err != nil {
        t.Fatalf("failed to load certificate %s: %v", filename, err)
    }

    return string(data)
}
```

### Custom Assertions

**File**: `internal/testutil/assert.go`

```go
package testutil

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

// AssertVersionEquals asserts two versions are equal
func AssertVersionEquals(t *testing.T, expected, actual *version.NuGetVersion) {
    t.Helper()

    assert.Equal(t, expected.Major, actual.Major, "Major version mismatch")
    assert.Equal(t, expected.Minor, actual.Minor, "Minor version mismatch")
    assert.Equal(t, expected.Patch, actual.Patch, "Patch version mismatch")
    assert.Equal(t, expected.ReleaseLabels, actual.ReleaseLabels, "Release labels mismatch")
}

// AssertFrameworkEquals asserts two frameworks are equal
func AssertFrameworkEquals(t *testing.T, expected, actual *framework.NuGetFramework) {
    t.Helper()

    assert.Equal(t, expected.Framework, actual.Framework, "Framework identifier mismatch")
    assert.Equal(t, expected.Version, actual.Version, "Framework version mismatch")
    assert.Equal(t, expected.Platform, actual.Platform, "Platform mismatch")
}

// AssertHTTPError asserts an HTTP error occurred
func AssertHTTPError(t *testing.T, err error, expectedStatus int) {
    t.Helper()

    assert.Error(t, err)

    var httpErr *HTTPError
    if assert.ErrorAs(t, err, &httpErr) {
        assert.Equal(t, expectedStatus, httpErr.StatusCode)
    }
}
```

### Test HTTP Server

**File**: `internal/testutil/server.go`

```go
package testutil

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

// NewTestServer creates a test HTTP server
func NewTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
    t.Helper()

    server := httptest.NewServer(handler)
    t.Cleanup(server.Close)

    return server
}

// NewV3TestServer creates a test server with V3 endpoints
func NewV3TestServer(t *testing.T) *httptest.Server {
    t.Helper()

    mux := http.NewServeMux()

    // Service index endpoint
    mux.HandleFunc("/v3/index.json", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(LoadFixture(t, "service-index.json")))
    })

    // Search endpoint
    mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
        query := r.URL.Query().Get("q")
        w.Header().Set("Content-Type", "application/json")

        if query == "Newtonsoft" {
            w.Write([]byte(LoadFixture(t, "search-newtonsoft.json")))
        } else {
            w.Write([]byte(`{"totalHits":0,"data":[]}`))
        }
    })

    server := httptest.NewServer(mux)
    t.Cleanup(server.Close)

    return server
}
```

---

## Golden File Testing

### Golden File Pattern

**File**: `pkg/gonuget/api/v3/search_golden_test.go`

```go
package v3

import (
    "encoding/json"
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/willibrandon/gonuget/internal/testutil"
)

func TestSearchResultParsing_GoldenFile(t *testing.T) {
    // Load golden file (real API response)
    golden := testutil.LoadFixture(t, "search-newtonsoft.json")

    // Parse JSON
    var result SearchResult
    err := json.Unmarshal([]byte(golden), &result)
    require.NoError(t, err)

    // Assertions
    require.Equal(t, int64(247), result.TotalHits)
    require.Greater(t, len(result.Data), 0)

    // Verify first result
    first := result.Data[0]
    require.Equal(t, "Newtonsoft.Json", first.ID)
    require.NotEmpty(t, first.Description)
    require.Greater(t, first.TotalDownloads, int64(0))

    // Verify versions
    require.Greater(t, len(first.Versions), 0)

    // Verify flexible string arrays
    require.Greater(t, len(first.Authors), 0)
    require.Greater(t, len(first.Tags), 0)
}
```

### Updating Golden Files

```bash
# Fetch fresh API responses
go test -tags=integration -update-golden

# Review changes
git diff testdata/fixtures/

# Commit if correct
git add testdata/fixtures/
git commit -m "Update golden files from API"
```

---

## Coverage Targets

### Overall Coverage

- **Minimum**: 80% coverage
- **Target**: 90% coverage
- **Critical paths**: 100% coverage (version parsing, framework compat)

### Coverage by Component

| Component | Target | Critical |
|-----------|--------|----------|
| version/ | 95% | Yes |
| framework/ | 95% | Yes |
| http/ | 85% | No |
| api/ | 85% | No |
| pack/ | 80% | No |
| signature/ | 80% | No |

### Running Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Coverage per package
go test -cover ./...

# Coverage with details
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
```

---

## CI/CD Integration

### GitHub Actions Workflow

**File**: `.github/workflows/test.yml`

```yaml
name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go: ['1.21', '1.22']

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go }}

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Run unit tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
          flags: unittests

      - name: Run integration tests
        if: github.event_name == 'push'
        run: go test -v -tags=integration ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest

  benchmark:
    runs-on: ubuntu-latest
    if: github.event_name == 'pull_request'
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Run benchmarks
        run: go test -bench=. -benchmem ./... | tee benchmark.txt

      - name: Comment PR with benchmarks
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const benchmark = fs.readFileSync('benchmark.txt', 'utf8');
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '### Benchmark Results\n```\n' + benchmark + '\n```'
            });
```

### Makefile

**File**: `Makefile`

```makefile
.PHONY: test test-unit test-integration test-e2e benchmark coverage lint

# Run all unit tests
test-unit:
	go test -v -race ./...

# Run integration tests (requires network)
test-integration:
	go test -v -tags=integration ./...

# Run end-to-end tests
test-e2e:
	go test -v -tags=e2e ./tests/e2e/...

# Run all tests
test: test-unit test-integration

# Run benchmarks
benchmark:
	go test -bench=. -benchmem ./...

# Generate coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	golangci-lint run

# Run tests with coverage and upload to codecov
coverage-ci:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	bash <(curl -s https://codecov.io/bash)

# Clean test cache
clean-test:
	go clean -testcache
```

---

## Test Quality Checklist

### Writing New Tests

- [ ] Test has clear, descriptive name
- [ ] Test uses table-driven approach if testing multiple cases
- [ ] Test checks both success and error cases
- [ ] Test uses `require` for critical assertions, `assert` for non-critical
- [ ] Test has no external dependencies (or uses mocks)
- [ ] Test cleans up resources (files, connections, etc.)
- [ ] Test runs quickly (<100ms for unit tests)
- [ ] Test is deterministic (no race conditions, no randomness)

### Code Review Checklist

- [ ] All new code has tests
- [ ] Coverage didn't decrease
- [ ] Integration tests added for new features
- [ ] Benchmarks added for performance-critical code
- [ ] Tests are passing in CI
- [ ] No flaky tests introduced

---

**Document Status**: Draft v1.0
**Last Updated**: 2025-01-19
**Next Review**: After implementation
