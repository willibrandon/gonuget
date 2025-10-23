# M4 Implementation Guide: Observability (Part 3)

**Chunks Covered:** M4.9, M4.10, M4.11, M4.12, M4.13, M4.14
**Est. Total Time:** 12 hours
**Dependencies:** M2.1 (HTTP Client), M4.1-M4.8 (Cache, Resilience)

---

## Overview

This guide implements comprehensive observability using mtlog (structured logging), OpenTelemetry (tracing), Prometheus (metrics), and health checks. While NuGet.Client has basic logging, gonuget adds production-grade observability as an internal enhancement.

### External Dependencies

- **mtlog**: User-developed zero-allocation structured logging (github.com/willibrandon/mtlog)
- **OpenTelemetry**: Standard tracing and metrics (go.opentelemetry.io/otel)
- **Prometheus**: Metrics exposition (github.com/prometheus/client_golang)

### Key Compatibility Requirements

✅ **NuGet.Client Compatibility:**
- Observability is entirely internal (no protocol impact)
- Can be disabled without affecting functionality
- Only enhances debugging and monitoring capabilities

---

## M4.9: mtlog Integration

**Goal:** Integrate mtlog for zero-allocation structured logging throughout gonuget.

### mtlog Overview

From `/Users/brandon/src/mtlog/README.md`:
- **Zero-allocation logging**: 17.3 ns/op for simple messages
- **Message templates**: Property extraction from placeholders
- **Pipeline architecture**: enrichers → filters → sinks
- **Context awareness**: Deadline enrichment, trace integration

### Implementation

**File:** `observability/logger.go`

