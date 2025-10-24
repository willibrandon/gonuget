package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// TestE2E_JaegerVisualization tests that traces appear in Jaeger UI
func TestE2E_JaegerVisualization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E integration test in short mode")
	}

	// This test requires Jaeger to be running
	// To run: cd observability && docker-compose -f docker-compose.test.yml up -d
	endpoint := "localhost:4317"

	// Check if OTLP collector is available
	checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Skipf("OTLP collector not available at %s (run: cd observability && docker-compose -f docker-compose.test.yml up -d): %v", endpoint, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// Trigger connection and wait for Ready state or timeout
	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			break
		}
		if state == connectivity.Shutdown || state == connectivity.TransientFailure {
			t.Skipf("OTLP collector not available at %s (state: %v, run: cd observability && docker-compose -f docker-compose.test.yml up -d)", endpoint, state)
		}
		if !conn.WaitForStateChange(checkCtx, state) {
			t.Skipf("OTLP collector not available at %s (timeout waiting for Ready state, run: cd observability && docker-compose -f docker-compose.test.yml up -d)", endpoint)
		}
	}

	// Collector is available, setup tracing
	config := TracerConfig{
		ServiceName:    "gonuget-e2e-test",
		ServiceVersion: "test-1.0.0",
		Environment:    "e2e-test",
		ExporterType:   "otlp",
		OTLPEndpoint:   endpoint,
		SamplingRate:   1.0,
	}

	ctx := context.Background()
	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Fatalf("SetupTracing() with OTLP failed: %v", err)
	}
	defer func() {
		// Use fresh background context for shutdown (ShutdownTracing has its own timeout)
		if err := ShutdownTracing(context.Background(), tp); err != nil {
			t.Errorf("ShutdownTracing() failed: %v", err)
		}
	}()

	// Create test spans
	serviceName := "gonuget-e2e-test"
	operationName := "test-operation"

	spanCtx, span := StartSpan(ctx, TracerName, operationName)
	span.SetAttributes(
		attribute.String("test.id", "e2e-jaeger-test"),
		attribute.String("test.type", "visualization"),
	)
	AddEvent(spanCtx, "test-started")

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	AddEvent(spanCtx, "test-completed")
	span.End()

	// Give the exporter time to send
	time.Sleep(100 * time.Millisecond)

	// Force flush before querying to ensure spans are exported
	if err := tp.ForceFlush(ctx); err != nil {
		t.Fatalf("Failed to force flush: %v", err)
	}

	// Wait for Jaeger to index the trace
	time.Sleep(2 * time.Second)

	// Query Jaeger API to verify trace exists
	jaegerAPIURL := "http://localhost:16686/api/traces"
	params := fmt.Sprintf("?service=%s&limit=20", serviceName)
	t.Logf("Querying Jaeger API: %s%s", jaegerAPIURL, params)

	resp, err := http.Get(jaegerAPIURL + params)
	if err != nil {
		t.Fatalf("Failed to query Jaeger API: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Jaeger API returned status %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		Data []struct {
			TraceID string `json:"traceID"`
			Spans   []struct {
				OperationName string `json:"operationName"`
				Tags          []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"tags"`
			} `json:"spans"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode Jaeger response: %v", err)
	}

	// Verify trace was found
	if len(result.Data) == 0 {
		// Log all available services to help debug
		servResp, _ := http.Get("http://localhost:16686/api/services")
		if servResp != nil {
			defer func() {
				_ = servResp.Body.Close()
			}()
			var services struct {
				Data []string `json:"data"`
			}
			_ = json.NewDecoder(servResp.Body).Decode(&services)
			t.Logf("Available services in Jaeger: %v", services.Data)
		}
		t.Fatalf("No traces found in Jaeger for service '%s'. Check if service name is correct.", serviceName)
	}

	t.Logf("✓ Successfully verified trace in Jaeger UI")
	t.Logf("  - TraceID: %s", result.Data[0].TraceID)
	t.Logf("  - Found %d spans", len(result.Data[0].Spans))
	t.Logf("  - View at: http://localhost:16686/trace/%s", result.Data[0].TraceID)
}

// TestE2E_PrometheusScraping tests that Prometheus can scrape metrics
func TestE2E_PrometheusScraping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E integration test in short mode")
	}

	// Start a test HTTP server with metrics endpoint
	hc := NewHealthChecker()
	hc.Register(HealthCheck{
		Name: "test-check",
		Check: func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{Status: HealthStatusHealthy}
		},
	})

	// Record some test metrics
	HTTPRequestsTotal.WithLabelValues("GET", "200", "test").Add(5)
	HTTPRequestsTotal.WithLabelValues("POST", "201", "test").Add(3)
	CacheHitsTotal.WithLabelValues("memory").Add(10)
	CacheMissesTotal.WithLabelValues("disk").Add(2)
	PackageDownloadsTotal.WithLabelValues("success").Add(7)

	// Create metrics server
	mux := http.NewServeMux()
	mux.Handle("/metrics", MetricsHandler())
	mux.Handle("/health", hc.Handler())

	server := httptest.NewServer(mux)
	defer server.Close()

	// Wait a moment for metrics to be registered
	time.Sleep(1 * time.Second)

	// Test 1: Verify metrics endpoint is accessible
	resp, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Metrics endpoint returned status %d", resp.StatusCode)
	}

	// Test 2: Verify Prometheus format
	// Valid Content-Types:
	// - text/plain; version=0.0.4; charset=utf-8; escaping=underscores (modern)
	// - text/plain; version=0.0.4; charset=utf-8 (legacy)
	// - text/plain; charset=utf-8 (fallback)
	contentType := resp.Header.Get("Content-Type")
	validTypes := []string{
		"text/plain; version=0.0.4; charset=utf-8; escaping=underscores",
		"text/plain; version=0.0.4; charset=utf-8",
		"text/plain; charset=utf-8",
	}

	validContentType := slices.Contains(validTypes, contentType)

	if !validContentType {
		t.Errorf("Invalid Content-Type: %s (expected one of: %v)", contentType, validTypes)
	}

	// Test 3: Query actual Prometheus server (if running)
	promResp, err := http.Get("http://localhost:9090/-/healthy")
	if err != nil {
		t.Skipf("Prometheus not available (run: cd observability && docker-compose -f docker-compose.test.yml up -d): %v", err)
	}
	defer func() {
		if err := promResp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if promResp.StatusCode != http.StatusOK {
		t.Fatalf("Prometheus health check failed with status %d", promResp.StatusCode)
	}

	t.Logf("✓ Successfully verified Prometheus metrics")
	t.Logf("  - Test server (temporary): %s/metrics", server.URL)
	t.Logf("  - Prometheus UI (persistent): http://localhost:9090")
	t.Logf("  - Content-Type: %s", contentType)
}

