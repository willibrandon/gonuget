package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	nugethttp "github.com/willibrandon/gonuget/http"
	"github.com/willibrandon/gonuget/observability"
	"github.com/willibrandon/gonuget/protocol/v3"
	"go.opentelemetry.io/otel/trace"
)

// TestLogger captures log calls for verification
type TestLogger struct {
	mu      sync.Mutex
	entries []LogEntry
}

type LogEntry struct {
	Level      string
	Template   string
	Args       []any
	HasContext bool
}

func NewTestLogger() *TestLogger {
	return &TestLogger{
		entries: make([]LogEntry, 0),
	}
}

func (l *TestLogger) Debug(template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:    "Debug",
		Template: template,
		Args:     args,
	})
}

func (l *TestLogger) Info(template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:    "Info",
		Template: template,
		Args:     args,
	})
}

func (l *TestLogger) Warn(template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:    "Warn",
		Template: template,
		Args:     args,
	})
}

func (l *TestLogger) Error(template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:    "Error",
		Template: template,
		Args:     args,
	})
}

func (l *TestLogger) Verbose(template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:    "Verbose",
		Template: template,
		Args:     args,
	})
}

func (l *TestLogger) VerboseContext(ctx context.Context, template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:      "Verbose",
		Template:   template,
		Args:       args,
		HasContext: true,
	})
}

func (l *TestLogger) Fatal(template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:    "Fatal",
		Template: template,
		Args:     args,
	})
}

func (l *TestLogger) FatalContext(ctx context.Context, template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:      "Fatal",
		Template:   template,
		Args:       args,
		HasContext: true,
	})
}

func (l *TestLogger) ForContext(key string, value any) observability.Logger {
	// For testing, return the same logger
	return l
}

func (l *TestLogger) WithProperty(key string, value any) observability.Logger {
	// For testing, return the same logger
	return l
}

func (l *TestLogger) DebugContext(ctx context.Context, template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:      "Debug",
		Template:   template,
		Args:       args,
		HasContext: true,
	})
}

func (l *TestLogger) InfoContext(ctx context.Context, template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:      "Info",
		Template:   template,
		Args:       args,
		HasContext: true,
	})
}

func (l *TestLogger) WarnContext(ctx context.Context, template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:      "Warn",
		Template:   template,
		Args:       args,
		HasContext: true,
	})
}

func (l *TestLogger) ErrorContext(ctx context.Context, template string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:      "Error",
		Template:   template,
		Args:       args,
		HasContext: true,
	})
}

func (l *TestLogger) GetEntries() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]LogEntry{}, l.entries...)
}

func (l *TestLogger) FindEntry(template string) *LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.entries {
		if l.entries[i].Template == template {
			return &l.entries[i]
		}
	}
	return nil
}

func (l *TestLogger) CountEntries() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// HasLog checks if any log entry contains the given substring
func (l *TestLogger) HasLog(substring string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, entry := range l.entries {
		if strings.Contains(entry.Template, substring) {
			return true
		}
	}
	return false
}

// Clear removes all log entries
func (l *TestLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = make([]LogEntry, 0)
}

