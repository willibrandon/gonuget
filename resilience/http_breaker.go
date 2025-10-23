package resilience

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// HTTPCircuitBreaker wraps circuit breakers for HTTP operations.
// It maintains per-host circuit breakers to isolate failures.
type HTTPCircuitBreaker struct {
	config   CircuitBreakerConfig
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
}

// NewHTTPCircuitBreaker creates a new HTTP circuit breaker.
func NewHTTPCircuitBreaker(config CircuitBreakerConfig) *HTTPCircuitBreaker {
	return &HTTPCircuitBreaker{
		config:   config,
		breakers: make(map[string]*CircuitBreaker),
	}
}

// NewHTTPCircuitBreakerWithDefaults creates a circuit breaker with default config.
func NewHTTPCircuitBreakerWithDefaults() *HTTPCircuitBreaker {
	return NewHTTPCircuitBreaker(DefaultCircuitBreakerConfig())
}

// getBreaker gets or creates a circuit breaker for a host.
func (hcb *HTTPCircuitBreaker) getBreaker(host string) *CircuitBreaker {
	// Fast path: read lock
	hcb.mu.RLock()
	breaker, exists := hcb.breakers[host]
	hcb.mu.RUnlock()

	if exists {
		return breaker
	}

	// Slow path: write lock
	hcb.mu.Lock()
	defer hcb.mu.Unlock()

	// Double-check after acquiring write lock
	breaker, exists = hcb.breakers[host]
	if exists {
		return breaker
	}

	// Create new breaker
	breaker = NewCircuitBreaker(hcb.config)
	hcb.breakers[host] = breaker
	return breaker
}

// HTTPOperation is a function that performs an HTTP operation.
type HTTPOperation func(ctx context.Context) (*http.Response, error)

// Execute executes an HTTP operation with circuit breaker protection.
// The host parameter is used to isolate circuit breakers per host.
func (hcb *HTTPCircuitBreaker) Execute(ctx context.Context, host string, op HTTPOperation) (*http.Response, error) {
	breaker := hcb.getBreaker(host)

	// Check if circuit allows execution
	if err := breaker.CanExecute(); err != nil {
		return nil, fmt.Errorf("circuit breaker open for %s: %w", host, err)
	}

	// Execute operation
	resp, err := op(ctx)

	// Record result
	if err != nil {
		// Network error or other failure
		breaker.RecordFailure()
		return nil, err
	}

	// Check HTTP status code
	if resp.StatusCode >= 500 {
		// Server error - record failure
		breaker.RecordFailure()
		return resp, nil
	}

	// Success
	breaker.RecordSuccess()
	return resp, nil
}

// Reset resets the circuit breaker for a specific host.
func (hcb *HTTPCircuitBreaker) Reset(host string) {
	hcb.mu.RLock()
	breaker, exists := hcb.breakers[host]
	hcb.mu.RUnlock()

	if exists {
		breaker.Reset()
	}
}

// ResetAll resets all circuit breakers.
func (hcb *HTTPCircuitBreaker) ResetAll() {
	hcb.mu.RLock()
	defer hcb.mu.RUnlock()

	for _, breaker := range hcb.breakers {
		breaker.Reset()
	}
}

// GetState returns the state of the circuit breaker for a host.
func (hcb *HTTPCircuitBreaker) GetState(host string) CircuitState {
	hcb.mu.RLock()
	breaker, exists := hcb.breakers[host]
	hcb.mu.RUnlock()

	if !exists {
		return StateClosed // No breaker yet means closed
	}

	return breaker.State()
}

// GetStats returns statistics for a specific host's circuit breaker.
func (hcb *HTTPCircuitBreaker) GetStats(host string) CircuitBreakerStats {
	hcb.mu.RLock()
	breaker, exists := hcb.breakers[host]
	hcb.mu.RUnlock()

	if !exists {
		// Return default stats for non-existent breaker
		return CircuitBreakerStats{
			State: StateClosed,
		}
	}

	return breaker.Stats()
}

// GetAllStats returns statistics for all circuit breakers.
func (hcb *HTTPCircuitBreaker) GetAllStats() map[string]CircuitBreakerStats {
	hcb.mu.RLock()
	defer hcb.mu.RUnlock()

	stats := make(map[string]CircuitBreakerStats, len(hcb.breakers))
	for host, breaker := range hcb.breakers {
		stats[host] = breaker.Stats()
	}

	return stats
}