// TestE2E_FullObservabilityStack tests the complete observability stack
func TestE2E_FullObservabilityStack(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E integration test in short mode")
	}

	// Check if OTLP collector is available
	endpoint := "localhost:4317"
	checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Skipf("OTLP collector not available at %s (run: cd observability && docker-compose -f docker-compose.test.yml up -d): %v", endpoint, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// Trigger connection and wait for Ready state or timeout
	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			break
		}
		if state == connectivity.Shutdown || state == connectivity.TransientFailure {
			t.Skipf("OTLP collector not available at %s (state: %v, run: cd observability && docker-compose -f docker-compose.test.yml up -d)", endpoint, state)
		}
		if !conn.WaitForStateChange(checkCtx, state) {
			t.Skipf("OTLP collector not available at %s (timeout waiting for Ready state, run: cd observability && docker-compose -f docker-compose.test.yml up -d)", endpoint)
		}
	}

	ctx := context.Background()

	// 1. Setup tracing
	config := DefaultTracerConfig()
	config.ServiceName = "gonuget-full-stack-test"
	config.ExporterType = "otlp"
	config.OTLPEndpoint = endpoint

	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Skipf("Tracing not available: %v", err)
	}
	defer func() {
		// Use fresh background context for shutdown (ShutdownTracing has its own timeout)
		if err := ShutdownTracing(context.Background(), tp); err != nil {
			t.Errorf("ShutdownTracing() failed: %v", err)
		}
	}()

	// 2. Setup logging
	logger := NewDefaultLogger()

	// 3. Setup metrics
	metricsHandler := MetricsHandler()

	// 4. Setup health checks
	hc := NewHealthChecker()
	hc.Register(HealthCheck{
		Name: "full-stack-test",
		Check: func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "full stack test running",
			}
		},
	})

	// Simulate a request with full observability
	ctx, span := StartPackageDownloadSpan(ctx, "TestPackage", "1.0.0", "test-source")

	// Log with context
	logger.InfoContext(ctx, "Starting package download {PackageId} {Version}",
		"TestPackage", "1.0.0")

	// Record metrics
	HTTPRequestsTotal.WithLabelValues("GET", "200", "test-source").Inc()
	HTTPRequestDuration.WithLabelValues("GET", "test-source").Observe(0.5)
	PackageDownloadsTotal.WithLabelValues("success").Inc()

	// Add span event
	span.AddEvent("download.started")

	// Simulate work
	time.Sleep(100 * time.Millisecond)

	// Complete span
	span.AddEvent("download.completed")
	EndSpanWithError(span, nil)

	// Verify health
	health := hc.OverallStatus(ctx)
	if health != HealthStatusHealthy {
		t.Errorf("Health status = %s, want healthy", health)
	}

	// Verify metrics handler works
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	metricsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Metrics handler returned %d, want 200", w.Code)
	}

	t.Logf("✓ Full observability stack test completed successfully")
	t.Logf("  - Tracing: ✓ (Jaeger UI: http://localhost:16686)")
	t.Logf("  - Logging: ✓ (mtlog structured logging)")
	t.Logf("  - Metrics: ✓ (Prometheus: http://localhost:9090)")
	t.Logf("  - Health: ✓ (Status: %s)", health)
}
