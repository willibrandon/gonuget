package observability

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthChecker_Register(t *testing.T) {
	hc := NewHealthChecker()

	check := HealthCheck{
		Name: "test-check",
		Check: func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{Status: HealthStatusHealthy}
		},
	}

	hc.Register(check)

	if len(hc.checks) != 1 {
		t.Errorf("Checks count = %d, want 1", len(hc.checks))
	}
}

func TestHealthChecker_Check(t *testing.T) {
	hc := NewHealthChecker()

	hc.Register(HealthCheck{
		Name: "healthy-check",
		Check: func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{Status: HealthStatusHealthy}
		},
	})

	hc.Register(HealthCheck{
		Name: "degraded-check",
		Check: func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{Status: HealthStatusDegraded}
		},
	})

	results := hc.Check(context.Background())

	if len(results) != 2 {
		t.Errorf("Results count = %d, want 2", len(results))
	}

	if results["healthy-check"].Status != HealthStatusHealthy {
		t.Errorf("healthy-check status = %s, want healthy", results["healthy-check"].Status)
	}

	if results["degraded-check"].Status != HealthStatusDegraded {
		t.Errorf("degraded-check status = %s, want degraded", results["degraded-check"].Status)
	}
}

func TestHealthChecker_OverallStatus(t *testing.T) {
	tests := []struct {
		name     string
		checks   []HealthCheckResult
		expected HealthStatus
	}{
		{
			name: "all healthy",
			checks: []HealthCheckResult{
				{Status: HealthStatusHealthy},
				{Status: HealthStatusHealthy},
			},
			expected: HealthStatusHealthy,
		},
		{
			name: "one degraded",
			checks: []HealthCheckResult{
				{Status: HealthStatusHealthy},
				{Status: HealthStatusDegraded},
			},
			expected: HealthStatusDegraded,
		},
		{
			name: "one unhealthy",
			checks: []HealthCheckResult{
				{Status: HealthStatusHealthy},
				{Status: HealthStatusUnhealthy},
			},
			expected: HealthStatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := NewHealthChecker()

			for i, result := range tt.checks {
				r := result // Capture for closure
				hc.Register(HealthCheck{
					Name: fmt.Sprintf("check-%d", i),
					Check: func(ctx context.Context) HealthCheckResult {
						return r
					},
				})
			}

			status := hc.OverallStatus(context.Background())
			if status != tt.expected {
				t.Errorf("OverallStatus = %s, want %s", status, tt.expected)
			}
		})
	}
}

func TestHealthChecker_Handler(t *testing.T) {
	hc := NewHealthChecker()

	hc.Register(HealthCheck{
		Name: "test-check",
		Check: func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "test message",
			}
		},
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := hc.Handler()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", resp.Header.Get("Content-Type"))
	}
}

func TestHealthChecker_Cache(t *testing.T) {
	hc := NewHealthChecker()

	callCount := 0
	hc.Register(HealthCheck{
		Name:   "cached-check",
		Cached: true,
		TTL:    100 * time.Millisecond,
		Check: func(ctx context.Context) HealthCheckResult {
			callCount++
			return HealthCheckResult{Status: HealthStatusHealthy}
		},
	})

	// First call - should execute
	hc.Check(context.Background())
	if callCount != 1 {
		t.Errorf("First check call count = %d, want 1", callCount)
	}

	// Second call - should use cache
	hc.Check(context.Background())
	if callCount != 1 {
		t.Errorf("Cached check call count = %d, want 1", callCount)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Third call - should execute again
	hc.Check(context.Background())
	if callCount != 2 {
		t.Errorf("After TTL check call count = %d, want 2", callCount)
	}
}

func TestHealthChecker_Handler_UnhealthyStatus(t *testing.T) {
	hc := NewHealthChecker()

	hc.Register(HealthCheck{
		Name: "unhealthy-check",
		Check: func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusUnhealthy,
				Message: "service down",
			}
		},
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := hc.Handler()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d, want 503", resp.StatusCode)
	}
}

func TestHealthChecker_Handler_DegradedStatus(t *testing.T) {
	hc := NewHealthChecker()

	hc.Register(HealthCheck{
		Name: "degraded-check",
		Check: func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusDegraded,
				Message: "high latency",
			}
		},
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := hc.Handler()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200 (degraded still operational)", resp.StatusCode)
	}
}

func TestHTTPSourceHealthCheck(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := HTTPSourceHealthCheck("test-source", server.URL, 5*time.Second)

	result := check.Check(context.Background())

	if result.Status != HealthStatusHealthy {
		t.Errorf("Status = %s, want healthy", result.Status)
	}

	if result.Message != "source reachable" {
		t.Errorf("Message = %s, want 'source reachable'", result.Message)
	}
}

func TestHTTPSourceHealthCheck_ServerError(t *testing.T) {
	// Create test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	check := HTTPSourceHealthCheck("test-source", server.URL, 5*time.Second)

	result := check.Check(context.Background())

	if result.Status != HealthStatusDegraded {
		t.Errorf("Status = %s, want degraded", result.Status)
	}

	if result.Message != "server error" {
		t.Errorf("Message = %s, want 'server error'", result.Message)
	}
}

func TestHTTPSourceHealthCheck_Unreachable(t *testing.T) {
	check := HTTPSourceHealthCheck("test-source", "http://invalid.local.test:99999", 100*time.Millisecond)

	result := check.Check(context.Background())

	if result.Status != HealthStatusUnhealthy {
		t.Errorf("Status = %s, want unhealthy", result.Status)
	}
}

func TestCacheHealthCheck_Healthy(t *testing.T) {
	check := CacheHealthCheck("test-cache", 50*1024*1024, 100*1024*1024) // 50MB / 100MB

	result := check.Check(context.Background())

	if result.Status != HealthStatusHealthy {
		t.Errorf("Status = %s, want healthy", result.Status)
	}

	if result.Message != "cache operational" {
		t.Errorf("Message = %s, want 'cache operational'", result.Message)
	}
}

func TestCacheHealthCheck_NearlyFull(t *testing.T) {
	check := CacheHealthCheck("test-cache", 96*1024*1024, 100*1024*1024) // 96MB / 100MB

	result := check.Check(context.Background())

	if result.Status != HealthStatusDegraded {
		t.Errorf("Status = %s, want degraded", result.Status)
	}

	if result.Message != "cache nearly full" {
		t.Errorf("Message = %s, want 'cache nearly full'", result.Message)
	}

	if result.Details["usage_percent"] != "96.0%" {
		t.Errorf("usage_percent = %s, want '96.0%%'", result.Details["usage_percent"])
	}
}

func TestHealthChecker_ConcurrentChecks(t *testing.T) {
	hc := NewHealthChecker()

	// Register multiple checks
	for i := range 10 {
		name := fmt.Sprintf("check-%d", i)
		hc.Register(HealthCheck{
			Name: name,
			Check: func(ctx context.Context) HealthCheckResult {
				time.Sleep(10 * time.Millisecond) // Simulate work
				return HealthCheckResult{Status: HealthStatusHealthy}
			},
		})
	}

	// Execute checks concurrently
	start := time.Now()
	results := hc.Check(context.Background())
	elapsed := time.Since(start)

	if len(results) != 10 {
		t.Errorf("Results count = %d, want 10", len(results))
	}

	// All checks should run concurrently, so total time should be ~10ms, not 100ms
	if elapsed > 50*time.Millisecond {
		t.Errorf("Elapsed time = %v, want < 50ms (concurrent execution)", elapsed)
	}
}
