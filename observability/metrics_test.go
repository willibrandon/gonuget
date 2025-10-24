package observability

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandler(t *testing.T) {
	// Record some metrics
	HTTPRequestsTotal.WithLabelValues("GET", "200", "nuget.org").Inc()
	CacheHitsTotal.WithLabelValues("memory").Inc()
	PackageDownloadsTotal.WithLabelValues("success").Inc()

	// Create test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Serve metrics
	handler := MetricsHandler()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	body := w.Body.String()

	// Verify metric presence
	expectedMetrics := []string{
		"gonuget_http_requests_total",
		"gonuget_cache_hits_total",
		"gonuget_package_downloads_total",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Metrics output missing: %s", metric)
		}
	}
}

func TestMetricDefinitions(t *testing.T) {
	// Test that all metric definitions exist and can be used
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "HTTPRequestsTotal",
			fn: func() {
				HTTPRequestsTotal.WithLabelValues("POST", "201", "nuget.org").Inc()
			},
		},
		{
			name: "HTTPRequestDuration",
			fn: func() {
				HTTPRequestDuration.WithLabelValues("GET", "nuget.org").Observe(0.5)
			},
		},
		{
			name: "CacheHitsTotal",
			fn: func() {
				CacheHitsTotal.WithLabelValues("memory").Inc()
			},
		},
		{
			name: "CacheMissesTotal",
			fn: func() {
				CacheMissesTotal.WithLabelValues("disk").Inc()
			},
		},
		{
			name: "CacheSizeBytes",
			fn: func() {
				CacheSizeBytes.WithLabelValues("memory").Set(1024)
			},
		},
		{
			name: "PackageDownloadsTotal",
			fn: func() {
				PackageDownloadsTotal.WithLabelValues("failure").Inc()
			},
		},
		{
			name: "PackageDownloadDuration",
			fn: func() {
				PackageDownloadDuration.WithLabelValues("Newtonsoft.Json").Observe(2.5)
			},
		},
		{
			name: "CircuitBreakerState",
			fn: func() {
				CircuitBreakerState.WithLabelValues("api.nuget.org").Set(1)
			},
		},
		{
			name: "CircuitBreakerFailures",
			fn: func() {
				CircuitBreakerFailures.WithLabelValues("api.nuget.org").Inc()
			},
		},
		{
			name: "RateLimitRequestsTotal",
			fn: func() {
				RateLimitRequestsTotal.WithLabelValues("nuget.org", "true").Inc()
			},
		},
		{
			name: "RateLimitTokens",
			fn: func() {
				RateLimitTokens.WithLabelValues("nuget.org").Set(100)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			tt.fn()
		})
	}
}

func TestMetricsExposure(t *testing.T) {
	// Record metrics with various labels
	HTTPRequestsTotal.WithLabelValues("GET", "200", "nuget.org").Inc()
	HTTPRequestsTotal.WithLabelValues("POST", "404", "nuget.org").Inc()
	HTTPRequestDuration.WithLabelValues("GET", "nuget.org").Observe(0.123)

	CacheHitsTotal.WithLabelValues("memory").Add(5)
	CacheMissesTotal.WithLabelValues("disk").Add(2)
	CacheSizeBytes.WithLabelValues("memory").Set(2048)

	PackageDownloadsTotal.WithLabelValues("success").Add(10)
	PackageDownloadDuration.WithLabelValues("Newtonsoft.Json").Observe(1.5)

	CircuitBreakerState.WithLabelValues("api.nuget.org").Set(0)
	CircuitBreakerFailures.WithLabelValues("api.nuget.org").Add(3)

	RateLimitRequestsTotal.WithLabelValues("nuget.org", "true").Add(100)
	RateLimitTokens.WithLabelValues("nuget.org").Set(50)

	// Create test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Serve metrics
	handler := MetricsHandler()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}

	body := w.Body.String()

	// Verify all metric types are present
	allMetrics := []string{
		"gonuget_http_requests_total",
		"gonuget_http_request_duration_seconds",
		"gonuget_cache_hits_total",
		"gonuget_cache_misses_total",
		"gonuget_cache_size_bytes",
		"gonuget_package_downloads_total",
		"gonuget_package_download_duration_seconds",
		"gonuget_circuit_breaker_state",
		"gonuget_circuit_breaker_failures_total",
		"gonuget_rate_limit_requests_total",
		"gonuget_rate_limit_tokens",
	}

	for _, metric := range allMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Metrics output missing: %s", metric)
		}
	}

	// Verify HELP and TYPE comments are present
	if !strings.Contains(body, "# HELP") {
		t.Error("Metrics output missing HELP comments")
	}

	if !strings.Contains(body, "# TYPE") {
		t.Error("Metrics output missing TYPE comments")
	}
}