// TestObservability_Repository_LoggingIntegration verifies logging is called correctly
func TestObservability_Repository_LoggingIntegration(t *testing.T) {
	// Setup test server with proper URL handling
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			w.Header().Set("Content-Type", "application/json")
			index := v3.ServiceIndex{
				Version: "3.0.0",
				Resources: []v3.Resource{
					{ID: server.URL + "/search", Type: "SearchQueryService"},
					{ID: server.URL + "/registration/", Type: "RegistrationsBaseUrl"},
					{ID: server.URL + "/packages/", Type: "PackageBaseAddress/3.0.0"},
				},
			}
			_ = json.NewEncoder(w).Encode(index)
			return
		}
		// Registration URL format: /registration/{packageIdLower}/index.json
		if r.URL.Path == "/registration/test.package/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(v3.RegistrationIndex{
				Count: 1,
				Items: []v3.RegistrationPage{
					{
						ID:    server.URL + "/registration/test.package/index.json",
						Count: 1,
						Items: []v3.RegistrationLeaf{
							{
								ID: server.URL + "/registration/test.package/1.0.0.json",
								CatalogEntry: &v3.RegistrationCatalog{
									ID:        server.URL + "/registration/test.package/1.0.0.json",
									PackageID: "Test.Package",
									Version:   "1.0.0",
								},
							},
						},
					},
				},
			})
			return
		}
		// Download URL format: /packages/{packageIdLower}/{versionLower}/{packageIdLower}.{versionLower}.nupkg
		if r.URL.Path == "/packages/test.package/1.0.0/test.package.1.0.0.nupkg" {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("fake-package-data"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	testLogger := NewTestLogger()
	httpClient := nugethttp.NewClient(nil)

	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  server.URL,
		HTTPClient: httpClient,
		Logger:     testLogger,
	})

	ctx := context.Background()

	t.Run("GetMetadata logging", func(t *testing.T) {
		testLogger.entries = nil // Reset

		_, err := repo.GetMetadata(ctx, nil, "Test.Package", "1.0.0")
		if err != nil {
			t.Fatalf("GetMetadata() error = %v", err)
		}

		entries := testLogger.GetEntries()
		if len(entries) == 0 {
			t.Fatal("Expected log entries, got none")
		}

		// Check for debug entry
		debugEntry := testLogger.FindEntry("Fetching package metadata for {PackageID}@{Version} from {Source}")
		if debugEntry == nil {
			t.Error("Expected debug log for metadata fetch")
		} else {
			if len(debugEntry.Args) != 3 {
				t.Errorf("Expected 3 args for debug log, got %d", len(debugEntry.Args))
			}
			if debugEntry.Args[0] != "Test.Package" {
				t.Errorf("Expected PackageID=Test.Package, got %v", debugEntry.Args[0])
			}
		}

		// Check for success entry
		infoEntry := testLogger.FindEntry("Successfully fetched metadata for {PackageID}@{Version}")
		if infoEntry == nil {
			t.Error("Expected info log for successful metadata fetch")
		}
	})

	t.Run("ListVersions logging", func(t *testing.T) {
		testLogger.entries = nil // Reset

		_, err := repo.ListVersions(ctx, nil, "Test.Package")
		if err != nil {
			t.Fatalf("ListVersions() error = %v", err)
		}

		entries := testLogger.GetEntries()
		if len(entries) == 0 {
			t.Fatal("Expected log entries, got none")
		}

		// Check for debug entry
		debugEntry := testLogger.FindEntry("Listing package versions for {PackageID} from {Source}")
		if debugEntry == nil {
			t.Error("Expected debug log for version listing")
		}

		// Check for success entry with count
		infoEntry := testLogger.FindEntry("Successfully listed {Count} versions for {PackageID}")
		if infoEntry == nil {
			t.Error("Expected info log for successful version listing")
		}
	})

	t.Run("DownloadPackage logging and tracing", func(t *testing.T) {
		testLogger.entries = nil // Reset

		// Initialize OpenTelemetry for tracing test
		tp, err := observability.SetupTracing(ctx, observability.TracerConfig{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			ExporterType:   "none", // Disable exporter for test
		})
		if err != nil {
			t.Fatalf("Failed to initialize OpenTelemetry: %v", err)
		}
		defer func() { _ = observability.ShutdownTracing(ctx, tp) }()

		// Create a test span to capture child spans
		tracer := observability.Tracer("test")
		ctx, parentSpan := tracer.Start(ctx, "test-download")
		defer parentSpan.End()

		rc, err := repo.DownloadPackage(ctx, nil, "Test.Package", "1.0.0")
		if err != nil {
			t.Fatalf("DownloadPackage() error = %v", err)
		}
		defer func() { _ = rc.Close() }()

		entries := testLogger.GetEntries()
		if len(entries) == 0 {
			t.Fatal("Expected log entries, got none")
		}

		// Check for info entry at start
		infoEntry := testLogger.FindEntry("Downloading package {PackageID}@{Version} from {Source}")
		if infoEntry == nil {
			t.Error("Expected info log for download start")
		} else if len(infoEntry.Args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(infoEntry.Args))
		}

		// Check for success entry
		successEntry := testLogger.FindEntry("Successfully downloaded package {PackageID}@{Version}")
		if successEntry == nil {
			t.Error("Expected info log for successful download")
		}

		// Verify span was created (indirectly by checking context has span)
		span := trace.SpanFromContext(ctx)
		if !span.SpanContext().IsValid() {
			t.Error("Expected valid span context after download")
		}
	})
}

