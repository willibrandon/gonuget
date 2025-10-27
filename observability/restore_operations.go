package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StartProtocolDetectionSpan starts a span for protocol detection
func StartProtocolDetectionSpan(ctx context.Context, sourceURL string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "protocol.detect",
		trace.WithAttributes(
			attribute.String("source.url", sourceURL),
		),
	)
}

// StartMetadataFetchSpan starts a span for metadata fetching
func StartMetadataFetchSpan(ctx context.Context, packageID, protocol string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "metadata.fetch",
		trace.WithAttributes(
			AttrPackageID.String(packageID),
			attribute.String("protocol", protocol),
		),
	)
}

// StartFileWriteSpan starts a span for file write operations
func StartFileWriteSpan(ctx context.Context, filePath string, size int64) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "file.write",
		trace.WithAttributes(
			attribute.String("file.path", filePath),
			attribute.Int64("file.size", size),
		),
	)
}

// StartRepositoryCreationSpan starts a span for repository creation
func StartRepositoryCreationSpan(ctx context.Context, sourceURL string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "repository.create",
		trace.WithAttributes(
			AttrSourceURL.String(sourceURL),
		),
	)
}

// StartResolverWalkSpan starts a span for dependency walker
func StartResolverWalkSpan(ctx context.Context, packageID string, targetFramework string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "resolver.walk",
		trace.WithAttributes(
			AttrPackageID.String(packageID),
			attribute.String("target.framework", targetFramework),
		),
	)
}

// StartMetadataFetchV2Span starts a span for V2 metadata fetch
func StartMetadataFetchV2Span(ctx context.Context, packageID string, source string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "metadata.fetch.v2",
		trace.WithAttributes(
			AttrPackageID.String(packageID),
			AttrSourceURL.String(source),
		),
	)
}

// StartMetadataFetchV3Span starts a span for V3 metadata fetch
func StartMetadataFetchV3Span(ctx context.Context, packageID string, source string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerName, "metadata.fetch.v3",
		trace.WithAttributes(
			AttrPackageID.String(packageID),
			AttrSourceURL.String(source),
		),
	)
}
