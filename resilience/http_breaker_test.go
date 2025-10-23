package resilience

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestHTTPCircuitBreaker_Success(t *testing.T) {
	hcb := NewHTTPCircuitBreakerWithDefaults()
	ctx := context.Background()
	host := "api.nuget.org"

	// Successful operation
	op := func(ctx context.Context) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}

	resp, err := hcb.Execute(ctx, host, op)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	// Circuit should be closed
	if state := hcb.GetState(host); state != StateClosed {
		t.Errorf("State = %v, want Closed", state)
	}
}

func TestHTTPCircuitBreaker_NetworkError(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	hcb := NewHTTPCircuitBreaker(config)
	ctx := context.Background()
	host := "api.nuget.org"

	// Failing operation
	networkErr := errors.New("network error")
	op := func(ctx context.Context) (*http.Response, error) {
		return nil, networkErr
	}

	// Record failures
	for i := uint(0); i < config.MaxFailures; i++ {
		resp, err := hcb.Execute(ctx, host, op)
		if err != networkErr {
			t.Errorf("Execute() error = %v, want %v", err, networkErr)
		}
		if resp != nil {
			t.Errorf("Execute() response = %v, want nil", resp)
		}
	}

	// Circuit should be open now
	if state := hcb.GetState(host); state != StateOpen {
		t.Errorf("State after failures = %v, want Open", state)
	}

	// Next request should be rejected
	_, err := hcb.Execute(ctx, host, op)
	if err == nil {
		t.Error("Execute() with open circuit should return error")
	}
}

func TestHTTPCircuitBreaker_ServerError(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	hcb := NewHTTPCircuitBreaker(config)
	ctx := context.Background()
	host := "api.nuget.org"

	// Operation returning 500 error
	op := func(ctx context.Context) (*http.Response, error) {
		return &http.Response{StatusCode: 500}, nil
	}

	// Record failures
	for i := uint(0); i < config.MaxFailures; i++ {
		resp, err := hcb.Execute(ctx, host, op)
		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
		if resp.StatusCode != 500 {
			t.Errorf("StatusCode = %d, want 500", resp.StatusCode)
		}
	}

	// Circuit should be open
	if state := hcb.GetState(host); state != StateOpen {
		t.Errorf("State after 5xx errors = %v, want Open", state)
	}
}

