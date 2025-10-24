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

	"github.com/willibrandon/gonuget/observability"
	"github.com/willibrandon/gonuget/resilience"
)

const (
	DefaultTimeout     = 30 * time.Second
	DefaultDialTimeout = 10 * time.Second
	DefaultUserAgent   = "gonuget/0.1.0"
)

// Client wraps http.Client with NuGet-specific configuration
type Client struct {
	httpClient     *http.Client
	userAgent      string
	timeout        time.Duration
	retryConfig    *RetryConfig
	logger         observability.Logger
	circuitBreaker *resilience.HTTPCircuitBreaker // Optional circuit breaker (nil disables)
	rateLimiter    *resilience.PerSourceLimiter   // Optional rate limiter (nil disables)
}

// Config holds HTTP client configuration
type Config struct {
	Timeout              time.Duration
	DialTimeout          time.Duration
	UserAgent            string
	TLSConfig            *tls.Config
	MaxIdleConns         int
	EnableHTTP2          bool
	RetryConfig          *RetryConfig
	Logger               observability.Logger             // Optional logger (nil uses NullLogger)
	EnableTracing        bool                             // Enable OpenTelemetry HTTP tracing
	CircuitBreakerConfig *resilience.CircuitBreakerConfig // Optional circuit breaker config (nil disables)
	RateLimiterConfig    *resilience.TokenBucketConfig    // Optional rate limiter config (nil disables)
}

// DefaultConfig returns a client configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Timeout:      DefaultTimeout,
		DialTimeout:  DefaultDialTimeout,
		UserAgent:    DefaultUserAgent,
		MaxIdleConns: 100,
		EnableHTTP2:  true,
		RetryConfig:  DefaultRetryConfig(),
	}
}

// NewClient creates a new HTTP client with the given configuration
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.RetryConfig == nil {
		cfg.RetryConfig = DefaultRetryConfig()
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

	// Wrap transport with tracing if enabled
	var finalTransport http.RoundTripper = transport
	if cfg.EnableTracing {
		finalTransport = observability.NewHTTPTracingTransport(transport, "github.com/willibrandon/gonuget/http")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = observability.NewNullLogger()
	}

	client := &Client{
		httpClient: &http.Client{
			Transport: finalTransport,
			Timeout:   cfg.Timeout,
		},
		userAgent:   cfg.UserAgent,
		timeout:     cfg.Timeout,
		retryConfig: cfg.RetryConfig,
		logger:      logger,
	}

	// Add circuit breaker if configured
	if cfg.CircuitBreakerConfig != nil {
		client.circuitBreaker = resilience.NewHTTPCircuitBreaker(*cfg.CircuitBreakerConfig)
	}

	// Add rate limiter if configured
	if cfg.RateLimiterConfig != nil {
		client.rateLimiter = resilience.NewPerSourceLimiter(*cfg.RateLimiterConfig)
	}

	return client
}

