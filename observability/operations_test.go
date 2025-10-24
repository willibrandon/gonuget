package observability

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestStartPackageDownloadSpan(t *testing.T) {
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

	ctx, span := StartPackageDownloadSpan(ctx, "Newtonsoft.Json", "13.0.3", "https://api.nuget.org")
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Error("Span context should be valid")
	}
}

func TestStartPackageRestoreSpan(t *testing.T) {
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

	ctx, span := StartPackageRestoreSpan(ctx, "/path/to/project.csproj", 5)
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Error("Span context should be valid")
	}
}

func TestStartCacheLookupSpan(t *testing.T) {
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

	ctx, span := StartCacheLookupSpan(ctx, "test-key")
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Error("Span context should be valid")
	}
}

func TestRecordCacheHit(t *testing.T) {
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

	ctx, span := StartCacheLookupSpan(ctx, "test-key")
	defer span.End()

	RecordCacheHit(ctx, true)
	// Should not panic

	RecordCacheHit(ctx, false)
	// Should not panic
}

func TestStartDependencyResolutionSpan(t *testing.T) {
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

	ctx, span := StartDependencyResolutionSpan(ctx, "Newtonsoft.Json", "net6.0")
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Error("Span context should be valid")
	}
}

func TestStartFrameworkSelectionSpan(t *testing.T) {
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

	ctx, span := StartFrameworkSelectionSpan(ctx, "net6.0", 3)
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Error("Span context should be valid")
	}
}

func TestRecordRetry(t *testing.T) {
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

	ctx, span := StartPackageDownloadSpan(ctx, "Test.Package", "1.0.0", "https://example.com")
	defer span.End()

	RecordRetry(ctx, 1, errors.New("connection timeout"))
	// Should not panic

	RecordRetry(ctx, 2, errors.New("connection timeout"))
	// Should not panic
}

func TestEndSpanWithError(t *testing.T) {
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

	// Test with error
	ctx, span := StartPackageDownloadSpan(ctx, "Test.Package", "1.0.0", "https://example.com")
	testErr := errors.New("download failed")
	EndSpanWithError(span, testErr)
	// Should not panic

	// Test without error
	ctx, span = StartPackageDownloadSpan(ctx, "Test.Package", "1.0.0", "https://example.com")
	EndSpanWithError(span, nil)
	// Should not panic
}

func TestTracerName(t *testing.T) {
	expected := "github.com/willibrandon/gonuget"
	if TracerName != expected {
		t.Errorf("TracerName = %q, want %q", TracerName, expected)
	}
}

func TestAttributeKeys(t *testing.T) {
	tests := []struct {
		name     string
		key      attribute.Key
		expected string
	}{
		{"PackageID", AttrPackageID, "nuget.package.id"},
		{"PackageVersion", AttrPackageVersion, "nuget.package.version"},
		{"SourceURL", AttrSourceURL, "nuget.source.url"},
		{"Framework", AttrFramework, "nuget.framework"},
		{"Operation", AttrOperation, "nuget.operation"},
		{"CacheHit", AttrCacheHit, "nuget.cache.hit"},
		{"RetryCount", AttrRetryCount, "nuget.retry.count"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.key) != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, string(tt.key), tt.expected)
			}
		})
	}
}