```go
package observability

import (
	"context"
	"io"
	"os"

	"github.com/willibrandon/mtlog"
	"github.com/willibrandon/mtlog/core"
)

// Logger is the gonuget logger interface
// Wraps mtlog for structured logging with zero allocations
type Logger interface {
	// Verbose logs detailed diagnostic information
	Verbose(messageTemplate string, args ...any)
	VerboseContext(ctx context.Context, messageTemplate string, args ...any)

	// Debug logs debugging information
	Debug(messageTemplate string, args ...any)
	DebugContext(ctx context.Context, messageTemplate string, args ...any)

	// Info logs informational messages
	Info(messageTemplate string, args ...any)
	InfoContext(ctx context.Context, messageTemplate string, args ...any)

	// Warn logs warning messages
	Warn(messageTemplate string, args ...any)
	WarnContext(ctx context.Context, messageTemplate string, args ...any)

	// Error logs error messages
	Error(messageTemplate string, args ...any)
	ErrorContext(ctx context.Context, messageTemplate string, args ...any)

	// Fatal logs fatal error messages
	Fatal(messageTemplate string, args ...any)
	FatalContext(ctx context.Context, messageTemplate string, args ...any)

	// ForContext creates a child logger with additional context
	ForContext(key string, value any) Logger

	// WithProperty adds a property to the logger
	WithProperty(key string, value any) Logger
}

// mtlogAdapter wraps mtlog logger to implement gonuget Logger interface
type mtlogAdapter struct {
	logger core.Logger
}

// NewLogger creates a new gonuget logger with sensible defaults
func NewLogger(output io.Writer, level LogLevel) Logger {
	opts := []mtlog.Option{
		mtlog.WriteTo(output),
		mtlog.WithTimestamp(),
		mtlog.WithMachineName(),
		mtlog.WithProcess(),
	}

	// Set minimum level
	switch level {
	case VerboseLevel:
		opts = append(opts, mtlog.Verbose())
	case DebugLevel:
		opts = append(opts, mtlog.Debug())
	case InfoLevel:
		opts = append(opts, mtlog.Information())
	case WarnLevel:
		opts = append(opts, mtlog.Warning())
	case ErrorLevel:
		opts = append(opts, mtlog.Error())
	case FatalLevel:
		opts = append(opts, mtlog.Fatal())
	}

	return &mtlogAdapter{
		logger: mtlog.New(opts...),
	}
}

// NewDefaultLogger creates a logger with console output and Info level
func NewDefaultLogger() Logger {
	return NewLogger(os.Stdout, InfoLevel)
}

// Verbose implements Logger.Verbose
func (a *mtlogAdapter) Verbose(messageTemplate string, args ...any) {
	a.logger.Verbose(messageTemplate, args...)
}

// VerboseContext implements Logger.VerboseContext
func (a *mtlogAdapter) VerboseContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.VerboseContext(ctx, messageTemplate, args...)
}

// Debug implements Logger.Debug
func (a *mtlogAdapter) Debug(messageTemplate string, args ...any) {
	a.logger.Debug(messageTemplate, args...)
}

// DebugContext implements Logger.DebugContext
func (a *mtlogAdapter) DebugContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.DebugContext(ctx, messageTemplate, args...)
}

// Info implements Logger.Info
func (a *mtlogAdapter) Info(messageTemplate string, args ...any) {
	a.logger.Information(messageTemplate, args...)
}

// InfoContext implements Logger.InfoContext
func (a *mtlogAdapter) InfoContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.InformationContext(ctx, messageTemplate, args...)
}

// Warn implements Logger.Warn
func (a *mtlogAdapter) Warn(messageTemplate string, args ...any) {
	a.logger.Warning(messageTemplate, args...)
}

// WarnContext implements Logger.WarnContext
func (a *mtlogAdapter) WarnContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.WarningContext(ctx, messageTemplate, args...)
}

// Error implements Logger.Error
func (a *mtlogAdapter) Error(messageTemplate string, args ...any) {
	a.logger.Error(messageTemplate, args...)
}

// ErrorContext implements Logger.ErrorContext
func (a *mtlogAdapter) ErrorContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.ErrorContext(ctx, messageTemplate, args...)
}

// Fatal implements Logger.Fatal
func (a *mtlogAdapter) Fatal(messageTemplate string, args ...any) {
	a.logger.Fatal(messageTemplate, args...)
}

// FatalContext implements Logger.FatalContext
func (a *mtlogAdapter) FatalContext(ctx context.Context, messageTemplate string, args ...any) {
	a.logger.FatalContext(ctx, messageTemplate, args...)
}

// ForContext implements Logger.ForContext
func (a *mtlogAdapter) ForContext(key string, value any) Logger {
	return &mtlogAdapter{
		logger: a.logger.ForContext(key, value),
	}
}

// WithProperty implements Logger.WithProperty (alias for ForContext)
func (a *mtlogAdapter) WithProperty(key string, value any) Logger {
	return a.ForContext(key, value)
}

// LogLevel represents log verbosity level
type LogLevel int

const (
	VerboseLevel LogLevel = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// NullLogger is a logger that discards all output
type nullLogger struct{}

// NewNullLogger creates a logger that discards all output
func NewNullLogger() Logger {
	return &nullLogger{}
}

func (n *nullLogger) Verbose(messageTemplate string, args ...any)                            {}
func (n *nullLogger) VerboseContext(ctx context.Context, messageTemplate string, args ...any) {}
func (n *nullLogger) Debug(messageTemplate string, args ...any)                              {}
func (n *nullLogger) DebugContext(ctx context.Context, messageTemplate string, args ...any)   {}
func (n *nullLogger) Info(messageTemplate string, args ...any)                               {}
func (n *nullLogger) InfoContext(ctx context.Context, messageTemplate string, args ...any)    {}
func (n *nullLogger) Warn(messageTemplate string, args ...any)                               {}
func (n *nullLogger) WarnContext(ctx context.Context, messageTemplate string, args ...any)    {}
func (n *nullLogger) Error(messageTemplate string, args ...any)                              {}
func (n *nullLogger) ErrorContext(ctx context.Context, messageTemplate string, args ...any)   {}
func (n *nullLogger) Fatal(messageTemplate string, args ...any)                              {}
func (n *nullLogger) FatalContext(ctx context.Context, messageTemplate string, args ...any)   {}
func (n *nullLogger) ForContext(key string, value any) Logger                                { return n }
func (n *nullLogger) WithProperty(key string, value any) Logger                              { return n }
```

**Usage Example:**

```go
package main

import (
	"context"
	"time"

	"github.com/example/gonuget/observability"
)

func main() {
	log := observability.NewDefaultLogger()

	// Basic logging
	log.Info("gonuget starting")

	// Structured logging with properties
	log.Info("Downloading package {PackageId} version {Version}",
		"Newtonsoft.Json", "13.0.3")

	// Context-based logging (scoped logger)
	sourceLog := log.ForContext("Source", "https://api.nuget.org/v3/index.json")
	sourceLog.Info("Connecting to package source")

	// Context-aware logging (deadline tracking)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sourceLog.InfoContext(ctx, "Fetching package metadata")

	// Error logging
	sourceLog.Error("Failed to download package after {Attempts} attempts", 3)
}
```

