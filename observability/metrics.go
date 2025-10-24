package observability

import (
	"net/http"

	dto "github.com/prometheus/client_model/go"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTPRequestsTotal counts HTTP requests by method, status code, and source
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_http_requests_total",
			Help: "Total number of HTTP requests by method and status",
		},
		[]string{"method", "status_code", "source"},
	)

	// HTTPRequestDuration tracks HTTP request duration in seconds
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gonuget_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to 16s
		},
		[]string{"method", "source"},
	)

	// CacheHitsTotal counts cache hits by cache tier
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_cache_hits_total",
			Help: "Total number of cache hits by cache tier",
		},
		[]string{"tier"}, // memory, disk
	)

	// CacheMissesTotal counts cache misses by cache tier
	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_cache_misses_total",
			Help: "Total number of cache misses by cache tier",
		},
		[]string{"tier"},
	)

	// CacheSizeBytes tracks current cache size in bytes by tier
	CacheSizeBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gonuget_cache_size_bytes",
			Help: "Current cache size in bytes by tier",
		},
		[]string{"tier"},
	)

	// PackageDownloadsTotal counts package downloads by status
	PackageDownloadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_package_downloads_total",
			Help: "Total number of package downloads by status",
		},
		[]string{"status"}, // success, failure
	)

	// PackageDownloadDuration tracks package download duration in seconds
	PackageDownloadDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gonuget_package_download_duration_seconds",
			Help:    "Package download duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 12), // 100ms to 6min
		},
		[]string{"package_id"},
	)

	// CircuitBreakerState tracks circuit breaker state by host
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gonuget_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"host"},
	)

	// CircuitBreakerFailures counts circuit breaker failures
	CircuitBreakerFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_circuit_breaker_failures_total",
			Help: "Total number of circuit breaker failures",
		},
		[]string{"host"},
	)

	// RateLimitRequestsTotal counts rate limited requests
	RateLimitRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gonuget_rate_limit_requests_total",
			Help: "Total number of rate limited requests",
		},
		[]string{"source", "allowed"}, // allowed: true/false
	)

	// RateLimitTokens tracks current number of available rate limit tokens
	RateLimitTokens = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gonuget_rate_limit_tokens",
			Help: "Current number of available rate limit tokens",
		},
		[]string{"source"},
	)
)

// MetricsHandler returns an HTTP handler for Prometheus metrics
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// StartMetricsServer starts an HTTP server exposing Prometheus metrics
func StartMetricsServer(addr string) error {
	http.Handle("/metrics", MetricsHandler())
	return http.ListenAndServe(addr, nil)
}

// GetCounterValue retrieves the current value of a counter metric with the given labels
// This is primarily intended for testing
func GetCounterValue(counter *prometheus.CounterVec, labels ...string) (float64, error) {
	metric, err := counter.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0, err
	}

	// Write metric to a DTO to read its value
	var pb dto.Metric
	if err := metric.Write(&pb); err != nil {
		return 0, err
	}

	if pb.Counter != nil {
		return pb.Counter.GetValue(), nil
	}

	return 0, nil
}
