# M4 Interop Test Requirements Analysis

**Version:** 1.0
**Last Updated:** 2025-10-22
**Purpose:** Determine which M4 chunks require interop tests with NuGet.Client

---

## Executive Summary

After analyzing the M4 implementation guides, PRD-INFRASTRUCTURE.md, PRD-TESTING.md, and available NuGet.Client APIs, this document concludes:

**Recommendation:** M4 requires **LIMITED interop testing** compared to M3. Only **M4.1-M4.4 (Cache)** chunks warrant interop tests because they implement algorithms that must precisely match NuGet.Client's behavior (hash computation, path generation, TTL validation).

**Rationale:**
- M4.5-M4.8 (Circuit breaker, rate limiting) are internal enhancements with no NuGet.Client equivalents
- M4.9-M4.14 (Observability) are purely internal and have no protocol impact
- M4.15 (HTTP/2-3) is transparent protocol negotiation with no testable external API

---

## M3 vs M4: Testing Philosophy Difference

### M3 (Package Operations)
**Why M3 needed extensive interop tests:**
- M3 implemented algorithms that **must match NuGet.Client exactly**
- Examples: version comparison, framework compatibility, signature verification, package reading
- These have **deterministic outputs** that can be cross-validated
- NuGet.Client exposes **public static utility methods** for these operations
- **327 interop tests** across 8 test classes validate all 15 actions

### M4 (Infrastructure & Resilience)
**Why M4 needs minimal interop tests:**
- Most of M4 is **internal implementation details** (circuit breaker, observability)
- Only **cache operations** have algorithmic parity requirements
- NuGet.Client cache utilities are **mostly internal**, with limited public APIs
- Performance and resilience patterns are **internal enhancements** that don't affect compatibility

---

## Chunk-by-Chunk Analysis

### M4.1: Cache - Memory (LRU)
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- NuGet.Client uses simple `ConcurrentDictionary`, not LRU
- LRU is a gonuget enhancement for better memory management
- No public APIs to cross-validate behavior
- Internal implementation detail

**Testing Strategy:**
- Unit tests for LRU eviction logic
- Benchmarks for cache hit performance (<5ms target)
- Coverage target: 90%+

---

### M4.2: Cache - Disk Persistence
**Interop Test Needed:** ✅ **YES** (High Priority)

**Reasoning:**
- Must match NuGet.Client's exact cache structure
- Hash algorithm must produce identical output
- Cache file paths must be compatible across implementations
- Two-phase atomic update pattern must match

**Available NuGet.Client APIs for Interop:**

1. **`CachingUtility.ComputeHash(string value, bool addIdentifiableCharacters = true)`**
   - Critical: SHA-256 truncated to 20 bytes, hex-encoded
   - Returns hash string used for folder names
   - **Test cases:** Known URLs, package IDs, long strings

2. **`CachingUtility.RemoveInvalidFileNameChars(string value)`**
   - Sanitizes file names by replacing invalid chars with underscores
   - Collapses consecutive underscores
   - **Test cases:** Paths with `/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|`

3. **`HttpCacheUtility.InitializeHttpCacheResult(string httpCacheDirectory, Uri sourceUri, string cacheKey, HttpSourceCacheContext context)`**
   - Generates cache file paths
   - Returns `HttpCacheResult` with `CacheFile` and `NewFile` paths
   - **Test cases:** Various source URIs and cache keys

**Proposed Interop Bridge Actions:**