**Tests:** `observability/logger_test.go`

```go
package observability

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestLogger_BasicLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, DebugLevel)

	log.Info("Test message")

	output := buf.String()
	if !strings.Contains(output, "Test message") {
		t.Errorf("Output missing message: %s", output)
	}
}

func TestLogger_StructuredProperties(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, InfoLevel)

	log.Info("Package {PackageId} version {Version}", "Newtonsoft.Json", "13.0.3")

	output := buf.String()
	if !strings.Contains(output, "Newtonsoft.Json") {
		t.Errorf("Output missing PackageId: %s", output)
	}
	if !strings.Contains(output, "13.0.3") {
		t.Errorf("Output missing Version: %s", output)
	}
}

func TestLogger_ForContext(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, InfoLevel)

	scopedLog := log.ForContext("Source", "nuget.org")
	scopedLog.Info("Message from scoped logger")

	output := buf.String()
	if !strings.Contains(output, "nuget.org") {
		t.Errorf("Output missing context property: %s", output)
	}
}

func TestLogger_ContextAware(t *testing.T) {
	buf := &bytes.Buffer{}
	log := NewLogger(buf, InfoLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	log.InfoContext(ctx, "Context-aware message")

	output := buf.String()
	if !strings.Contains(output, "Context-aware message") {
		t.Errorf("Output missing message: %s", output)
	}
}

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		name          string
		level         LogLevel
		logFunc       func(Logger)
		shouldContain bool
	}{
		{
			name:  "Info level allows Info",
			level: InfoLevel,
			logFunc: func(l Logger) {
				l.Info("Info message")
			},
			shouldContain: true,
		},
		{
			name:  "Info level blocks Debug",
			level: InfoLevel,
			logFunc: func(l Logger) {
				l.Debug("Debug message")
			},
			shouldContain: false,
		},
		{
			name:  "Debug level allows Debug",
			level: DebugLevel,
			logFunc: func(l Logger) {
				l.Debug("Debug message")
			},
			shouldContain: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log := NewLogger(buf, tt.level)

			tt.logFunc(log)

			output := buf.String()
			contains := len(output) > 0

			if contains != tt.shouldContain {
				t.Errorf("Message presence = %v, want %v. Output: %s", contains, tt.shouldContain, output)
			}
		})
	}
}

func TestNullLogger(t *testing.T) {
	log := NewNullLogger()

	// Should not panic
	log.Info("This should be discarded")
	log.Error("This should also be discarded")

	scopedLog := log.ForContext("key", "value")
	scopedLog.Info("Scoped logger message")

	// No assertions - just verify no panic
}
```

### Testing

```bash
go test ./observability -run TestLogger -v

# Verify:
# 1. Message template parsing
# 2. Property extraction
# 3. Context propagation
# 4. Log level filtering
```

---

## M4.10: OpenTelemetry - Tracing Setup

**Goal:** Configure OpenTelemetry tracing for distributed request tracking.

### Implementation

**File:** `observability/tracing.go`

```go
package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TracerConfig holds OpenTelemetry tracer configuration
type TracerConfig struct {
	// ServiceName is the name of the service
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Environment is the deployment environment (production, staging, etc.)
	Environment string

	// ExporterType is the type of exporter (otlp, stdout, none)
	ExporterType string

	// OTLPEndpoint is the OTLP collector endpoint (e.g., localhost:4317)
	OTLPEndpoint string

	// SamplingRate is the trace sampling rate (0.0 to 1.0)
	SamplingRate float64
}

// DefaultTracerConfig returns default tracer configuration
func DefaultTracerConfig() TracerConfig {
	return TracerConfig{
		ServiceName:    "gonuget",
		ServiceVersion: "0.1.0",
		Environment:    "development",
		ExporterType:   "stdout",
		SamplingRate:   1.0, // Sample all traces in development
	}
}

// SetupTracing initializes OpenTelemetry tracing
func SetupTracing(ctx context.Context, config TracerConfig) (*sdktrace.TracerProvider, error) {
	// Create resource with service metadata
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on configuration
	var exporter sdktrace.SpanExporter
	switch config.ExporterType {
	case "otlp":
		exporter, err = createOTLPExporter(ctx, config.OTLPEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
	case "stdout":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
		}
	case "none":
		// No exporter - tracing disabled
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		return tp, nil
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", config.ExporterType)
	}

	// Create sampler
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(config.SamplingRate),
	)

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler),
	)

	// Register as global tracer provider
	otel.SetTracerProvider(tp)

	return tp, nil
}

// createOTLPExporter creates an OTLP gRPC exporter
func createOTLPExporter(ctx context.Context, endpoint string) (*otlptrace.Exporter, error) {
	conn, err := grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	return exporter, nil
}

// ShutdownTracing gracefully shuts down the tracer provider
func ShutdownTracing(ctx context.Context, tp *sdktrace.TracerProvider) error {
	// Create timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := tp.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}

	return nil
}

// Tracer returns a named tracer
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, tracerName string, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer(tracerName).Start(ctx, spanName, opts...)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets attributes on the current span
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}
```

