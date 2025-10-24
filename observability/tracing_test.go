package observability

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
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
	defer func() {
		if err := ShutdownTracing(ctx, tp); err != nil {
			t.Errorf("ShutdownTracing() failed: %v", err)
		}
	}()

	// Create a test span
	tracer := Tracer("test")
	_, span := tracer.Start(ctx, "test-operation")
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
	defer func() {
		if err := ShutdownTracing(ctx, tp); err != nil {
			t.Errorf("ShutdownTracing() failed: %v", err)
		}
	}()
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()
	config := DefaultTracerConfig()

	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Fatalf("SetupTracing() failed: %v", err)
	}
	defer func() {
		if err := ShutdownTracing(ctx, tp); err != nil {
			t.Errorf("ShutdownTracing() failed: %v", err)
		}
	}()

	_, span := StartSpan(ctx, "gonuget", "test-span")
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
	defer func() {
		if err := ShutdownTracing(ctx, tp); err != nil {
			t.Errorf("ShutdownTracing() failed: %v", err)
		}
	}()

	ctx, span := StartSpan(ctx, "gonuget", "test-span")
	defer span.End()

	// Test AddEvent
	AddEvent(ctx, "test-event", attribute.String("event.key", "event.value"))

	// Test SetAttributes
	SetAttributes(ctx, attribute.Int("request.count", 42))

	// Test RecordError
	RecordError(ctx, context.DeadlineExceeded)

	// Test SpanFromContext
	retrievedSpan := SpanFromContext(ctx)
	if !retrievedSpan.SpanContext().IsValid() {
		t.Error("SpanFromContext should return a valid span")
	}
	if retrievedSpan.SpanContext().TraceID() != span.SpanContext().TraceID() {
		t.Error("SpanFromContext should return span with same TraceID")
	}

	// Should not panic
}

func TestSetupTracing_InvalidExporter(t *testing.T) {
	ctx := context.Background()
	config := TracerConfig{
		ServiceName:  "gonuget-test",
		ExporterType: "invalid",
	}

	_, err := SetupTracing(ctx, config)
	if err == nil {
		t.Error("SetupTracing with invalid exporter should return error")
	}
}

func TestDefaultTracerConfig(t *testing.T) {
	config := DefaultTracerConfig()

	if config.ServiceName != "gonuget" {
		t.Errorf("Expected ServiceName=gonuget, got %s", config.ServiceName)
	}
	if config.ServiceVersion != "0.1.0" {
		t.Errorf("Expected ServiceVersion=0.1.0, got %s", config.ServiceVersion)
	}
	if config.Environment != "development" {
		t.Errorf("Expected Environment=development, got %s", config.Environment)
	}
	if config.ExporterType != "stdout" {
		t.Errorf("Expected ExporterType=stdout, got %s", config.ExporterType)
	}
	if config.SamplingRate != 1.0 {
		t.Errorf("Expected SamplingRate=1.0, got %f", config.SamplingRate)
	}
}

func TestShutdownTracing(t *testing.T) {
	ctx := context.Background()
	config := DefaultTracerConfig()

	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Fatalf("SetupTracing() failed: %v", err)
	}

	// Test shutdown
	err = ShutdownTracing(ctx, tp)
	if err != nil {
		t.Errorf("ShutdownTracing() failed: %v", err)
	}
}

func TestTracerFunction(t *testing.T) {
	ctx := context.Background()
	config := DefaultTracerConfig()

	tp, err := SetupTracing(ctx, config)
	if err != nil {
		t.Fatalf("SetupTracing() failed: %v", err)
	}
	defer func() {
		if err := ShutdownTracing(ctx, tp); err != nil {
			t.Errorf("ShutdownTracing() failed: %v", err)
		}
	}()

	tracer := Tracer("test-tracer")
	if tracer == nil {
		t.Error("Tracer() should not return nil")
	}
}

func TestSetupTracing_OTLPIntegration(t *testing.T) {
	// This is an integration test that requires an OTLP collector
	// To run: docker run -d -p 4317:4317 -p 16686:16686 jaegertracing/all-in-one:latest
	endpoint := "localhost:4317"

	// Check if collector is available by attempting to connect
	checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Skipf("OTLP collector not available at %s (run: docker run -d -p 4317:4317 jaegertracing/all-in-one:latest): %v", endpoint, err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close gRPC connection: %v", err)
		}
	}()

	// Trigger connection and wait for Ready state or timeout
	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			break
		}
		if state == connectivity.Shutdown || state == connectivity.TransientFailure {
			t.Skipf("OTLP collector not available at %s (state: %v, run: docker compose -f docker-compose.test.yml up -d)", endpoint, state)
			return
		}
		if !conn.WaitForStateChange(checkCtx, state) {
			// Timeout waiting for state change
			t.Skipf("OTLP collector not available at %s (timeout waiting for Ready state, run: docker compose -f docker-compose.test.yml up -d)", endpoint)
			return
		}
	}

	// Collector is available, run the test
	config := TracerConfig{
		ServiceName:    "gonuget-integration-test",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   "otlp",
		OTLPEndpoint:   endpoint,
		OTLPInsecure:   true, // Local Jaeger uses insecure gRPC
		SamplingRate:   1.0,
	}

	tp, err := SetupTracing(context.Background(), config)
	if err != nil {
		t.Fatalf("SetupTracing() with OTLP failed: %v", err)
	}
	defer func() {
		if err := ShutdownTracing(context.Background(), tp); err != nil {
			t.Errorf("ShutdownTracing() failed: %v", err)
		}
	}()

	// Create test spans
	spanCtx := context.Background()
	spanCtx, span := StartSpan(spanCtx, "gonuget", "integration-test-span")
	span.SetAttributes(
		attribute.String("test.type", "integration"),
		attribute.String("collector.endpoint", endpoint),
	)
	AddEvent(spanCtx, "test-started")

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	AddEvent(spanCtx, "test-completed")
	span.End()

	// Give the exporter time to send
	time.Sleep(100 * time.Millisecond)

	t.Logf("Integration test completed. View traces at http://localhost:16686")
}