// Do executes an HTTP request with context and user agent
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	// Extract host for circuit breaker and rate limiter
	host := req.URL.Host

	// Apply rate limiting before request
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, host); err != nil {
			c.logger.WarnContext(ctx, "HTTP {Method} {URL} rate limit wait failed: {Error}",
				req.Method, req.URL.String(), err)
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	c.logger.DebugContext(ctx, "HTTP {Method} {URL}", req.Method, req.URL.String())

	// Execute request with circuit breaker protection
	executeRequest := func(context.Context) (*http.Response, error) {
		start := time.Now()
		resp, err := c.httpClient.Do(req)
		duration := time.Since(start)

		if err != nil {
			c.logger.WarnContext(ctx, "HTTP {Method} {URL} failed after {Duration}ms: {Error}",
				req.Method, req.URL.String(), duration.Milliseconds(), err)
			observability.HTTPRequestsTotal.WithLabelValues(req.Method, "error", req.URL.Host).Inc()
			return nil, err
		}

		c.logger.DebugContext(ctx, "HTTP {Method} {URL} → {StatusCode} ({Duration}ms)",
			req.Method, req.URL.String(), resp.StatusCode, duration.Milliseconds())
		observability.HTTPRequestsTotal.WithLabelValues(req.Method, fmt.Sprintf("%d", resp.StatusCode), req.URL.Host).Inc()
		observability.HTTPRequestDuration.WithLabelValues(req.Method, req.URL.Host).Observe(duration.Seconds())

		return resp, nil
	}

	// Apply circuit breaker if configured
	if c.circuitBreaker != nil {
		return c.circuitBreaker.Execute(ctx, host, executeRequest)
	}

	return executeRequest(ctx)
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

// DoWithRetry executes an HTTP request with retry logic
func (c *Client) DoWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Extract host for circuit breaker and rate limiter
	host := req.URL.Host

	// Apply rate limiting before retry attempts
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, host); err != nil {
			c.logger.WarnContext(ctx, "HTTP {Method} {URL} rate limit wait failed: {Error}",
				req.Method, req.URL.String(), err)
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	c.logger.DebugContext(ctx, "HTTP {Method} {URL} with retry (max={MaxRetries})",
		req.Method, req.URL.String(), c.retryConfig.MaxRetries)

	// Execute retry logic with circuit breaker wrapping entire sequence
	executeWithRetry := func(context.Context) (*http.Response, error) {
		var lastErr error
		var resp *http.Response

		for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
			// Clone request for retry (body may have been consumed)
			reqClone := req.Clone(ctx)
			if req.Header.Get("User-Agent") == "" {
				reqClone.Header.Set("User-Agent", c.userAgent)
			}

			resp, lastErr = c.httpClient.Do(reqClone)

			// Success
			if lastErr == nil && !IsRetriableStatus(resp.StatusCode) {
				if attempt > 0 {
					c.logger.InfoContext(ctx, "HTTP {Method} {URL} succeeded after {Attempt} retries",
						req.Method, req.URL.String(), attempt)
				}
				return resp, nil
			}

			// Check if error is retriable
			if lastErr != nil && !IsRetriable(lastErr) {
				c.logger.WarnContext(ctx, "HTTP {Method} {URL} failed with non-retriable error: {Error}",
					req.Method, req.URL.String(), lastErr)
				return nil, lastErr
			}

			// Check if status is retriable
			if resp != nil && !IsRetriableStatus(resp.StatusCode) {
				return resp, nil
			}

			// Don't sleep after last attempt
			if attempt < c.retryConfig.MaxRetries {
				var backoff time.Duration

				// Check for Retry-After header
				if resp != nil {
					retryAfter := ParseRetryAfter(resp.Header.Get("Retry-After"))
					if retryAfter > 0 {
						backoff = retryAfter
					}
				}

				// Fall back to exponential backoff if no Retry-After
				if backoff == 0 {
					backoff = c.retryConfig.CalculateBackoff(attempt)
				}

				c.logger.DebugContext(ctx, "HTTP {Method} {URL} retry {Attempt}/{MaxRetries} after {Backoff}ms",
					req.Method, req.URL.String(), attempt+1, c.retryConfig.MaxRetries, backoff.Milliseconds())

				// Close response body before retry
				if resp != nil {
					_ = resp.Body.Close()
				}

				select {
				case <-time.After(backoff):
					// Continue to next attempt
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
		}

		if lastErr != nil {
			c.logger.ErrorContext(ctx, "HTTP {Method} {URL} failed after {MaxRetries} retries: {Error}",
				req.Method, req.URL.String(), c.retryConfig.MaxRetries, lastErr)
			return nil, fmt.Errorf("after %d retries: %w", c.retryConfig.MaxRetries, lastErr)
		}

		return resp, nil
	}

	// Apply circuit breaker if configured (wraps entire retry sequence)
	if c.circuitBreaker != nil {
		return c.circuitBreaker.Execute(ctx, host, executeWithRetry)
	}

	return executeWithRetry(ctx)
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

// WithRetryConfig sets custom retry configuration
func WithRetryConfig(retryCfg *RetryConfig) Option {
	return func(cfg *Config) {
		cfg.RetryConfig = retryCfg
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(n int) Option {
	return func(cfg *Config) {
		if cfg.RetryConfig == nil {
			cfg.RetryConfig = DefaultRetryConfig()
		}
		cfg.RetryConfig.MaxRetries = n
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