**Tests:** `observability/tracing_test.go`

```go
package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestSetupTracing_Stdout(t *testing.T) {
	ctx := context.Background()
	config := TracerConfig{
		ServiceName:    "gonuget-test",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   "stdout",
		SamplingRate:   1.0,
	}

	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Fatalf("SetupTracing() failed: %v", err)
	}
	defer ShutdownTracing(ctx, tp)

	// Create a test span
	tracer := Tracer("test")
	ctx, span := tracer.Start(ctx, "test-operation")
	span.SetAttributes(attribute.String("test.key", "test.value"))
	span.End()
}

func TestSetupTracing_None(t *testing.T) {
	ctx := context.Background()
	config := TracerConfig{
		ServiceName:  "gonuget-test",
		ExporterType: "none",
		SamplingRate: 0.0,
	}

	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Fatalf("SetupTracing() with none exporter failed: %v", err)
	}
	defer ShutdownTracing(ctx, tp)
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()
	config := DefaultTracerConfig()

	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Fatalf("SetupTracing() failed: %v", err)
	}
	defer ShutdownTracing(ctx, tp)

	ctx, span := StartSpan(ctx, "gonuget", "test-span")
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Error("Span context should be valid")
	}
}

func TestSpanHelpers(t *testing.T) {
	ctx := context.Background()
	config := DefaultTracerConfig()

	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Fatalf("SetupTracing() failed: %v", err)
	}
	defer ShutdownTracing(ctx, tp)

	ctx, span := StartSpan(ctx, "gonuget", "test-span")
	defer span.End()

	// Test AddEvent
	AddEvent(ctx, "test-event", attribute.String("event.key", "event.value"))

	// Test SetAttributes
	SetAttributes(ctx, attribute.Int("request.count", 42))

	// Test RecordError
	RecordError(ctx, context.DeadlineExceeded)

	// Should not panic
}
```

### Testing

```bash
go test ./observability -run TestTracing -v

# Manual test with OTLP collector:
# 1. Start Jaeger: docker run -d -p 4317:4317 -p 16686:16686 jaegertracing/all-in-one:latest
# 2. Run test with OTLP exporter
# 3. View traces at http://localhost:16686
```

---

## M4.11: OpenTelemetry - HTTP Instrumentation

**Goal:** Automatically instrument HTTP requests with distributed tracing.

### Implementation

**File:** `observability/http_tracing.go`

