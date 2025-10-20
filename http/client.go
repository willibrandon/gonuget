// Package http provides HTTP client functionality for NuGet protocol operations.
//
// It wraps the standard http.Client with NuGet-specific configuration including
// configurable timeouts, user agent management, and HTTP/2 support.
package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	DefaultTimeout     = 30 * time.Second
	DefaultDialTimeout = 10 * time.Second
	DefaultUserAgent   = "gonuget/0.1.0"
)

// Client wraps http.Client with NuGet-specific configuration
type Client struct {
	httpClient *http.Client
	userAgent  string
	timeout    time.Duration
}

// Config holds HTTP client configuration
type Config struct {
	Timeout      time.Duration
	DialTimeout  time.Duration
	UserAgent    string
	TLSConfig    *tls.Config
	MaxIdleConns int
	EnableHTTP2  bool
}

// DefaultConfig returns a client configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Timeout:      DefaultTimeout,
		DialTimeout:  DefaultDialTimeout,
		UserAgent:    DefaultUserAgent,
		MaxIdleConns: 100,
		EnableHTTP2:  true,
	}
}

// NewClient creates a new HTTP client with the given configuration
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     cfg.TLSConfig,
		ForceAttemptHTTP2:   cfg.EnableHTTP2,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		},
		userAgent: cfg.UserAgent,
		timeout:   cfg.Timeout,
	}
}

// Do executes an HTTP request with context and user agent
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	return c.httpClient.Do(req)
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	return c.Do(ctx, req)
}

// SetUserAgent updates the client's user agent string
func (c *Client) SetUserAgent(ua string) {
	c.userAgent = ua
}

// Option is a functional option for configuring the client
type Option func(*Config)

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}

// WithUserAgent sets the user agent string
func WithUserAgent(ua string) Option {
	return func(cfg *Config) {
		cfg.UserAgent = ua
	}
}

// WithTLSConfig sets custom TLS configuration
func WithTLSConfig(tlsCfg *tls.Config) Option {
	return func(cfg *Config) {
		cfg.TLSConfig = tlsCfg
	}
}

// WithMaxIdleConns sets the maximum idle connections
func WithMaxIdleConns(n int) Option {
	return func(cfg *Config) {
		cfg.MaxIdleConns = n
	}
}

// NewClientWithOptions creates a client with functional options
func NewClientWithOptions(opts ...Option) *Client {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return NewClient(cfg)
}
