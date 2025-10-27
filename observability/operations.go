package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName is the tracer name for gonuget operations
	TracerName = "github.com/willibrandon/gonuget"
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

// StartServiceIndexFetchSpan starts a span for service index fetch
func StartServiceIndexFetchSpan(ctx context.Context, sourceURL string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "service_index.fetch",
		trace.WithAttributes(
			AttrSourceURL.String(sourceURL),
			AttrOperation.String("fetch_service_index"),
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