func TestHTTPCircuitBreaker_PerHost_Isolation(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	hcb := NewHTTPCircuitBreaker(config)
	ctx := context.Background()

	host1 := "api.nuget.org"
	host2 := "api.example.com"

	// Fail host1
	failOp := func(ctx context.Context) (*http.Response, error) {
		return nil, errors.New("network error")
	}

	for i := uint(0); i < config.MaxFailures; i++ {
		_, err := hcb.Execute(ctx, host1, failOp)
		if err == nil {
			t.Fatal("Expected error from failOp")
		}
	}

	// host1 should be open
	if state := hcb.GetState(host1); state != StateOpen {
		t.Errorf("host1 state = %v, want Open", state)
	}

	// host2 should still be closed
	if state := hcb.GetState(host2); state != StateClosed {
		t.Errorf("host2 state = %v, want Closed", state)
	}

	// host2 should still accept requests
	successOp := func(ctx context.Context) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}

	resp, err := hcb.Execute(ctx, host2, successOp)
	if err != nil {
		t.Errorf("Execute() on host2 error = %v, want nil", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("host2 StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestHTTPCircuitBreaker_Reset(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	hcb := NewHTTPCircuitBreaker(config)
	ctx := context.Background()
	host := "api.nuget.org"

	// Open the circuit
	failOp := func(ctx context.Context) (*http.Response, error) {
		return nil, errors.New("network error")
	}

	for i := uint(0); i < config.MaxFailures; i++ {
		_, err := hcb.Execute(ctx, host, failOp)
		if err == nil {
			t.Fatal("Expected error from failOp")
		}
	}

	if state := hcb.GetState(host); state != StateOpen {
		t.Fatalf("State = %v, want Open", state)
	}

	// Reset the circuit
	hcb.Reset(host)

	if state := hcb.GetState(host); state != StateClosed {
		t.Errorf("State after Reset = %v, want Closed", state)
	}

	// Should accept requests again
	successOp := func(ctx context.Context) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}

	resp, err := hcb.Execute(ctx, host, successOp)
	if err != nil {
		t.Errorf("Execute() after Reset error = %v, want nil", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestHTTPCircuitBreaker_Stats(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	hcb := NewHTTPCircuitBreaker(config)
	ctx := context.Background()

	host1 := "api.nuget.org"
	host2 := "api.example.com"

	// Record some failures on host1
	failOp := func(ctx context.Context) (*http.Response, error) {
		return nil, errors.New("network error")
	}

	_, err := hcb.Execute(ctx, host1, failOp)
	if err == nil {
		t.Fatal("Expected error from failOp")
	}
	_, err = hcb.Execute(ctx, host1, failOp)
	if err == nil {
		t.Fatal("Expected error from failOp")
	}

	// Success on host2
	successOp := func(ctx context.Context) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}

	_, err = hcb.Execute(ctx, host2, successOp)
	if err != nil {
		t.Fatalf("Execute() on host2 error = %v", err)
	}

	// Check individual stats
	stats1 := hcb.GetStats(host1)
	if stats1.State != StateClosed {
		t.Errorf("host1 State = %v, want Closed", stats1.State)
	}
	if stats1.Failures != 2 {
		t.Errorf("host1 Failures = %d, want 2", stats1.Failures)
	}

	stats2 := hcb.GetStats(host2)
	if stats2.State != StateClosed {
		t.Errorf("host2 State = %v, want Closed", stats2.State)
	}
	if stats2.Failures != 0 {
		t.Errorf("host2 Failures = %d, want 0", stats2.Failures)
	}

	// Check all stats
	allStats := hcb.GetAllStats()
	if len(allStats) != 2 {
		t.Errorf("GetAllStats() returned %d hosts, want 2", len(allStats))
	}

	if _, exists := allStats[host1]; !exists {
		t.Error("GetAllStats() missing host1")
	}
	if _, exists := allStats[host2]; !exists {
		t.Error("GetAllStats() missing host2")
	}
}

func TestHTTPCircuitBreaker_ResetAll(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	hcb := NewHTTPCircuitBreaker(config)
	ctx := context.Background()

	host1 := "api.nuget.org"
	host2 := "api.example.com"

	// Open both circuits
	failOp := func(ctx context.Context) (*http.Response, error) {
		return nil, errors.New("network error")
	}

	for i := uint(0); i < config.MaxFailures; i++ {
		_, err := hcb.Execute(ctx, host1, failOp)
		if err == nil {
			t.Fatal("Expected error from failOp")
		}
		_, err = hcb.Execute(ctx, host2, failOp)
		if err == nil {
			t.Fatal("Expected error from failOp")
		}
	}

	// Both should be open
	if state := hcb.GetState(host1); state != StateOpen {
		t.Errorf("host1 state before ResetAll = %v, want Open", state)
	}
	if state := hcb.GetState(host2); state != StateOpen {
		t.Errorf("host2 state before ResetAll = %v, want Open", state)
	}

	// Reset all
	hcb.ResetAll()

	// Both should be closed
	if state := hcb.GetState(host1); state != StateClosed {
		t.Errorf("host1 state after ResetAll = %v, want Closed", state)
	}
	if state := hcb.GetState(host2); state != StateClosed {
		t.Errorf("host2 state after ResetAll = %v, want Closed", state)
	}
}

func TestHTTPCircuitBreaker_GetStats_NonExistentHost(t *testing.T) {
	hcb := NewHTTPCircuitBreakerWithDefaults()

	// Get stats for host that doesn't exist yet
	stats := hcb.GetStats("nonexistent.example.com")

	if stats.State != StateClosed {
		t.Errorf("State for nonexistent host = %v, want Closed", stats.State)
	}
	if stats.Failures != 0 {
		t.Errorf("Failures for nonexistent host = %d, want 0", stats.Failures)
	}
}
