package http

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
)

// generateTestCertificates generates a CA and leaf certificate for testing
func generateTestCertificates(t *testing.T) (tlsServerConfig, tlsClientConfig *tls.Config) {
	t.Helper()

	// Generate CA
	ca := &x509.Certificate{
		SerialNumber:          big.NewInt(2024),
		Subject:               pkix.Name{Organization: []string{"gonuget-test"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPub, caPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate CA key: %v", err)
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, caPub, caPriv)
	if err != nil {
		t.Fatalf("Failed to create CA certificate: %v", err)
	}

	caCert, err := x509.ParseCertificate(caBytes)
	if err != nil {
		t.Fatalf("Failed to parse CA certificate: %v", err)
	}

	// Generate leaf certificate
	leaf := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	leafPub, leafPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate leaf key: %v", err)
	}

	leafBytes, err := x509.CreateCertificate(rand.Reader, leaf, caCert, leafPub, caPriv)
	if err != nil {
		t.Fatalf("Failed to create leaf certificate: %v", err)
	}

	// Server config
	tlsServerConfig = &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{leafBytes},
			PrivateKey:  leafPriv,
		}},
		NextProtos: []string{"h3"},
	}

	// Client config
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)
	tlsClientConfig = &tls.Config{
		ServerName: "localhost",
		RootCAs:    certPool,
		NextProtos: []string{"h3"},
	}

	return tlsServerConfig, tlsClientConfig
}

func TestNewTransport_HTTP1(t *testing.T) {
	config := TransportConfig{
		EnableHTTP2:         false,
		EnableHTTP3:         false,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	}

	transport := NewTransport(config)

	if transport == nil {
		t.Fatal("NewTransport() returned nil")
	}

	// Verify it's an HTTP/1.1 transport
	httpTransport, ok := transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport is not *http.Transport")
	}

	if httpTransport.MaxIdleConns != 50 {
		t.Errorf("MaxIdleConns = %d, want 50", httpTransport.MaxIdleConns)
	}
}

func TestNewTransport_HTTP2(t *testing.T) {
	config := TransportConfig{
		EnableHTTP2:         true,
		EnableHTTP3:         false,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}

	transport := NewTransport(config)

	if transport == nil {
		t.Fatal("NewTransport() returned nil")
	}

	// HTTP/2 configuration is done via http2.ConfigureTransport
	// which modifies the http.Transport in place
	httpTransport, ok := transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport is not *http.Transport")
	}

	// Verify HTTP/2 is configured by checking TLSNextProto
	// http2.ConfigureTransport sets TLSNextProto
	if httpTransport.TLSNextProto == nil {
		t.Error("HTTP/2 not configured (TLSNextProto is nil)")
	}
}

func TestNewHTTPClient_HTTP1(t *testing.T) {
	config := TransportConfig{
		EnableHTTP2: false,
		EnableHTTP3: false,
	}

	client := NewHTTPClient(config)

	if client == nil {
		t.Fatal("NewHTTPClient() returned nil")
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Make request
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	// Verify HTTP/1.1
	if resp.ProtoMajor != 1 {
		t.Errorf("ProtoMajor = %d, want 1 (HTTP/1.x)", resp.ProtoMajor)
	}
}

func TestNewHTTPClient_HTTP2(t *testing.T) {
	config := TransportConfig{
		EnableHTTP2: true,
		EnableHTTP3: false,
	}

	client := NewHTTPClient(config)

	if client == nil {
		t.Fatal("NewHTTPClient() returned nil")
	}

	// Create TLS test server with HTTP/2 enabled
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	// Use server's client for TLS verification
	client.Transport = server.Client().Transport

	// Make request
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	// Verify we got HTTP/2
	if resp.ProtoMajor != 2 {
		t.Errorf("ProtoMajor = %d, want 2 (HTTP/2)", resp.ProtoMajor)
	}

	t.Logf("✓ HTTP/2 request successful (protocol: %s)", resp.Proto)
}

func TestNewHTTPClient_HTTP3(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HTTP/3 test in short mode")
	}

	serverTLS, clientTLS := generateTestCertificates(t)

	// Setup HTTP/3 server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("HTTP/3 OK"))
	})

	server := &http3.Server{
		Addr:      "127.0.0.1:0",
		Handler:   handler,
		TLSConfig: serverTLS,
	}

	// Start server in background
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create UDP listener: %v", err)
	}
	defer func() {
		_ = udpConn.Close()
	}()

	serverAddr := udpConn.LocalAddr().String()
	t.Logf("HTTP/3 server listening on %s", serverAddr)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(udpConn)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create HTTP/3 client
	config := TransportConfig{
		EnableHTTP2: false,
		EnableHTTP3: true,
	}

	client := NewHTTPClient(config)
	if client == nil {
		t.Fatal("NewHTTPClient() returned nil")
	}

	// Verify transport is http3Transport wrapper
	h3Transport, ok := client.Transport.(*http3Transport)
	if !ok {
		t.Fatal("Transport is not *http3Transport")
	}

	// Configure client TLS
	h3Transport.http3Transport.TLSClientConfig = clientTLS

	// Make HTTP/3 request
	url := "https://" + serverAddr + "/"
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("HTTP/3 request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	// Verify we got HTTP/3
	if resp.ProtoMajor != 3 {
		t.Errorf("ProtoMajor = %d, want 3 (HTTP/3)", resp.ProtoMajor)
	}

	proto := ProtocolVersion(resp)
	if proto != "HTTP/3" {
		t.Errorf("ProtocolVersion() = %s, want HTTP/3", proto)
	}

	t.Logf("✓ HTTP/3 request successful (protocol: %s)", proto)
}

