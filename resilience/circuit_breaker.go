package resilience

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the current state of the circuit breaker.
type CircuitState int

const (
	StateClosed   CircuitState = iota // Normal operation
	StateOpen                          // Failing, reject requests
	StateHalfOpen                      // Testing if service recovered
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}

var (
	// ErrCircuitOpen is returned when circuit breaker is in Open state.
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitBreakerConfig holds circuit breaker configuration.
type CircuitBreakerConfig struct {
	// MaxFailures is the number of failures before opening circuit.
	MaxFailures uint

	// Timeout is how long to wait in Open state before trying Half-Open.
	Timeout time.Duration

	// MaxHalfOpenRequests is max concurrent requests in Half-Open state.
	MaxHalfOpenRequests uint
}

// DefaultCircuitBreakerConfig returns default configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:         5,               // Open after 5 consecutive failures
		Timeout:             60 * time.Second, // Wait 60s before retry
		MaxHalfOpenRequests: 1,                // Only 1 request in half-open state
	}
}

// CircuitBreaker implements the three-state circuit breaker pattern.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	mu                sync.RWMutex
	state             CircuitState
	failures          uint
	lastFailureTime   time.Time
	halfOpenSuccesses uint
	halfOpenFailures  uint
	halfOpenActive    uint
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// CanExecute checks if a request can proceed.
func (cb *CircuitBreaker) CanExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Normal operation - allow all requests
		return nil

	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			// Transition to Half-Open
			cb.state = StateHalfOpen
			cb.halfOpenSuccesses = 0
			cb.halfOpenFailures = 0
			cb.halfOpenActive = 0
			// Fall through to HalfOpen case to increment counter
		} else {
			// Still in timeout period
			return ErrCircuitOpen
		}
		fallthrough

	case StateHalfOpen:
		// Allow limited requests to test service health
		if cb.halfOpenActive >= cb.config.MaxHalfOpenRequests {
			return ErrCircuitOpen
		}
		cb.halfOpenActive++
		return nil

	default:
		return ErrCircuitOpen
	}
}

// RecordSuccess records a successful operation.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Reset failure counter on success
		cb.failures = 0

	case StateHalfOpen:
		cb.halfOpenActive--
		cb.halfOpenSuccesses++

		// Transition back to Closed after successful test
		cb.state = StateClosed
		cb.failures = 0

	case StateOpen:
		// Should not happen, but reset if it does
		cb.state = StateClosed
		cb.failures = 0
	}
}

// RecordFailure records a failed operation.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.config.MaxFailures {
			// Transition to Open state
			cb.state = StateOpen
		}

	case StateHalfOpen:
		cb.halfOpenActive--
		cb.halfOpenFailures++
		// Any failure in half-open immediately opens circuit
		cb.state = StateOpen

	case StateOpen:
		// Already open, nothing to do
	}
}

// Reset manually resets the circuit breaker to Closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpenSuccesses = 0
	cb.halfOpenFailures = 0
	cb.halfOpenActive = 0
}

// Stats returns current circuit breaker statistics.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:             cb.state,
		Failures:          cb.failures,
		LastFailureTime:   cb.lastFailureTime,
		HalfOpenSuccesses: cb.halfOpenSuccesses,
		HalfOpenFailures:  cb.halfOpenFailures,
		HalfOpenActive:    cb.halfOpenActive,
	}
}

// CircuitBreakerStats holds circuit breaker statistics.
type CircuitBreakerStats struct {
	State             CircuitState
	Failures          uint
	LastFailureTime   time.Time
	HalfOpenSuccesses uint
	HalfOpenFailures  uint
	HalfOpenActive    uint
}
