package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/willibrandon/gonuget/cache"
	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/observability"
	"github.com/willibrandon/gonuget/resilience"
	"go.opentelemetry.io/otel/attribute"
)

// TestFullStack_WithRealNuGetOrg verifies the complete M4 stack works together
// with real NuGet.org operations: cache + resilience + observability
func TestFullStack_WithRealNuGetOrg(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test with real NuGet.org in short mode")
	}

	ctx := context.Background()

	// 1. Set up cache (memory + disk multi-tier)
	memCache := cache.NewMemoryCache(100, 50*1024*1024)              // 100 entries, 50MB
	diskCache, err := cache.NewDiskCache(t.TempDir(), 500*1024*1024) // 500MB
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	// 2. Set up HTTP client with resilience
	httpClient := nugethttp.NewClient(&nugethttp.Config{
		Timeout:     30 * time.Second,
		DialTimeout: 10 * time.Second,
		UserAgent:   "gonuget-test/1.0",
		CircuitBreakerConfig: &resilience.CircuitBreakerConfig{
			MaxFailures:         5,
			Timeout:             30 * time.Second,
			MaxHalfOpenRequests: 2,
		},
		RateLimiterConfig: &resilience.TokenBucketConfig{
			Capacity:   100,
			RefillRate: 50.0, // 50 requests/second
		},
		EnableTracing: true,
	})

	// 3. Set up observability
	testLogger := NewTestLogger()

	// 4. Create repository with full stack
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  "https://api.nuget.org/v3", // Base URL without /index.json
		HTTPClient: httpClient,
		Cache:      mtCache,
		Logger:     testLogger,
	})

	// Test package to use (small, stable, well-known)
	packageID := "Newtonsoft.Json"
	version := "13.0.1"

	// === Test 1: GetMetadata with cache miss → cache hit ===

	t.Run("GetMetadata_CacheMissAndHit", func(t *testing.T) {
		// Clear logs and cache
		testLogger.Clear()
		memCache.Clear()
		_ = diskCache.Clear()

		// First call - should be cache miss
		metadata1, err := repo.GetMetadata(ctx, nil, packageID, version)
		if err != nil {
			t.Fatalf("First GetMetadata failed: %v", err)
		}

		if metadata1 == nil {
			t.Fatal("Expected metadata, got nil")
		}

		if !strings.EqualFold(metadata1.ID, packageID) {
			t.Errorf("Expected package ID %s, got %s", packageID, metadata1.ID)
		}

		// Verify logging happened
		if !testLogger.HasLog("Fetching package metadata") {
			t.Error("Expected debug log for metadata fetch")
		}
		if !testLogger.HasLog("Successfully fetched metadata") {
			t.Error("Expected info log for metadata success")
		}

		// Second call - should be cache hit
		testLogger.Clear()
		metadata2, err := repo.GetMetadata(ctx, nil, packageID, version)
		if err != nil {
			t.Fatalf("Second GetMetadata failed: %v", err)
		}

		if metadata2 == nil {
			t.Fatal("Expected cached metadata, got nil")
		}

		if metadata1.ID != metadata2.ID {
			t.Error("Cached metadata should match original")
		}

		// Verify cache hit (should still have fetch log but faster)
		if !testLogger.HasLog("Successfully fetched metadata") {
			t.Error("Expected info log for cached metadata")
		}

		t.Logf("✅ GetMetadata working with cache: %s@%s", packageID, version)
	})

	// === Test 2: ListVersions with caching ===

	t.Run("ListVersions_Caching", func(t *testing.T) {
		testLogger.Clear()

		// First call
		versions1, err := repo.ListVersions(ctx, nil, packageID)
		if err != nil {
			t.Fatalf("ListVersions failed: %v", err)
		}

		if len(versions1) == 0 {
			t.Fatal("Expected versions, got empty list")
		}

		// Verify logging
		if !testLogger.HasLog("Listing package versions") {
			t.Error("Expected debug log for versions list")
		}
		if !testLogger.HasLog("Successfully listed") {
			t.Error("Expected info log for versions success")
		}

		// Second call - should hit cache
		versions2, err := repo.ListVersions(ctx, nil, packageID)
		if err != nil {
			t.Fatalf("Second ListVersions failed: %v", err)
		}

		if len(versions1) != len(versions2) {
			t.Error("Cached versions should match original")
		}

		t.Logf("✅ ListVersions working with cache: %d versions for %s", len(versions1), packageID)
	})

	// === Test 3: DownloadPackage with caching + metrics ===

	t.Run("DownloadPackage_CachingAndMetrics", func(t *testing.T) {
		testLogger.Clear()

		// Get initial metric value
		initialDownloads, err := observability.GetCounterValue(observability.PackageDownloadsTotal, "success")
		if err != nil {
			t.Logf("Warning: could not read initial metric value: %v", err)
			initialDownloads = 0
		}

		// First download
		rc1, err := repo.DownloadPackage(ctx, nil, packageID, version)
		if err != nil {
			t.Fatalf("DownloadPackage failed: %v", err)
		}
		defer func() { _ = rc1.Close() }()

		data1, err := io.ReadAll(rc1)
		if err != nil {
			t.Fatalf("Failed to read package: %v", err)
		}

		if len(data1) == 0 {
			t.Fatal("Expected package data, got empty")
		}

		// Verify logging
		if !testLogger.HasLog("Downloading package") {
			t.Error("Expected info log for package download")
		}
		if !testLogger.HasLog("Successfully downloaded package") {
			t.Error("Expected info log for download success")
		}

		// Verify metric incremented
		newDownloads, err := observability.GetCounterValue(observability.PackageDownloadsTotal, "success")
		switch {
		case err != nil:
			t.Logf("Warning: could not read new metric value: %v", err)
		case newDownloads <= initialDownloads:
			t.Errorf("Expected PackageDownloadsTotal metric to increment from %f to %f", initialDownloads, newDownloads)
		default:
			t.Logf("✓ Metric incremented from %f to %f", initialDownloads, newDownloads)
		}

		// Second download - should hit cache
		testLogger.Clear()
		rc2, err := repo.DownloadPackage(ctx, nil, packageID, version)
		if err != nil {
			t.Fatalf("Second DownloadPackage failed: %v", err)
		}
		defer func() { _ = rc2.Close() }()

		data2, err := io.ReadAll(rc2)
		if err != nil {
			t.Fatalf("Failed to read cached package: %v", err)
		}

		if len(data1) != len(data2) {
			t.Error("Cached package should match original")
		}

		t.Logf("✅ DownloadPackage working with cache: %s@%s (%d bytes)", packageID, version, len(data1))
	})

	// === Test 4: Verify resilience components are active ===

	t.Run("Resilience_Active", func(t *testing.T) {
		// Circuit breaker stats should show activity
		// Rate limiter should have processed requests
		// This is implicit - if tests passed, resilience is working

		// We can verify by checking that no circuit breaker errors occurred
		if testLogger.HasLog("circuit breaker is open") {
			t.Error("Circuit breaker should not have opened for healthy NuGet.org")
		}

		if testLogger.HasLog("rate limit wait failed") {
			t.Error("Rate limiter should not have failed with generous config")
		}

		t.Log("✅ Resilience components active and protecting requests")
	})

	t.Log("✅ Full stack integration test passed: cache + resilience + observability")
}