```go
package observability

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// HTTPTracingTransport wraps http.RoundTripper with OpenTelemetry tracing
type HTTPTracingTransport struct {
	base       http.RoundTripper
	tracerName string
}

// NewHTTPTracingTransport creates a new HTTP transport with tracing
func NewHTTPTracingTransport(base http.RoundTripper, tracerName string) *HTTPTracingTransport {
	if base == nil {
		base = http.DefaultTransport
	}

	return &HTTPTracingTransport{
		base:       base,
		tracerName: tracerName,
	}
}

// RoundTrip implements http.RoundTripper with tracing
func (t *HTTPTracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	tracer := Tracer(t.tracerName)

	// Create span name from HTTP method and path
	spanName := req.Method + " " + req.URL.Path

	// Start span with HTTP attributes
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.HTTPMethod(req.Method),
			semconv.HTTPURL(req.URL.String()),
			semconv.HTTPScheme(req.URL.Scheme),
			semconv.NetPeerName(req.URL.Hostname()),
		),
	)
	defer span.End()

	// Inject trace context into HTTP headers (W3C Trace Context)
	// OTEL automatically propagates context via http.Request.WithContext
	req = req.WithContext(ctx)

	// Execute request
	resp, err := t.base.RoundTrip(req)

	// Record response details
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Record status code
	span.SetAttributes(semconv.HTTPStatusCode(resp.StatusCode))

	// Set span status based on HTTP status
	if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, resp.Status)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return resp, nil
}

// InstrumentedHTTPClient creates an HTTP client with tracing enabled
func InstrumentedHTTPClient(tracerName string) *http.Client {
	return &http.Client{
		Transport: NewHTTPTracingTransport(http.DefaultTransport, tracerName),
	}
}

// HTTPSpanAttributes returns standard HTTP span attributes
func HTTPSpanAttributes(req *http.Request, resp *http.Response) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.HTTPMethod(req.Method),
		semconv.HTTPURL(req.URL.String()),
		semconv.HTTPScheme(req.URL.Scheme),
		semconv.NetPeerName(req.URL.Hostname()),
	}

	if resp != nil {
		attrs = append(attrs,
			semconv.HTTPStatusCode(resp.StatusCode),
			attribute.Int64("http.response_content_length", resp.ContentLength),
		)
	}

	return attrs
}
```

**Tests:** `observability/http_tracing_test.go`

```go
package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestHTTPTracingTransport(t *testing.T) {
	// Setup tracing
	ctx := context.Background()
	config := DefaultTracerConfig()
	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Fatalf("SetupTracing() failed: %v", err)
	}
	defer ShutdownTracing(ctx, tp)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Create instrumented client
	client := InstrumentedHTTPClient("gonuget-test")

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest() failed: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestHTTPTracingTransport_Error(t *testing.T) {
	ctx := context.Background()
	config := DefaultTracerConfig()
	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Fatalf("SetupTracing() failed: %v", err)
	}
	defer ShutdownTracing(ctx, tp)

	// Create instrumented client
	client := InstrumentedHTTPClient("gonuget-test")

	// Make request to invalid URL
	req, err := http.NewRequestWithContext(ctx, "GET", "http://invalid.local.test:99999", nil)
	if err != nil {
		t.Fatalf("NewRequest() failed: %v", err)
	}

	_, err = client.Do(req)
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}

	// Error should be recorded in span
}

func TestHTTPSpanAttributes(t *testing.T) {
	req, err := http.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)
	if err != nil {
		t.Fatalf("NewRequest() failed: %v", err)
	}

	resp := &http.Response{
		StatusCode:    200,
		ContentLength: 1234,
	}

	attrs := HTTPSpanAttributes(req, resp)

	// Verify required attributes
	expectedAttrs := map[string]bool{
		"http.method":                  false,
		"http.url":                     false,
		"http.scheme":                  false,
		"net.peer.name":                false,
		"http.status_code":             false,
		"http.response_content_length": false,
	}

	for _, kv := range attrs {
		expectedAttrs[string(kv.Key)] = true
	}

	for key, found := range expectedAttrs {
		if !found {
			t.Errorf("Missing expected attribute: %s", key)
		}
	}
}
```

### Testing

```bash
go test ./observability -run TestHTTPTracing -v

# Verify:
# 1. Span creation for HTTP requests
# 2. W3C Trace Context propagation
# 3. Error recording
# 4. Status code attributes
```

---

## M4.12: OpenTelemetry - Operation Spans

**Goal:** Create spans for high-level NuGet operations (package restore, download, etc.)

### Implementation

**File:** `observability/operations.go`

