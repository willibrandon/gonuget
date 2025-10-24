package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPTracingTransport(t *testing.T) {
	// Setup tracing
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

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close() failed: %v", err)
		}
	}()

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
	defer func() {
		if err := ShutdownTracing(ctx, tp); err != nil {
			t.Errorf("ShutdownTracing() failed: %v", err)
		}
	}()

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

func TestHTTPTracingTransport_4xxError(t *testing.T) {
	// Setup tracing
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

	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	// Create instrumented client
	client := InstrumentedHTTPClient("gonuget-test")

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/missing", nil)
	if err != nil {
		t.Fatalf("NewRequest() failed: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close() failed: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
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

func TestHTTPSpanAttributes_NoResponse(t *testing.T) {
	req, err := http.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)
	if err != nil {
		t.Fatalf("NewRequest() failed: %v", err)
	}

	// Test with nil response
	attrs := HTTPSpanAttributes(req, nil)

	// Should have request attributes but not response attributes
	expectedRequestAttrs := map[string]bool{
		"http.method":   false,
		"http.url":      false,
		"http.scheme":   false,
		"net.peer.name": false,
	}

	for _, kv := range attrs {
		key := string(kv.Key)
		if _, ok := expectedRequestAttrs[key]; ok {
			expectedRequestAttrs[key] = true
		}
	}

	for key, found := range expectedRequestAttrs {
		if !found {
			t.Errorf("Missing expected request attribute: %s", key)
		}
	}

	// Should not have response attributes
	for _, kv := range attrs {
		key := string(kv.Key)
		if key == "http.status_code" || key == "http.response_content_length" {
			t.Errorf("Should not have response attribute %s when response is nil", key)
		}
	}
}

func TestNewHTTPTracingTransport_NilBase(t *testing.T) {
	transport := NewHTTPTracingTransport(nil, "test")

	if transport == nil {
		t.Fatal("NewHTTPTracingTransport() returned nil")
	}

	if transport.base == nil {
		t.Error("transport.base should not be nil when nil is passed")
	}

	if transport.tracerName != "test" {
		t.Errorf("tracerName = %s, want test", transport.tracerName)
	}
}

func TestInstrumentedHTTPClient(t *testing.T) {
	client := InstrumentedHTTPClient("test-client")

	if client == nil {
		t.Fatal("InstrumentedHTTPClient() returned nil")
	}

	if client.Transport == nil {
		t.Error("client.Transport should not be nil")
	}

	// Verify it's our tracing transport
	if _, ok := client.Transport.(*HTTPTracingTransport); !ok {
		t.Error("client.Transport should be *HTTPTracingTransport")
	}
}
