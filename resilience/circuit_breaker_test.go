package resilience

import (
	"testing"
	"time"
)

func TestCircuitBreaker_StateClosed(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Should be closed initially
	if cb.State() != StateClosed {
		t.Errorf("Initial state = %v, want Closed", cb.State())
	}

	// Should allow execution
	if err := cb.CanExecute(); err != nil {
		t.Errorf("CanExecute() in Closed state returned error: %v", err)
	}

	// Record successes - should stay closed
	cb.RecordSuccess()
	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Errorf("State after successes = %v, want Closed", cb.State())
	}
}

func TestCircuitBreaker_StateTransition_ClosedToOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Record failures until circuit opens
	for i := uint(0); i < config.MaxFailures; i++ {
		if cb.State() != StateClosed {
			t.Fatalf("State before failure %d = %v, want Closed", i+1, cb.State())
		}
		cb.RecordFailure()
	}

	// Should be open now
	if cb.State() != StateOpen {
		t.Errorf("State after %d failures = %v, want Open", config.MaxFailures, cb.State())
	}

	// Should reject execution
	if err := cb.CanExecute(); err != ErrCircuitOpen {
		t.Errorf("CanExecute() in Open state = %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreaker_StateTransition_OpenToHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := uint(0); i < config.MaxFailures; i++ {
		cb.RecordFailure()
	}

	if cb.State() != StateOpen {
		t.Fatalf("State after failures = %v, want Open", cb.State())
	}

	// Wait for timeout
	time.Sleep(config.Timeout + 10*time.Millisecond)

	// Should transition to Half-Open on next CanExecute
	if err := cb.CanExecute(); err != nil {
		t.Errorf("CanExecute() after timeout = %v, want nil", err)
	}

	if cb.State() != StateHalfOpen {
		t.Errorf("State after timeout = %v, want HalfOpen", cb.State())
	}
}

func TestCircuitBreaker_StateTransition_HalfOpenToClosed(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := uint(0); i < config.MaxFailures; i++ {
		cb.RecordFailure()
	}

	// Wait and transition to Half-Open
	time.Sleep(config.Timeout + 10*time.Millisecond)
	if err := cb.CanExecute(); err != nil {
		t.Fatalf("CanExecute() after timeout = %v, want nil", err)
	}

	if cb.State() != StateHalfOpen {
		t.Fatalf("State = %v, want HalfOpen", cb.State())
	}

	// Record success - should close circuit
	cb.RecordSuccess()

	if cb.State() != StateClosed {
		t.Errorf("State after success in HalfOpen = %v, want Closed", cb.State())
	}
}

func TestCircuitBreaker_StateTransition_HalfOpenToOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := uint(0); i < config.MaxFailures; i++ {
		cb.RecordFailure()
	}

	// Wait and transition to Half-Open
	time.Sleep(config.Timeout + 10*time.Millisecond)
	if err := cb.CanExecute(); err != nil {
		t.Fatalf("CanExecute() after timeout = %v, want nil", err)
	}

	if cb.State() != StateHalfOpen {
		t.Fatalf("State = %v, want HalfOpen", cb.State())
	}

	// Record failure - should reopen circuit
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Errorf("State after failure in HalfOpen = %v, want Open", cb.State())
	}
}

func TestCircuitBreaker_HalfOpen_MaxRequests(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             50 * time.Millisecond,
		MaxHalfOpenRequests: 2,
	}
	cb := NewCircuitBreaker(config)

	// Open circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Transition to Half-Open
	time.Sleep(config.Timeout + 10*time.Millisecond)

	// Should allow MaxHalfOpenRequests
	for i := uint(0); i < config.MaxHalfOpenRequests; i++ {
		if err := cb.CanExecute(); err != nil {
			t.Errorf("CanExecute() request %d = %v, want nil", i+1, err)
		}
	}

	// Next request should be rejected
	if err := cb.CanExecute(); err != ErrCircuitOpen {
		t.Errorf("CanExecute() beyond max = %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config)

	// Open circuit
	for i := uint(0); i < config.MaxFailures; i++ {
		cb.RecordFailure()
	}

	if cb.State() != StateOpen {
		t.Fatalf("State after failures = %v, want Open", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("State after Reset() = %v, want Closed", cb.State())
	}

	// Should allow execution
	if err := cb.CanExecute(); err != nil {
		t.Errorf("CanExecute() after Reset() = %v, want nil", err)
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := NewCircuitBreaker(config)

	// Record some failures
	cb.RecordFailure()
	cb.RecordFailure()

	stats := cb.Stats()

	if stats.State != StateClosed {
		t.Errorf("Stats.State = %v, want Closed", stats.State)
	}
	if stats.Failures != 2 {
		t.Errorf("Stats.Failures = %d, want 2", stats.Failures)
	}
	if stats.LastFailureTime.IsZero() {
		t.Error("Stats.LastFailureTime should be set")
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{StateClosed, "Closed"},
		{StateOpen, "Open"},
		{StateHalfOpen, "HalfOpen"},
		{CircuitState(999), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("CircuitState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