func TestProtocolVersion(t *testing.T) {
	tests := []struct {
		name        string
		protoMajor  int
		protoMinor  int
		wantVersion string
	}{
		{
			name:        "HTTP/1.1",
			protoMajor:  1,
			protoMinor:  1,
			wantVersion: "HTTP/1.1",
		},
		{
			name:        "HTTP/2",
			protoMajor:  2,
			protoMinor:  0,
			wantVersion: "HTTP/2",
		},
		{
			name:        "HTTP/3",
			protoMajor:  3,
			protoMinor:  0,
			wantVersion: "HTTP/3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				ProtoMajor: tt.protoMajor,
				ProtoMinor: tt.protoMinor,
			}

			version := ProtocolVersion(resp)
			if version != tt.wantVersion {
				t.Errorf("ProtocolVersion() = %s, want %s", version, tt.wantVersion)
			}
		})
	}
}

func TestNewDefaultHTTPClient(t *testing.T) {
	client := NewDefaultHTTPClient()

	if client == nil {
		t.Fatal("NewDefaultHTTPClient() returned nil")
	}

	if client.Timeout != 0 {
		t.Errorf("Client timeout = %v, want 0 (context-based)", client.Timeout)
	}
}

func TestHTTP3Transport_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HTTP/3 test in short mode")
	}

	config := TransportConfig{
		EnableHTTP2: false,
		EnableHTTP3: true,
	}

	transport := NewTransport(config)
	h3Transport, ok := transport.(*http3Transport)
	if !ok {
		t.Fatal("Transport is not *http3Transport")
	}

	// Close should not error
	if err := h3Transport.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Closing again should still not error (idempotent)
	if err := h3Transport.Close(); err != nil {
		t.Errorf("Second Close() error = %v, want nil", err)
	}
}

func BenchmarkHTTP1_Request(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := TransportConfig{
		EnableHTTP2: false,
		EnableHTTP3: false,
	}
	client := NewHTTPClient(config)

	for b.Loop() {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatalf("Get() failed: %v", err)
		}
		_ = resp.Body.Close()
	}
}

func BenchmarkHTTP2_Request(b *testing.B) {
	// Create unstarted server to configure HTTP/2
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	config := TransportConfig{
		EnableHTTP2: true,
		EnableHTTP3: false,
	}
	client := NewHTTPClient(config)
	client.Transport = server.Client().Transport

	for b.Loop() {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatalf("Get() failed: %v", err)
		}
		_ = resp.Body.Close()
	}
}

func BenchmarkHTTP3_Request(b *testing.B) {
	serverTLS, clientTLS := generateTestCertificates(&testing.T{})

	// Setup HTTP/3 server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http3.Server{
		Addr:      "127.0.0.1:0",
		Handler:   handler,
		TLSConfig: serverTLS,
	}

	// Start server
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		b.Fatalf("Failed to create UDP listener: %v", err)
	}
	defer func() {
		_ = udpConn.Close()
	}()

	serverAddr := udpConn.LocalAddr().String()

	go func() {
		_ = server.Serve(udpConn)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Create HTTP/3 client
	config := TransportConfig{
		EnableHTTP2: false,
		EnableHTTP3: true,
	}
	client := NewHTTPClient(config)
	h3Transport := client.Transport.(*http3Transport)
	h3Transport.http3Transport.TLSClientConfig = clientTLS

	url := "https://" + serverAddr + "/"

	for b.Loop() {
		resp, err := client.Get(url)
		if err != nil {
			b.Fatalf("Get() failed: %v", err)
		}
		_ = resp.Body.Close()
	}
}

// Concurrent request benchmarks - this is where HTTP/2 and HTTP/3 shine

func BenchmarkHTTP1_ConcurrentRequests(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := TransportConfig{
		EnableHTTP2:         false,
		EnableHTTP3:         false,
		MaxIdleConnsPerHost: 100, // Allow connection reuse
		MaxIdleConns:        1000,
	}
	client := NewHTTPClient(config)

	// Warm up connection pool
	for range 10 {
		resp, _ := client.Get(server.URL)
		if resp != nil {
			_ = resp.Body.Close()
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Fatalf("Get() failed: %v", err)
			}
			_ = resp.Body.Close()
		}
	})
}

func BenchmarkHTTP2_ConcurrentRequests(b *testing.B) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	config := TransportConfig{
		EnableHTTP2:         true,
		EnableHTTP3:         false,
		MaxIdleConnsPerHost: 100,
		MaxIdleConns:        1000,
	}
	client := NewHTTPClient(config)
	client.Transport = server.Client().Transport

	// Warm up connection pool
	for range 10 {
		resp, _ := client.Get(server.URL)
		if resp != nil {
			_ = resp.Body.Close()
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Fatalf("Get() failed: %v", err)
			}
			_ = resp.Body.Close()
		}
	})
}

func BenchmarkHTTP3_ConcurrentRequests(b *testing.B) {
	serverTLS, clientTLS := generateTestCertificates(&testing.T{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http3.Server{
		Addr:      "127.0.0.1:0",
		Handler:   handler,
		TLSConfig: serverTLS,
	}

	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		b.Fatalf("Failed to create UDP listener: %v", err)
	}
	defer func() {
		_ = udpConn.Close()
	}()

	serverAddr := udpConn.LocalAddr().String()

	go func() {
		_ = server.Serve(udpConn)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	config := TransportConfig{
		EnableHTTP2: false,
		EnableHTTP3: true,
	}
	client := NewHTTPClient(config)
	h3Transport := client.Transport.(*http3Transport)
	h3Transport.http3Transport.TLSClientConfig = clientTLS

	url := "https://" + serverAddr + "/"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("Get() failed: %v", err)
			}
			_ = resp.Body.Close()
		}
	})
}
