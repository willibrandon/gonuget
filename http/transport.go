package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

// TransportConfig configures HTTP transport with protocol support
type TransportConfig struct {
	// EnableHTTP2 enables HTTP/2 support (default: true)
	EnableHTTP2 bool

	// EnableHTTP3 enables HTTP/3 support (default: false, experimental)
	EnableHTTP3 bool

	// MaxIdleConns controls the maximum number of idle connections
	MaxIdleConns int

	// MaxIdleConnsPerHost controls idle connections per host
	MaxIdleConnsPerHost int

	// IdleConnTimeout is the maximum time an idle connection will remain idle
	IdleConnTimeout time.Duration

	// TLSHandshakeTimeout is the maximum time for TLS handshake
	TLSHandshakeTimeout time.Duration

	// ResponseHeaderTimeout is the maximum time to wait for response headers
	ResponseHeaderTimeout time.Duration

	// ExpectContinueTimeout is the time to wait for 100-Continue response
	ExpectContinueTimeout time.Duration

	// MaxConnsPerHost limits total connections per host
	MaxConnsPerHost int
}

// DefaultTransportConfig returns default transport configuration
func DefaultTransportConfig() TransportConfig {
	return TransportConfig{
		EnableHTTP2:           true,
		EnableHTTP3:           false, // Experimental
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxConnsPerHost:       0, // Unlimited
	}
}

// NewTransport creates an HTTP transport with configured protocol support
func NewTransport(config TransportConfig) http.RoundTripper {
	// Base HTTP/1.1 transport
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		MaxConnsPerHost:       config.MaxConnsPerHost,
	}

	// Enable HTTP/2 if requested
	if config.EnableHTTP2 {
		// Configure HTTP/2 transport
		// This enables automatic HTTP/2 negotiation via ALPN when using TLS
		_ = http2.ConfigureTransport(transport)
		// Ignore errors - falls back to HTTP/1.1 if HTTP/2 configuration fails
	}

	// HTTP/3 support is handled separately via alternate transport
	if config.EnableHTTP3 {
		return newHTTP3Transport(transport)
	}

	return transport
}

// http3Transport wraps HTTP/1.1 and HTTP/3 transports with automatic fallback
type http3Transport struct {
	http1Transport http.RoundTripper
	http3Transport *http3.Transport
}

// newHTTP3Transport creates a transport with HTTP/3 and HTTP/1.1 fallback
func newHTTP3Transport(http1Transport http.RoundTripper) *http3Transport {
	return &http3Transport{
		http1Transport: http1Transport,
		http3Transport: &http3.Transport{
			TLSClientConfig: &tls.Config{
				// Use default TLS config
				MinVersion: tls.VersionTLS12,
			},
			QUICConfig: &quic.Config{
				// Enable 0-RTT for performance
				Allow0RTT: true,
			},
		},
	}
}

// RoundTrip implements http.RoundTripper with HTTP/3 fallback
func (t *http3Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Try HTTP/3 first for HTTPS requests
	if req.URL.Scheme == "https" {
		resp, err := t.http3Transport.RoundTrip(req)
		if err == nil {
			return resp, nil
		}
		// HTTP/3 failed, fall back to HTTP/1.1 or HTTP/2
	}

	// Fallback to HTTP/1.1 or HTTP/2
	return t.http1Transport.RoundTrip(req)
}

// Close closes the HTTP/3 transport
func (t *http3Transport) Close() error {
	return t.http3Transport.Close()
}

// NewHTTPClient creates an HTTP client with configured transport
func NewHTTPClient(config TransportConfig) *http.Client {
	transport := NewTransport(config)

	return &http.Client{
		Transport: transport,
		Timeout:   0, // No timeout at client level (use context)
	}
}

// NewDefaultHTTPClient creates an HTTP client with default configuration
func NewDefaultHTTPClient() *http.Client {
	return NewHTTPClient(DefaultTransportConfig())
}

// ProtocolVersion returns the HTTP protocol version from response
func ProtocolVersion(resp *http.Response) string {
	if resp.ProtoMajor == 3 {
		return "HTTP/3"
	}
	if resp.ProtoMajor == 2 {
		return "HTTP/2"
	}
	return "HTTP/1.1"
}