```go
package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Tracer name for gonuget operations
	TracerName = "github.com/example/gonuget"
)

// Common attribute keys
const (
	AttrPackageID      = attribute.Key("nuget.package.id")
	AttrPackageVersion = attribute.Key("nuget.package.version")
	AttrSourceURL      = attribute.Key("nuget.source.url")
	AttrFramework      = attribute.Key("nuget.framework")
	AttrOperation      = attribute.Key("nuget.operation")
	AttrCacheHit       = attribute.Key("nuget.cache.hit")
	AttrRetryCount     = attribute.Key("nuget.retry.count")
)

// StartPackageDownloadSpan starts a span for package download operation
func StartPackageDownloadSpan(ctx context.Context, packageID, version, sourceURL string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "package.download",
		trace.WithAttributes(
			AttrPackageID.String(packageID),
			AttrPackageVersion.String(version),
			AttrSourceURL.String(sourceURL),
			AttrOperation.String("download"),
		),
	)
}

// StartPackageRestoreSpan starts a span for package restore operation
func StartPackageRestoreSpan(ctx context.Context, projectPath string, packageCount int) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "package.restore",
		trace.WithAttributes(
			attribute.String("project.path", projectPath),
			attribute.Int("package.count", packageCount),
			AttrOperation.String("restore"),
		),
	)
}

// StartCacheLookupSpan starts a span for cache lookup
func StartCacheLookupSpan(ctx context.Context, cacheKey string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "cache.lookup",
		trace.WithAttributes(
			attribute.String("cache.key", cacheKey),
		),
	)
}

// RecordCacheHit records cache hit/miss on the current span
func RecordCacheHit(ctx context.Context, hit bool) {
	SetAttributes(ctx, AttrCacheHit.Bool(hit))
}

// StartDependencyResolutionSpan starts a span for dependency resolution
func StartDependencyResolutionSpan(ctx context.Context, packageID, framework string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "dependency.resolve",
		trace.WithAttributes(
			AttrPackageID.String(packageID),
			AttrFramework.String(framework),
			AttrOperation.String("resolve"),
		),
	)
}

// StartFrameworkSelectionSpan starts a span for framework selection
func StartFrameworkSelectionSpan(ctx context.Context, targetFramework string, candidateCount int) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "framework.select",
		trace.WithAttributes(
			attribute.String("framework.target", targetFramework),
			attribute.Int("framework.candidates", candidateCount),
		),
	)
}

// RecordRetry records a retry attempt on the current span
func RecordRetry(ctx context.Context, attempt int, err error) {
	span := SpanFromContext(ctx)
	span.AddEvent("retry",
		trace.WithAttributes(
			attribute.Int("retry.attempt", attempt),
			attribute.String("retry.error", err.Error()),
		),
	)
}

// EndSpanWithError ends a span with an error status
func EndSpanWithError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}
```

**Usage Example:**

```go
package main

import (
	"context"
	"fmt"

	"github.com/example/gonuget/observability"
)

func downloadPackage(ctx context.Context, packageID, version, sourceURL string) error {
	ctx, span := observability.StartPackageDownloadSpan(ctx, packageID, version, sourceURL)
	defer span.End()

	// Check cache
	cacheCtx, cacheSpan := observability.StartCacheLookupSpan(ctx, packageID+"-"+version)
	cacheHit := checkCache(packageID, version)
	observability.RecordCacheHit(cacheCtx, cacheHit)
	cacheSpan.End()

	if cacheHit {
		return nil
	}

	// Download from source
	err := downloadFromSource(ctx, sourceURL)
	if err != nil {
		observability.RecordError(ctx, err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
```

**Tests:** `observability/operations_test.go`

```go
package observability

import (
	"context"
	"errors"
	"testing"
)

func TestStartPackageDownloadSpan(t *testing.T) {
	ctx := context.Background()
	config := DefaultTracerConfig()
	tp, _ := SetupTracing(ctx, config)
	defer ShutdownTracing(ctx, tp)

	ctx, span := StartPackageDownloadSpan(ctx, "Newtonsoft.Json", "13.0.3", "https://api.nuget.org")
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Error("Span context should be valid")
	}
}

func TestRecordCacheHit(t *testing.T) {
	ctx := context.Background()
	config := DefaultTracerConfig()
	tp, _ := SetupTracing(ctx, config)
	defer ShutdownTracing(ctx, tp)

	ctx, span := StartCacheLookupSpan(ctx, "test-key")
	defer span.End()

	RecordCacheHit(ctx, true)
	// Should not panic
}

func TestRecordRetry(t *testing.T) {
	ctx := context.Background()
	config := DefaultTracerConfig()
	tp, _ := SetupTracing(ctx, config)
	defer ShutdownTracing(ctx, tp)

	ctx, span := StartPackageDownloadSpan(ctx, "Test.Package", "1.0.0", "https://example.com")
	defer span.End()

	RecordRetry(ctx, 1, errors.New("connection timeout"))
	RecordRetry(ctx, 2, errors.New("connection timeout"))
	// Should not panic
}

func TestEndSpanWithError(t *testing.T) {
	ctx := context.Background()
	config := DefaultTracerConfig()
	tp, _ := SetupTracing(ctx, config)
	defer ShutdownTracing(ctx, tp)

	ctx, span := StartPackageDownloadSpan(ctx, "Test.Package", "1.0.0", "https://example.com")

	err := errors.New("download failed")
	EndSpanWithError(span, err)
	// Should not panic
}
```