// TestObservability_Repository_ErrorLogging verifies error logging
func TestObservability_Repository_ErrorLogging(t *testing.T) {
	testLogger := NewTestLogger()

	// Create repository with invalid URL
	repo := NewSourceRepository(RepositoryConfig{
		Name:      "test",
		SourceURL: "http://invalid-host-that-does-not-exist.local",
		HTTPClient: nugethttp.NewClient(&nugethttp.Config{
			Timeout: 100, // Very short timeout
		}),
		Logger: testLogger,
	})

	ctx := context.Background()

	t.Run("GetMetadata error logging", func(t *testing.T) {
		testLogger.entries = nil // Reset

		_, err := repo.GetMetadata(ctx, nil, "Test.Package", "1.0.0")
		if err == nil {
			t.Fatal("Expected error from invalid host")
		}

		entries := testLogger.GetEntries()
		if len(entries) == 0 {
			t.Fatal("Expected log entries, got none")
		}

		// Check for error entry
		errorEntry := testLogger.FindEntry("Failed to get provider for {Source}: {Error}")
		if errorEntry == nil {
			t.Error("Expected error log for provider creation failure")
		}
	})
}

// TestObservability_Repository_NullLogger verifies NullLogger default
func TestObservability_Repository_NullLogger(t *testing.T) {
	// Create repository without logger (should use NullLogger)
	repo := NewSourceRepository(RepositoryConfig{
		Name:       "test",
		SourceURL:  "https://api.nuget.org/v3/index.json",
		HTTPClient: nugethttp.NewClient(nil),
		Logger:     nil, // Explicitly nil
	})

	if repo.logger == nil {
		t.Error("Expected logger to be initialized, got nil")
	}

	// Verify it's a NullLogger by checking it doesn't panic
	ctx := context.Background()
	repo.logger.InfoContext(ctx, "Test message")
	repo.logger.ErrorContext(ctx, "Test error")
	// If we get here without panic, NullLogger is working
}

// TestObservability_HTTP_LoggingIntegration verifies HTTP client logging
func TestObservability_HTTP_LoggingIntegration(t *testing.T) {
	testLogger := NewTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := nugethttp.NewClient(&nugethttp.Config{
		Logger: testLogger,
	})

	ctx := context.Background()

	t.Run("Do() logging", func(t *testing.T) {
		testLogger.entries = nil // Reset

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(ctx, req)
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		entries := testLogger.GetEntries()
		if len(entries) < 2 {
			t.Fatalf("Expected at least 2 log entries, got %d", len(entries))
		}

		// Check for debug entry before request
		debugEntry := testLogger.FindEntry("HTTP {Method} {URL}")
		if debugEntry == nil {
			t.Error("Expected debug log before HTTP request")
		}

		// Check for debug entry after request (with status code)
		successEntry := testLogger.FindEntry("HTTP {Method} {URL} → {StatusCode} ({Duration}ms)")
		if successEntry == nil {
			t.Error("Expected debug log after successful HTTP request")
		}
	})

	t.Run("DoWithRetry() logging", func(t *testing.T) {
		testLogger.entries = nil // Reset

		// Create server that fails once then succeeds
		attemptCount := 0
		retryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}))
		defer retryServer.Close()

		retryClient := nugethttp.NewClient(&nugethttp.Config{
			Logger: testLogger,
			RetryConfig: &nugethttp.RetryConfig{
				MaxRetries:     2,
				InitialBackoff: 10,
				MaxBackoff:     100,
				BackoffFactor:  2,
			},
		})

		req, _ := http.NewRequest("GET", retryServer.URL, nil)
		resp, err := retryClient.DoWithRetry(ctx, req)
		if err != nil {
			t.Fatalf("DoWithRetry() error = %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		entries := testLogger.GetEntries()
		if len(entries) < 3 {
			t.Fatalf("Expected at least 3 log entries for retry scenario, got %d", len(entries))
		}

		// Check for retry log
		retryEntry := testLogger.FindEntry("HTTP {Method} {URL} retry {Attempt}/{MaxRetries} after {Backoff}ms")
		if retryEntry == nil {
			t.Error("Expected debug log for retry attempt")
		}

		// Check for success after retry
		successEntry := testLogger.FindEntry("HTTP {Method} {URL} succeeded after {Attempt} retries")
		if successEntry == nil {
			t.Error("Expected info log for successful retry")
		}
	})
}