```json
// Action: compute_cache_hash
{
  "action": "compute_cache_hash",
  "data": {
    "value": "https://api.nuget.org/v3/index.json",
    "addIdentifiableCharacters": true
  }
}
// Expected response:
{
  "success": true,
  "data": {
    "hash": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0$x.json"
  }
}

// Action: sanitize_cache_filename
{
  "action": "sanitize_cache_filename",
  "data": {
    "value": "package:id/with\\invalid*chars?.nupkg"
  }
}
// Expected response:
{
  "success": true,
  "data": {
    "sanitized": "package_id_with_invalid_chars_.nupkg"
  }
}

// Action: generate_cache_paths
{
  "action": "generate_cache_paths",
  "data": {
    "cacheDirectory": "/tmp/nuget-cache",
    "sourceUri": "https://api.nuget.org/v3/index.json",
    "cacheKey": "registration/newtonsoft.json/index.json",
    "maxAge": 1800
  }
}
// Expected response:
{
  "success": true,
  "data": {
    "baseFolderName": "a1b2c3d4...",
    "cacheFile": "/tmp/nuget-cache/a1b2c3d4.../registration_newtonsoft.json_index.json.dat",
    "newFile": "/tmp/nuget-cache/a1b2c3d4.../registration_newtonsoft.json_index.json.dat-new"
  }
}
```

**Test Coverage Target:** 95% (critical path)

---

### M4.3: Multi-Tier Cache
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- L1/L2 cache coordination is internal implementation
- NuGet.Client doesn't expose multi-tier cache APIs
- Can be validated with unit tests and integration tests

**Testing Strategy:**
- Unit tests for L1 → L2 promotion
- Integration tests for cache hit/miss patterns
- Performance benchmarks for tier coordination
- Coverage target: 90%

---

### M4.4: Cache Validation (ETag, TTL)
**Interop Test Needed:** ⚠️ **PARTIAL** (TTL only)

**Reasoning:**
- TTL validation must match NuGet.Client (30-minute default)
- NuGet.Client does **NOT** use ETag/If-None-Match (confirmed in analysis)
- ETag support is a gonuget enhancement

**Available NuGet.Client APIs for Interop:**

1. **`CachingUtility.ReadCacheFile(TimeSpan maxAge, string cacheFile)`**
   - Validates file age against TTL threshold
   - Returns `FileStream` if valid, `null` if expired
   - **Test cases:** Files with various ages, expired files, missing files

**Proposed Interop Bridge Action:**

```json
// Action: validate_cache_file
{
  "action": "validate_cache_file",
  "data": {
    "cacheFile": "/tmp/nuget-cache/a1b2c3d4.../test.dat",
    "maxAgeSeconds": 1800
  }
}
// Expected response:
{
  "success": true,
  "data": {
    "valid": true,
    "age": 450.5,
    "expired": false
  }
}
```

**Test Coverage Target:** 90%

---

### M4.5: Circuit Breaker - State Machine
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- NuGet.Client **does NOT implement circuit breakers**
- This is a gonuget enhancement for resilience
- No equivalent NuGet.Client APIs to cross-validate

**Testing Strategy:**
- Unit tests for state transitions (Closed → Open → Half-Open)
- Unit tests for failure counting and thresholds
- Unit tests for timeout and recovery logic
- Coverage target: 90%

---

### M4.6: Circuit Breaker - Integration with HTTP
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- Internal enhancement wrapping HTTP operations
- No protocol-level changes
- No NuGet.Client equivalent

**Testing Strategy:**
- Integration tests with mock HTTP failures
- Verify circuit opens after threshold failures
- Verify recovery in Half-Open state
- Coverage target: 90%

---

### M4.7: Rate Limiter - Token Bucket
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- NuGet.Client uses **semaphore-based throttling** (IThrottle, SemaphoreSlimThrottle)
- Token bucket is a gonuget enhancement (more restrictive, but compatible)
- No public APIs for rate limiting in NuGet.Client

**Testing Strategy:**
- Unit tests for token bucket refill rates
- Unit tests for burst capacity
- Benchmarks for token acquisition
- Coverage target: 90%

---

### M4.8: Rate Limiter - Per-Source
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- Per-source isolation is a gonuget enhancement
- NuGet.Client has global throttling via IThrottle
- No cross-validation needed

**Testing Strategy:**
- Unit tests for per-source limiter isolation
- Integration tests with multiple sources
- Coverage target: 90%

