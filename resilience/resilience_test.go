package resilience

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"
)

// Race detection tests - run with: go test -race

func TestCircuitBreaker_Concurrent(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:         5,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 2,
	}
	cb := NewCircuitBreaker(config)

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range 100 {
				_ = cb.CanExecute()
				if j%10 == 0 {
					cb.RecordSuccess()
				} else if j%3 == 0 {
					cb.RecordFailure()
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestTokenBucket_Concurrent(t *testing.T) {
	tb := NewTokenBucket(TokenBucketConfig{
		Capacity:      100,
		RefillRate:    50.0,
		InitialTokens: 100,
	})

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 50 {
				tb.Allow()
			}
		})
	}
	wg.Wait()
}

func TestHTTPCircuitBreaker_Concurrent(t *testing.T) {
	hcb := NewHTTPCircuitBreakerWithDefaults()
	ctx := context.Background()

	hosts := []string{
		"api.nuget.org",
		"pkgs.dev.azure.com",
		"github.com",
	}

	var wg sync.WaitGroup
	for _, host := range hosts {
		for i := range 10 {
			wg.Add(1)
			go func(h string, id int) {
				defer wg.Done()
				op := func(ctx context.Context) (*http.Response, error) {
					if id%3 == 0 {
						return &http.Response{StatusCode: 500}, nil
					}
					return &http.Response{StatusCode: 200}, nil
				}
				_, _ = hcb.Execute(ctx, h, op)
			}(host, i)
		}
	}
	wg.Wait()
}

func TestPerSourceLimiter_Concurrent(t *testing.T) {
	psl := NewPerSourceLimiterWithDefaults()

	sources := []string{
		"https://api.nuget.org/v3/index.json",
		"https://pkgs.dev.azure.com/example/index.json",
		"https://github.com/example/index.json",
	}

	var wg sync.WaitGroup
	for _, source := range sources {
		for range 10 {
			wg.Add(1)
			go func(s string) {
				defer wg.Done()
				for range 20 {
					psl.Allow(s)
				}
			}(source)
		}
	}
	wg.Wait()
}

// Benchmarks - run with: go test -bench=. -benchmem

func BenchmarkCircuitBreaker_CanExecute(b *testing.B) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	b.ResetTimer()
	for b.Loop() {
		_ = cb.CanExecute()
	}
}

func BenchmarkCircuitBreaker_RecordSuccess(b *testing.B) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	b.ResetTimer()
	for b.Loop() {
		cb.RecordSuccess()
	}
}

func BenchmarkCircuitBreaker_RecordFailure(b *testing.B) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	b.ResetTimer()
	for b.Loop() {
		cb.RecordFailure()
	}
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	tb := NewTokenBucket(DefaultTokenBucketConfig())
	b.ResetTimer()
	for b.Loop() {
		tb.Allow()
	}
}

func BenchmarkTokenBucket_AllowN(b *testing.B) {
	tb := NewTokenBucket(DefaultTokenBucketConfig())
	b.ResetTimer()
	for b.Loop() {
		tb.AllowN(5)
	}
}

func BenchmarkTokenBucket_Tokens(b *testing.B) {
	tb := NewTokenBucket(DefaultTokenBucketConfig())
	b.ResetTimer()
	for b.Loop() {
		tb.Tokens()
	}
}

func BenchmarkHTTPCircuitBreaker_Execute(b *testing.B) {
	hcb := NewHTTPCircuitBreakerWithDefaults()
	ctx := context.Background()
	host := "api.nuget.org"

	op := func(ctx context.Context) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = hcb.Execute(ctx, host, op)
	}
}

func BenchmarkHTTPCircuitBreaker_GetState(b *testing.B) {
	hcb := NewHTTPCircuitBreakerWithDefaults()
	host := "api.nuget.org"

	// Initialize with one request
	hcb.GetState(host)

	b.ResetTimer()
	for b.Loop() {
		hcb.GetState(host)
	}
}

func BenchmarkPerSourceLimiter_Allow(b *testing.B) {
	psl := NewPerSourceLimiterWithDefaults()
	source := "https://api.nuget.org/v3/index.json"

	b.ResetTimer()
	for b.Loop() {
		psl.Allow(source)
	}
}

func BenchmarkPerSourceLimiter_GetStats(b *testing.B) {
	psl := NewPerSourceLimiterWithDefaults()
	source := "https://api.nuget.org/v3/index.json"

	// Initialize limiter
	psl.Allow(source)

	b.ResetTimer()
	for b.Loop() {
		psl.GetStats(source)
	}
}

// Parallel benchmarks

func BenchmarkCircuitBreaker_Parallel(b *testing.B) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.CanExecute()
			cb.RecordSuccess()
		}
	})
}

func BenchmarkTokenBucket_Parallel(b *testing.B) {
	tb := NewTokenBucket(DefaultTokenBucketConfig())
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow()
		}
	})
}

func BenchmarkPerSourceLimiter_Parallel(b *testing.B) {
	psl := NewPerSourceLimiterWithDefaults()
	sources := []string{
		"https://api.nuget.org/v3/index.json",
		"https://pkgs.dev.azure.com/example/index.json",
		"https://github.com/example/index.json",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			source := sources[i%len(sources)]
			psl.Allow(source)
			i++
		}
	})
}
