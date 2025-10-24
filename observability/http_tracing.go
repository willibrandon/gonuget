package observability

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
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

	// Update request with new context
	req = req.WithContext(ctx)

	// Inject trace context into HTTP headers using W3C Trace Context propagator
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

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