---

### M4.9: mtlog Integration
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- Logging is entirely internal (no protocol impact)
- NuGet.Client uses different logging framework (ILogger)
- No cross-validation needed

**Testing Strategy:**
- Unit tests for logger interface
- Integration tests for log output formats
- Benchmarks for zero-allocation performance
- Coverage target: 85% (observability code)

---

### M4.10: OpenTelemetry - Tracing Setup
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- Tracing is internal observability
- No protocol impact
- Can be disabled without affecting functionality

**Testing Strategy:**
- Integration tests with OTLP exporters
- Unit tests for span creation
- Coverage target: 85%

---

### M4.11: OpenTelemetry - HTTP Instrumentation
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- HTTP tracing is transparent to protocol
- No NuGet.Client equivalent

**Testing Strategy:**
- Integration tests with HTTP middleware
- Verify trace context propagation (W3C Trace Context)
- Coverage target: 85%

---

### M4.12: OpenTelemetry - Operation Spans
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- Operation-level spans are internal
- No protocol impact

**Testing Strategy:**
- Unit tests for span attributes
- Integration tests for span hierarchies
- Coverage target: 85%

---

### M4.13: Prometheus Metrics
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- Metrics exposition is internal
- No protocol impact

**Testing Strategy:**
- Unit tests for metric registration
- Integration tests for scrape endpoint
- Coverage target: 85%

---

### M4.14: Health Checks
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- Health check system is internal
- No NuGet.Client equivalent

**Testing Strategy:**
- Unit tests for check registration
- Integration tests for aggregate status
- Coverage target: 85%

---

### M4.15: HTTP/2 and HTTP/3 Support
**Interop Test Needed:** ❌ **NO**

**Reasoning:**
- Protocol negotiation is transparent to application layer
- NuGet.Client supports HTTP/2 via .NET runtime
- No testable APIs (automatic negotiation)

**Testing Strategy:**
- Integration tests with HTTP/2 servers
- Verify ALPN negotiation
- Verify fallback to HTTP/1.1
- Coverage target: 85%

---

## Summary: Required Interop Tests

### New Interop Bridge Actions Needed

Only **3 new actions** for cache operations:

| Action | Chunk | Priority | NuGet.Client API |
|--------|-------|----------|------------------|
| `compute_cache_hash` | M4.2 | P0 | `CachingUtility.ComputeHash()` |
| `sanitize_cache_filename` | M4.2 | P0 | `CachingUtility.RemoveInvalidFileNameChars()` |
| `generate_cache_paths` | M4.2 | P1 | `HttpCacheUtility.InitializeHttpCacheResult()` |
| `validate_cache_file` | M4.4 | P1 | `CachingUtility.ReadCacheFile()` |

**Total:** 4 actions (compared to 15 for M3)

---

## Test Coverage Requirements by Chunk

Per PRD-TESTING.md requirements:

| Chunk | Coverage Target | Test Types | Interop Tests |
|-------|----------------|------------|---------------|
| M4.1 | 90% | Unit, Benchmark | ❌ No |
| M4.2 | 95% | Unit, Interop, Integration | ✅ Yes (4 actions) |
| M4.3 | 90% | Unit, Integration | ❌ No |
| M4.4 | 90% | Unit, Interop, Integration | ⚠️ Partial (1 action) |
| M4.5 | 90% | Unit | ❌ No |
| M4.6 | 90% | Unit, Integration | ❌ No |
| M4.7 | 90% | Unit, Benchmark | ❌ No |
| M4.8 | 90% | Unit, Integration | ❌ No |
| M4.9 | 85% | Unit, Integration, Benchmark | ❌ No |
| M4.10 | 85% | Unit, Integration | ❌ No |
| M4.11 | 85% | Unit, Integration | ❌ No |
| M4.12 | 85% | Unit, Integration | ❌ No |
| M4.13 | 85% | Unit, Integration | ❌ No |
| M4.14 | 85% | Unit, Integration | ❌ No |
| M4.15 | 85% | Integration | ❌ No |