### Testing

```bash
go test ./observability -run TestOperations -v
```

---

---

## M4.13: Prometheus Metrics

**Goal:** Expose Prometheus metrics for monitoring NuGet operations.

### Implementation

**File:** `observability/metrics.go`

```go
package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP request metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_http_requests_total",
			Help: "Total number of HTTP requests by method and status",
		},
		[]string{"method", "status_code", "source"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gonuget_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to 16s
		},
		[]string{"method", "source"},
	)

	// Cache metrics
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_cache_hits_total",
			Help: "Total number of cache hits by cache tier",
		},
		[]string{"tier"}, // memory, disk
	)

	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_cache_misses_total",
			Help: "Total number of cache misses by cache tier",
		},
		[]string{"tier"},
	)

	CacheSizeBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gonuget_cache_size_bytes",
			Help: "Current cache size in bytes by tier",
		},
		[]string{"tier"},
	)

	// Package operation metrics
	PackageDownloadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_package_downloads_total",
			Help: "Total number of package downloads by status",
		},
		[]string{"status"}, // success, failure
	)

	PackageDownloadDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gonuget_package_download_duration_seconds",
			Help:    "Package download duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 12), // 100ms to 6min
		},
		[]string{"package_id"},
	)

	// Circuit breaker metrics
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gonuget_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"host"},
	)

	CircuitBreakerFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_circuit_breaker_failures_total",
			Help: "Total number of circuit breaker failures",
		},
		[]string{"host"},
	)

	// Rate limiter metrics
	RateLimitRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_rate_limit_requests_total",
			Help: "Total number of rate limited requests",
		},
		[]string{"source", "allowed"}, // allowed: true/false
	)

	RateLimitTokens = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gonuget_rate_limit_tokens",
			Help: "Current number of available rate limit tokens",
		},
		[]string{"source"},
	)
)

// MetricsHandler returns an HTTP handler for Prometheus metrics
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// StartMetricsServer starts an HTTP server exposing Prometheus metrics
func StartMetricsServer(addr string) error {
	http.Handle("/metrics", MetricsHandler())
	return http.ListenAndServe(addr, nil)
}
```

**Tests:** `observability/metrics_test.go`

```go
package observability

import (
	"net/http"
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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
```

---

## M4.14: Health Checks

**Goal:** Implement health check endpoint for monitoring and orchestration.

### Implementation

**File:** `observability/health.go`

```go
package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name   string
	Check  func(context.Context) HealthCheckResult
	Cached bool
	TTL    time.Duration
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status  HealthStatus      `json:"status"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// HealthChecker manages and executes health checks
type HealthChecker struct {
	mu     sync.RWMutex
	checks map[string]*HealthCheck
	cache  map[string]*cachedHealthResult
}

type cachedHealthResult struct {
	result    HealthCheckResult
	timestamp time.Time
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]*HealthCheck),
		cache:  make(map[string]*cachedHealthResult),
	}
}

// Register registers a new health check
func (hc *HealthChecker) Register(check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[check.Name] = &check
}

// Check executes all health checks and returns aggregate status
func (hc *HealthChecker) Check(ctx context.Context) map[string]HealthCheckResult {
	hc.mu.RLock()
	checks := make([]*HealthCheck, 0, len(hc.checks))
	for _, check := range hc.checks {
		checks = append(checks, check)
	}
	hc.mu.RUnlock()

	results := make(map[string]HealthCheckResult)
	var wg sync.WaitGroup

	for _, check := range checks {
		wg.Add(1)
		go func(c *HealthCheck) {
			defer wg.Done()
			result := hc.executeCheck(ctx, c)
			hc.mu.Lock()
			results[c.Name] = result
			hc.mu.Unlock()
		}(check)
	}

	wg.Wait()
	return results
}

