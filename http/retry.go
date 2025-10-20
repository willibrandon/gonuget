package http

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"net"
	"net/http"
	"syscall"
	"time"
)

const (
	DefaultMaxRetries     = 3
	DefaultInitialBackoff = 1 * time.Second
	DefaultMaxBackoff     = 30 * time.Second
	DefaultBackoffFactor  = 2.0
	DefaultJitterFactor   = 0.1
)

// RetryConfig holds retry behavior configuration
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
	JitterFactor   float64
}

// DefaultRetryConfig returns retry configuration with sensible defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:     DefaultMaxRetries,
		InitialBackoff: DefaultInitialBackoff,
		MaxBackoff:     DefaultMaxBackoff,
		BackoffFactor:  DefaultBackoffFactor,
		JitterFactor:   DefaultJitterFactor,
	}
}

// IsRetriable determines if an error should be retried
func IsRetriable(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are retriable
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Connection reset, refused, timeout
	if errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	return false
}

// IsRetriableStatus determines if an HTTP status code should be retried
func IsRetriableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, // 429
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return true
	default:
		return false
	}
}

// CalculateBackoff computes exponential backoff with jitter
func (rc *RetryConfig) CalculateBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Exponential backoff: initialBackoff * (factor ^ attempt)
	backoff := float64(rc.InitialBackoff) * math.Pow(rc.BackoffFactor, float64(attempt))

	// Cap at max backoff
	if backoff > float64(rc.MaxBackoff) {
		backoff = float64(rc.MaxBackoff)
	}

	// Add jitter: backoff * (1 ± jitterFactor)
	jitter := backoff * rc.JitterFactor * (2*rand.Float64() - 1)
	backoff += jitter

	// Ensure positive
	if backoff < 0 {
		backoff = float64(rc.InitialBackoff)
	}

	return time.Duration(backoff)
}