**Overall M4 Coverage Target:** 90% (per PRD-TESTING.md)

---

## Implementation Plan

### Phase 1: Update Interop Bridge (gonuget side)
**File:** `cmd/nuget-interop-test/main.go`

Add 4 new handler functions:
1. `handleComputeCacheHash()`
2. `handleSanitizeCacheFilename()`
3. `handleGenerateCachePaths()`
4. `handleValidateCacheFile()`

### Phase 2: Update C# Bridge Client
**File:** `tests/nuget-client-interop/GonugetInterop.Tests/TestHelpers/GonugetBridge.cs`

Add 4 new C# wrapper methods:
1. `ComputeCacheHash(string value, bool addIdentifiableCharacters)`
2. `SanitizeCacheFilename(string value)`
3. `GenerateCachePaths(string cacheDir, Uri sourceUri, string cacheKey, int maxAge)`
4. `ValidateCacheFile(string cacheFile, int maxAgeSeconds)`

### Phase 3: Create C# Test Suite
**File:** `tests/nuget-client-interop/GonugetInterop.Tests/CacheInteropTests.cs`

Implement test class with ~50 test cases:
- Hash computation (10 cases)
- Filename sanitization (10 cases)
- Cache path generation (15 cases)
- Cache validation (15 cases)

### Phase 4: Update Implementation Guides
Update M4.2 and M4.4 guides with:
- Interop test requirements
- Bridge action specifications
- Test case examples

---

## Acceptance Criteria

From PRD-INFRASTRUCTURE.md and PRD-TESTING.md:

### Functional Acceptance
- ✅ Cache hash algorithm matches NuGet.Client exactly
- ✅ Cache file paths compatible with NuGet.Client
- ✅ TTL validation produces same results
- ✅ Filename sanitization consistent

### Coverage Acceptance
- ✅ Cache package: 90%+ coverage (95% for M4.2)
- ✅ All interop tests pass (100%)
- ✅ Overall M4: 90%+ coverage

### Performance Acceptance (from PRD)
- ✅ Cache hit <5ms (memory)
- ✅ Cache hit <50ms (disk)
- ✅ Reduces network requests 80%+

---

## Comparison to M3 Interop Testing

| Metric | M3 (Package Operations) | M4 (Infrastructure) |
|--------|------------------------|---------------------|
| **Total Chunks** | 15 | 15 |
| **Chunks Needing Interop** | 12 (80%) | 2 (13%) |
| **Total Interop Actions** | 15 | 4 |
| **C# Test Classes** | 8 | 1 |
| **Total Interop Tests** | 327 | ~50 (estimated) |
| **Reason for Difference** | Public APIs, deterministic algorithms | Internal enhancements, no equivalents |

**Key Insight:** M4 is primarily about **internal infrastructure** and **observability**, whereas M3 was about **algorithmic compatibility** with public APIs. This explains why M4 needs far fewer interop tests.

---

## Conclusion

**Recommendation:** Add interop tests only for M4.2 (Disk Cache) and M4.4 (Cache Validation) chunks. The remaining M4 chunks should rely on:

1. **Unit tests** for correctness
2. **Integration tests** for end-to-end behavior
3. **Benchmarks** for performance validation
4. **Race detection** for concurrency safety (`go test -race`)

This focused approach ensures:
- ✅ Critical cache algorithms match NuGet.Client
- ✅ Internal enhancements don't add unnecessary test burden
- ✅ Overall 90%+ coverage target is achievable
- ✅ Test maintenance remains practical

**Next Steps:**
1. Update M4.2 and M4.4 implementation guides with interop test sections
2. Implement 4 new interop bridge actions
3. Create C# test suite with ~50 test cases
4. Add acceptance criteria to implementation guides

---

**END OF M4-INTEROP-ANALYSIS.md**