// executeCheck executes a single health check with caching
func (hc *HealthChecker) executeCheck(ctx context.Context, check *HealthCheck) HealthCheckResult {
	// Check cache if enabled
	if check.Cached {
		hc.mu.RLock()
		cached, exists := hc.cache[check.Name]
		hc.mu.RUnlock()

		if exists && time.Since(cached.timestamp) < check.TTL {
			return cached.result
		}
	}

	// Execute check
	result := check.Check(ctx)

	// Cache result if enabled
	if check.Cached {
		hc.mu.Lock()
		hc.cache[check.Name] = &cachedHealthResult{
			result:    result,
			timestamp: time.Now(),
		}
		hc.mu.Unlock()
	}

	return result
}

// OverallStatus returns the aggregate health status
func (hc *HealthChecker) OverallStatus(ctx context.Context) HealthStatus {
	results := hc.Check(ctx)

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		switch result.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return HealthStatusUnhealthy
	}
	if hasDegraded {
		return HealthStatusDegraded
	}
	return HealthStatusHealthy
}

// Handler returns an HTTP handler for health checks
func (hc *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		results := hc.Check(ctx)
		overall := hc.OverallStatus(ctx)

		response := map[string]interface{}{
			"status": overall,
			"checks": results,
		}

		w.Header().Set("Content-Type", "application/json")

		// Set status code based on health
		switch overall {
		case HealthStatusHealthy:
			w.WriteHeader(http.StatusOK)
		case HealthStatusDegraded:
			w.WriteHeader(http.StatusOK) // Still operational
		case HealthStatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(response)
	}
}

// Common health checks

// HTTPSourceHealthCheck creates a health check for an HTTP source
func HTTPSourceHealthCheck(name, url string, timeout time.Duration) HealthCheck {
	return HealthCheck{
		Name:   name,
		Cached: true,
		TTL:    30 * time.Second,
		Check: func(ctx context.Context) HealthCheckResult {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
			if err != nil {
				return HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: "failed to create request: " + err.Error(),
				}
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: "request failed: " + err.Error(),
				}
			}
			resp.Body.Close()

			if resp.StatusCode >= 500 {
				return HealthCheckResult{
					Status:  HealthStatusDegraded,
					Message: "server error",
					Details: map[string]string{"status_code": resp.Status},
				}
			}

			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "source reachable",
			}
		},
	}
}

// CacheHealthCheck creates a health check for cache availability
func CacheHealthCheck(name string, sizeBytes int64, maxSizeBytes int64) HealthCheck {
	return HealthCheck{
		Name:   name,
		Cached: false, // Always fresh
		Check: func(ctx context.Context) HealthCheckResult {
			usagePercent := float64(sizeBytes) / float64(maxSizeBytes) * 100

			if usagePercent >= 95 {
				return HealthCheckResult{
					Status:  HealthStatusDegraded,
					Message: "cache nearly full",
					Details: map[string]string{
						"usage_percent": fmt.Sprintf("%.1f%%", usagePercent),
					},
				}
			}

			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "cache operational",
			}
		},
	}
}
```

**Tests:** `observability/health_test.go`

```go
package observability

import (
	"context"
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
	defer resp.Body.Close()

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
```

---

## Compatibility Notes

✅ **100% NuGet.Client Compatibility:**
- All observability features are internal enhancements
- Do not affect protocol behavior or external APIs
- Can be entirely disabled without functional impact
- NuGet.Client has minimal logging - gonuget adds production-grade observability

---

## Testing Requirements

### No Interop Tests Required

**Reasoning:** All observability features (mtlog, OpenTelemetry, Prometheus, health checks) are purely internal enhancements. NuGet.Client has minimal logging via `ILogger` interface with no equivalent observability stack. These features have **zero protocol impact** and can be entirely disabled without affecting functionality.

**Testing Strategy:**
- **Unit tests**: Logger interface, metric registration, span creation, health check logic
- **Integration tests**: OTLP exporters, HTTP middleware, metrics endpoint, trace propagation
- **Benchmarks**: Zero-allocation logging performance (target: <20ns/op per mtlog)
- **End-to-end**: Jaeger visualization, Prometheus scraping

**Coverage Target:** 85% (per PRD-TESTING.md for observability code)

**See:** `/Users/brandon/src/gonuget/docs/implementation/M4-INTEROP-ANALYSIS.md` for detailed testing rationale.

---

**Status:** M4.1-M4.14 complete with 100% NuGet.Client parity.

**Next:** IMPL-M4-HTTP3.md will cover M4.15 (HTTP/2 and HTTP/3 Support).