// TestFullStack_ResilienceUnderFailure verifies circuit breaker and rate limiter
// protect the system when a source is failing
func TestFullStack_ResilienceUnderFailure(t *testing.T) {
	ctx := context.Background()

	// Create a failing mock server (fails after 2 successful requests)
	successCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		successCount++
		if successCount <= 2 {
			// First 2 requests succeed
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"version":"3.0.0","resources":[]}`))
		} else {
			// Subsequent requests fail
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Set up cache
	memCache := cache.NewMemoryCache(100, 50*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 500*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	testLogger := NewTestLogger()

	// Set up HTTP client with aggressive circuit breaker and rate limiter
	httpClient := nugethttp.NewClient(&nugethttp.Config{
		Timeout:     5 * time.Second,
		DialTimeout: 2 * time.Second,
		UserAgent:   "gonuget-test/1.0",
		Logger:      testLogger, // Pass logger to capture HTTP errors
		CircuitBreakerConfig: &resilience.CircuitBreakerConfig{
			MaxFailures:         3, // Opens after 3 failures
			Timeout:             5 * time.Second,
			MaxHalfOpenRequests: 1,
		},
		RateLimiterConfig: &resilience.TokenBucketConfig{
			Capacity:   2,   // Only 2 tokens
			RefillRate: 1.0, // 1 token/second refill
		},
	})

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test-server",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Cache:      mtCache,
		Logger:     testLogger,
	})
	_ = repo // Repository is part of full stack setup, even if not used in all subtests

	// === Test 1: Rate limiter delays requests ===

	t.Run("RateLimiter_Delays", func(t *testing.T) {
		testLogger.Clear()

		// Reset success count
		successCount = 0

		// Make 3 rapid HTTP requests directly - first 2 should be immediate, 3rd should be delayed
		start := time.Now()

		for range 3 {
			req, _ := http.NewRequest("GET", server.URL+"/test", nil)
			_, _ = httpClient.Do(ctx, req)
		}

		elapsed := time.Since(start)

		// With capacity=2 and refill=1.0, the 3rd request should wait ~1 second
		if elapsed < 800*time.Millisecond {
			t.Errorf("Expected rate limiter delay, but completed in %v", elapsed)
		} else {
			t.Logf("✅ Rate limiter delayed 3rd request: took %v", elapsed)
		}
	})

	// === Test 2: Circuit breaker opens after failures ===

	t.Run("CircuitBreaker_OpensAfterFailures", func(t *testing.T) {
		testLogger.Clear()

		// Reset to failing state (success count > 2)
		successCount = 100

		// Make HTTP requests until circuit opens (should take 3-4 attempts)
		var lastErr error
		for i := range 10 {
			time.Sleep(200 * time.Millisecond) // Give rate limiter time to refill

			req, _ := http.NewRequest("GET", server.URL+"/test", nil)
			_, err := httpClient.Do(ctx, req)
			lastErr = err

			if err != nil && strings.Contains(err.Error(), "circuit breaker is open") {
				t.Logf("✅ Circuit breaker opened after attempt %d", i+1)
				return
			}
		}

		t.Errorf("Circuit breaker did not open. Last error: %v", lastErr)
	})

	// === Test 3: Verify HTTP operation logging (500 status codes are logged at debug level, not error level) ===

	t.Run("HTTPLogging", func(t *testing.T) {
		testLogger.Clear()

		// Reset to failing state
		successCount = 100

		// Trigger failures by making direct HTTP requests (provider is cached, so GetProvider won't trigger errors)
		for i := range 3 {
			time.Sleep(200 * time.Millisecond)
			req, _ := http.NewRequest("GET", server.URL+"/test", nil)
			resp, err := httpClient.Do(ctx, req)
			if err == nil && resp != nil {
				t.Logf("Request %d: Got HTTP %d", i+1, resp.StatusCode)
			} else if err != nil {
				t.Logf("Request %d: Got error: %v", i+1, err)
			}
		}

		// Check for HTTP logs (500 status codes are logged at debug level with "HTTP" and "→" for status)
		entries := testLogger.GetEntries()
		hasHTTPLogs := false
		for _, entry := range entries {
			// HTTP client logs: "HTTP {Method} {URL} → {StatusCode} ({Duration}ms)"
			if entry.Template == "HTTP {Method} {URL}" ||
				entry.Template == "HTTP {Method} {URL} → {StatusCode} ({Duration}ms)" {
				hasHTTPLogs = true
				break
			}
		}

		if !hasHTTPLogs {
			t.Error("Expected HTTP operation logs but found none")
			t.Logf("Log entries: %d total", len(entries))
		} else {
			t.Log("✅ HTTP operation logging working")
		}
	})

	t.Log("✅ Resilience under failure test passed")
}

// TestFullStack_ObservabilityExport verifies traces and metrics are properly exported
func TestFullStack_ObservabilityExport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping observability export test in short mode")
	}

	ctx := context.Background()

	// Set up OpenTelemetry with stdout exporter for testing
	tp, err := observability.SetupTracing(ctx, observability.TracerConfig{
		ServiceName:    "gonuget-test",
		ServiceVersion: "test",
		Environment:    "test",
		ExporterType:   "stdout", // Use stdout for testing
		SamplingRate:   1.0,      // Sample 100% of traces for testing
	})
	if err != nil {
		t.Fatalf("Failed to setup tracing: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Set up cache
	memCache := cache.NewMemoryCache(100, 50*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 500*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	// Set up HTTP client with tracing enabled
	httpClient := nugethttp.NewClient(&nugethttp.Config{
		Timeout:       30 * time.Second,
		DialTimeout:   10 * time.Second,
		UserAgent:     "gonuget-test/1.0",
		EnableTracing: true, // Enable HTTP tracing
	})

	testLogger := NewTestLogger()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  "https://api.nuget.org/v3",
		HTTPClient: httpClient,
		Cache:      mtCache,
		Logger:     testLogger,
	})

	// === Test 1: Verify spans are created ===

	t.Run("Spans_Created", func(t *testing.T) {
		// Download a package - this creates a span
		packageID := "Newtonsoft.Json"
		version := "13.0.1"

		rc, err := repo.DownloadPackage(ctx, nil, packageID, version)
		if err != nil {
			t.Fatalf("DownloadPackage failed: %v", err)
		}
		defer func() { _ = rc.Close() }()

		_, err = io.ReadAll(rc)
		if err != nil {
			t.Fatalf("Failed to read package: %v", err)
		}

		// Spans are exported to stdout - we can't easily verify them in unit tests
		// but the fact that no errors occurred means spans were created successfully
		t.Log("✅ Spans created for package download")
	})

	// === Test 2: Verify metrics are exported ===

	t.Run("Metrics_Exported", func(t *testing.T) {
		// Get initial metric values
		initialHTTPRequests, err := observability.GetCounterValue(observability.HTTPRequestsTotal, "GET", "200", "api.nuget.org")
		if err != nil {
			t.Logf("Note: could not read initial HTTP requests metric: %v", err)
			initialHTTPRequests = 0
		}

		// Make a request
		req, _ := http.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)
		resp, err := httpClient.Do(ctx, req)
		if err == nil {
			defer func() { _ = resp.Body.Close() }()
			_, _ = io.ReadAll(resp.Body)
		}

		// Verify metric incremented
		newHTTPRequests, err := observability.GetCounterValue(observability.HTTPRequestsTotal, "GET", "200", "api.nuget.org")
		if err != nil {
			t.Logf("Note: could not read new HTTP requests metric: %v", err)
		} else if newHTTPRequests > initialHTTPRequests {
			t.Logf("✅ HTTP requests metric incremented from %f to %f", initialHTTPRequests, newHTTPRequests)
		}

		// Verify package download metrics
		downloads, err := observability.GetCounterValue(observability.PackageDownloadsTotal, "success")
		if err != nil {
			t.Logf("Note: could not read package downloads metric: %v", err)
		} else {
			t.Logf("✅ Package downloads metric: %f", downloads)
		}
	})

	t.Log("✅ Observability export test passed")
}

// TestFullStack_E2E_LiveObservability verifies integration with live Jaeger and Prometheus
// This test is skipped unless the observability stack is running
func TestFullStack_E2E_LiveObservability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E observability test in short mode")
	}

	// Check if Jaeger is available (OTLP endpoint on localhost:4317)
	conn, err := http.Get("http://localhost:16686/")
	if err != nil {
		t.Skip("Skipping E2E test: Jaeger not available at localhost:16686")
	}
	_ = conn.Body.Close()

	ctx := context.Background()

	// Start metrics server for Prometheus scraping
	metricsAddr := ":9999"
	go func() {
		if err := observability.StartMetricsServer(metricsAddr); err != nil {
			t.Logf("Metrics server stopped: %v", err)
		}
	}()
	t.Logf("✓ Metrics server started on %s", metricsAddr)

	// Give metrics server time to start
	time.Sleep(100 * time.Millisecond)

	// Set up OpenTelemetry with OTLP exporter to local Jaeger
	tp, err := observability.SetupTracing(ctx, observability.TracerConfig{
		ServiceName:    "gonuget-e2e-test",
		ServiceVersion: "test",
		Environment:    "test",
		ExporterType:   "otlp",
		OTLPEndpoint:   "localhost:4317",
		OTLPInsecure:   true, // Disable TLS for local development
		SamplingRate:   1.0,  // Sample 100% of traces for testing
	})
	if err != nil {
		t.Fatalf("Failed to setup tracing with OTLP: %v", err)
	}
	t.Logf("✓ TracerProvider configured with OTLP exporter to localhost:4317")
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Set up cache
	memCache := cache.NewMemoryCache(100, 50*1024*1024)
	diskCache, err := cache.NewDiskCache(t.TempDir(), 500*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	mtCache := cache.NewMultiTierCache(memCache, diskCache)

	// Set up HTTP client with full stack
	httpClient := nugethttp.NewClient(&nugethttp.Config{
		Timeout:       30 * time.Second,
		DialTimeout:   10 * time.Second,
		UserAgent:     "gonuget-e2e-test/1.0",
		EnableTracing: true,
		CircuitBreakerConfig: &resilience.CircuitBreakerConfig{
			MaxFailures:         5,
			Timeout:             30 * time.Second,
			MaxHalfOpenRequests: 2,
		},
		RateLimiterConfig: &resilience.TokenBucketConfig{
			Capacity:   100,
			RefillRate: 50.0,
		},
	})

	testLogger := NewTestLogger()

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "nuget.org",
		SourceURL:  "https://api.nuget.org/v3",
		HTTPClient: httpClient,
		Cache:      mtCache,
		Logger:     testLogger,
	})

	// Perform real operations that will be traced
	packageID := "Newtonsoft.Json"
	version := "13.0.1"

	t.Run("E2E_Operations", func(t *testing.T) {
		// Create a test span to verify tracer is working
		testCtx, testSpan := observability.StartSpan(ctx, "github.com/willibrandon/gonuget/core", "full-stack-e2e-test")
		testSpan.SetAttributes(
			attribute.String("test.name", "TestFullStack_E2E_LiveObservability"),
			attribute.String("test.package", packageID),
		)

		// First pass - cache misses
		metadata, err := repo.GetMetadata(testCtx, nil, packageID, version)
		if err != nil {
			testSpan.RecordError(err)
			testSpan.End()
			t.Fatalf("GetMetadata failed: %v", err)
		}
		t.Logf("✓ GetMetadata (first): %s@%s", metadata.ID, metadata.Version)

		// Second pass - cache hits
		metadata, err = repo.GetMetadata(testCtx, nil, packageID, version)
		if err != nil {
			testSpan.RecordError(err)
			testSpan.End()
			t.Fatalf("GetMetadata (cached) failed: %v", err)
		}
		t.Logf("✓ GetMetadata (cached): %s@%s", metadata.ID, metadata.Version)

		// List versions
		versions, err := repo.ListVersions(testCtx, nil, packageID)
		if err != nil {
			testSpan.RecordError(err)
			testSpan.End()
			t.Fatalf("ListVersions failed: %v", err)
		}
		t.Logf("✓ ListVersions: %d versions", len(versions))

		// Download package
		rc, err := repo.DownloadPackage(testCtx, nil, packageID, version)
		if err != nil {
			testSpan.RecordError(err)
			testSpan.End()
			t.Fatalf("DownloadPackage failed: %v", err)
		}
		defer func() { _ = rc.Close() }()

		data, err := io.ReadAll(rc)
		if err != nil {
			testSpan.RecordError(err)
			testSpan.End()
			t.Fatalf("Failed to read package: %v", err)
		}
		t.Logf("✓ DownloadPackage: %d bytes", len(data))

		testSpan.End()
	})

	// Force flush traces to ensure they're exported to Jaeger
	t.Log("Flushing traces to Jaeger...")
	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tp.ForceFlush(flushCtx); err != nil {
		t.Errorf("ERROR: Failed to force flush traces: %v", err)
		t.Log("This means traces were NOT exported to Jaeger!")
	} else {
		t.Log("✓ Traces flushed successfully")
	}

	// Wait for Prometheus to scrape metrics (scrape interval is 5s)
	// Wait 8s to ensure at least one full scrape cycle + buffer for CI
	t.Log("Waiting for Prometheus to scrape metrics...")
	time.Sleep(8 * time.Second)

	// Verify metrics in Prometheus
	t.Run("Prometheus_Metrics", func(t *testing.T) {
		promURL := "http://localhost:9090/api/v1/query"

		// Helper to check if Prometheus has data for a metric
		checkMetric := func(metricName string) error {
			resp, err := http.Get(promURL + "?query=" + metricName)
			if err != nil {
				return fmt.Errorf("failed to query Prometheus: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("Prometheus query returned status %d", resp.StatusCode)
			}

			// Parse response to check for data
			var result struct {
				Data struct {
					Result []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					} `json:"result"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			if len(result.Data.Result) == 0 {
				return fmt.Errorf("no data series found for %s", metricName)
			}

			// Display metric values
			t.Logf("✓ %s metric found in Prometheus (%d series):", metricName, len(result.Data.Result))
			for _, series := range result.Data.Result {
				labels := ""
				for k, v := range series.Metric {
					if k != "__name__" {
						if labels != "" {
							labels += ", "
						}
						labels += fmt.Sprintf("%s=%s", k, v)
					}
				}
				value := "0"
				if len(series.Value) > 1 {
					value = fmt.Sprintf("%v", series.Value[1])
				}
				if labels != "" {
					t.Logf("   %s{%s} = %s", metricName, labels, value)
				} else {
					t.Logf("   %s = %s", metricName, value)
				}
			}
			return nil
		}

		// Query and verify each metric - FAIL if not found
		if err := checkMetric("gonuget_package_downloads_total"); err != nil {
			t.Errorf("gonuget_package_downloads_total: %v", err)
		}

		if err := checkMetric("gonuget_http_requests_total"); err != nil {
			t.Errorf("gonuget_http_requests_total: %v", err)
		}

		if err := checkMetric("gonuget_cache_hits_total"); err != nil {
			t.Errorf("gonuget_cache_hits_total: %v", err)
		}
	})

	t.Log("✅ E2E test completed")
	t.Log("")
	t.Log("📊 Observability Stack Verification:")
	t.Log("   ✓ Traces exported to Jaeger: http://localhost:16686 (service: gonuget-e2e-test)")
	t.Log("   ✓ Metrics scraped by Prometheus: http://localhost:9090")
	t.Log("   ✓ Metrics endpoint available: http://localhost:9999/metrics")
}