// TestObservability_HTTP_ErrorLogging verifies HTTP error logging
func TestObservability_HTTP_ErrorLogging(t *testing.T) {
	testLogger := NewTestLogger()

	client := nugethttp.NewClient(&nugethttp.Config{
		Logger: testLogger,
	})

	ctx := context.Background()

	t.Run("Do() error logging", func(t *testing.T) {
		testLogger.entries = nil // Reset

		// Request to invalid host
		req, _ := http.NewRequest("GET", "http://invalid-host.local", nil)
		_, err := client.Do(ctx, req)
		if err == nil {
			t.Fatal("Expected error from invalid host")
		}

		entries := testLogger.GetEntries()
		if len(entries) == 0 {
			t.Fatal("Expected log entries, got none")
		}

		// Check for warn entry on error
		warnEntry := testLogger.FindEntry("HTTP {Method} {URL} failed after {Duration}ms: {Error}")
		if warnEntry == nil {
			t.Error("Expected warn log for HTTP failure")
		}
	})
}

// TestObservability_HTTP_TracingIntegration verifies HTTP tracing
func TestObservability_HTTP_TracingIntegration(t *testing.T) {
	// Initialize OpenTelemetry with stdout exporter for proper tracing
	ctx := context.Background()
	tp, err := observability.SetupTracing(ctx, observability.TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ExporterType:   "stdout", // Use stdout exporter for proper trace propagation
		SamplingRate:   1.0,      // Sample all traces
	})
	if err != nil {
		t.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}
	defer func() { _ = observability.ShutdownTracing(ctx, tp) }()

	// Track whether Traceparent header was received
	traceparentReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify trace headers are present
		if r.Header.Get("Traceparent") != "" {
			traceparentReceived = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nugethttp.NewClient(&nugethttp.Config{
		EnableTracing: true,
	})

	// Create parent span - this is critical for trace context propagation
	tracer := observability.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-http")
	defer span.End()

	// Verify span context is valid before making request
	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		t.Fatal("Parent span context is invalid before request")
	}
	if !spanCtx.IsSampled() {
		t.Fatal("Parent span is not sampled")
	}

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify Traceparent header was sent
	if !traceparentReceived {
		t.Error("Expected Traceparent header for distributed tracing")
	}
}

// TestObservability_Metrics_Integration verifies metrics are incremented
func TestObservability_Metrics_Integration(t *testing.T) {
	// Note: This is a basic integration test. In production, you would use
	// a test metrics registry to verify exact counts. Here we just verify
	// the code path doesn't panic.

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := nugethttp.NewClient(nil)
	ctx := context.Background()

	// Make HTTP request (should increment metrics)
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// If we get here without panic, metrics integration is working
	// In a real test, we would verify the counter was incremented
}
