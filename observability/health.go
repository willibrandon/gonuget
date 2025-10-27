package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	// HealthStatusHealthy indicates the service is healthy.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded indicates the service is degraded.
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusUnhealthy indicates the service is unhealthy.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name   string
	Check  func(context.Context) HealthCheckResult
	Cached bool
	TTL    time.Duration
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status  HealthStatus      `json:"status"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// HealthChecker manages and executes health checks
type HealthChecker struct {
	mu     sync.RWMutex
	checks map[string]*HealthCheck
	cache  map[string]*cachedHealthResult
}

type cachedHealthResult struct {
	result    HealthCheckResult
	timestamp time.Time
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]*HealthCheck),
		cache:  make(map[string]*cachedHealthResult),
	}
}

// Register registers a new health check
func (hc *HealthChecker) Register(check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[check.Name] = &check
}

// Check executes all health checks and returns aggregate status
func (hc *HealthChecker) Check(ctx context.Context) map[string]HealthCheckResult {
	hc.mu.RLock()
	checks := make([]*HealthCheck, 0, len(hc.checks))
	for _, check := range hc.checks {
		checks = append(checks, check)
	}
	hc.mu.RUnlock()

	results := make(map[string]HealthCheckResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, check := range checks {
		wg.Add(1)
		go func(c *HealthCheck) {
			defer wg.Done()
			result := hc.executeCheck(ctx, c)
			mu.Lock()
			results[c.Name] = result
			mu.Unlock()
		}(check)
	}

	wg.Wait()
	return results
}

// executeCheck executes a single health check with caching
func (hc *HealthChecker) executeCheck(ctx context.Context, check *HealthCheck) HealthCheckResult {
	// Check cache if enabled
	if check.Cached {
		hc.mu.RLock()
		cached, exists := hc.cache[check.Name]
		hc.mu.RUnlock()

		if exists && time.Since(cached.timestamp) < check.TTL {
			return cached.result
		}
	}

	// Execute check
	result := check.Check(ctx)

	// Cache result if enabled
	if check.Cached {
		hc.mu.Lock()
		hc.cache[check.Name] = &cachedHealthResult{
			result:    result,
			timestamp: time.Now(),
		}
		hc.mu.Unlock()
	}

	return result
}

// OverallStatus returns the aggregate health status
func (hc *HealthChecker) OverallStatus(ctx context.Context) HealthStatus {
	results := hc.Check(ctx)

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		switch result.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return HealthStatusUnhealthy
	}
	if hasDegraded {
		return HealthStatusDegraded
	}
	// HealthStatusHealthy indicates the service is healthy.
	return HealthStatusHealthy
}

// Handler returns an HTTP handler for health checks
func (hc *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		results := hc.Check(ctx)
		overall := hc.OverallStatus(ctx)

		response := map[string]any{
			"status": overall,
			"checks": results,
		}

		w.Header().Set("Content-Type", "application/json")

		// Set status code based on health
		switch overall {
		// HealthStatusHealthy indicates the service is healthy.
		case HealthStatusHealthy:
			w.WriteHeader(http.StatusOK)
		case HealthStatusDegraded:
			w.WriteHeader(http.StatusOK) // Still operational
		case HealthStatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Log error but don't fail request - response may be partially written
			return
		}
	}
}

// HTTPSourceHealthCheck creates a health check for an HTTP source
func HTTPSourceHealthCheck(name, url string, timeout time.Duration) HealthCheck {
	return HealthCheck{
		Name:   name,
		Cached: true,
		TTL:    30 * time.Second,
		Check: func(ctx context.Context) HealthCheckResult {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
			if err != nil {
				return HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: "failed to create request: " + err.Error(),
				}
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: "request failed: " + err.Error(),
				}
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode >= 500 {
				return HealthCheckResult{
					Status:  HealthStatusDegraded,
					Message: "server error",
					Details: map[string]string{"status_code": resp.Status},
				}
			}

			return HealthCheckResult{
				// HealthStatusHealthy indicates the service is healthy.
				Status:  HealthStatusHealthy,
				Message: "source reachable",
			}
		},
	}
}

// CacheHealthCheck creates a health check for cache availability
func CacheHealthCheck(name string, sizeBytes int64, maxSizeBytes int64) HealthCheck {
	return HealthCheck{
		Name:   name,
		Cached: false, // Always fresh
		Check: func(ctx context.Context) HealthCheckResult {
			usagePercent := float64(sizeBytes) / float64(maxSizeBytes) * 100

			if usagePercent >= 95 {
				return HealthCheckResult{
					Status:  HealthStatusDegraded,
					Message: "cache nearly full",
					Details: map[string]string{
						"usage_percent": fmt.Sprintf("%.1f%%", usagePercent),
					},
				}
			}

			return HealthCheckResult{
				// HealthStatusHealthy indicates the service is healthy.
				Status:  HealthStatusHealthy,
				Message: "cache operational",
			}
		},
	}
}
