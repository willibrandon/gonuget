package http

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, DefaultTimeout)
	}
	if cfg.UserAgent != DefaultUserAgent {
		t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, DefaultUserAgent)
	}
	if !cfg.EnableHTTP2 {
		t.Error("EnableHTTP2 = false, want true")
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want string
	}{
		{
			name: "nil config uses defaults",
			cfg:  nil,
			want: DefaultUserAgent,
		},
		{
			name: "custom user agent",
			cfg:  &Config{UserAgent: "custom/1.0"},
			want: "custom/1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.cfg)
			if client.userAgent != tt.want {
				t.Errorf("userAgent = %q, want %q", client.userAgent, tt.want)
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		ua := r.Header.Get("User-Agent")
		if ua != DefaultUserAgent {
			t.Errorf("User-Agent = %q, want %q", ua, DefaultUserAgent)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewClient(nil)
	ctx := context.Background()

	resp, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithOptions(WithTimeout(50 * time.Millisecond))
	ctx := context.Background()

	_, err := client.Get(ctx, server.URL)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestFunctionalOptions(t *testing.T) {
	client := NewClientWithOptions(
		WithTimeout(5*time.Second),
		WithUserAgent("test/1.0"),
		WithMaxIdleConns(50),
	)

	if client.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", client.timeout)
	}
	if client.userAgent != "test/1.0" {
		t.Errorf("userAgent = %q, want test/1.0", client.userAgent)
	}
}

func TestSetUserAgent(t *testing.T) {
	client := NewClient(nil)
	if client.userAgent != DefaultUserAgent {
		t.Errorf("initial userAgent = %q, want %q", client.userAgent, DefaultUserAgent)
	}

	client.SetUserAgent("custom/2.0")
	if client.userAgent != "custom/2.0" {
		t.Errorf("after SetUserAgent, userAgent = %q, want custom/2.0", client.userAgent)
	}
}

func TestWithTLSConfig(t *testing.T) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	client := NewClientWithOptions(
		WithTLSConfig(tlsConfig),
	)

	// Verify TLS config is set on the transport
	transport := client.httpClient.Transport.(*http.Transport)
	if transport.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig is nil")
	}
	if transport.TLSClientConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %d, want %d (TLS 1.3)", transport.TLSClientConfig.MinVersion, tls.VersionTLS13)
	}
}

func TestWithRetryConfig(t *testing.T) {
	customRetry := &RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  3.0,
		JitterFactor:   0.5,
	}

	client := NewClientWithOptions(
		WithRetryConfig(customRetry),
	)

	if client.retryConfig.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", client.retryConfig.MaxRetries)
	}
	if client.retryConfig.InitialBackoff != 500*time.Millisecond {
		t.Errorf("InitialBackoff = %v, want 500ms", client.retryConfig.InitialBackoff)
	}
	if client.retryConfig.BackoffFactor != 3.0 {
		t.Errorf("BackoffFactor = %f, want 3.0", client.retryConfig.BackoffFactor)
	}
}

func TestWithMaxRetries(t *testing.T) {
	t.Run("with existing retry config", func(t *testing.T) {
		customRetry := &RetryConfig{
			MaxRetries:     1,
			InitialBackoff: 100 * time.Millisecond,
		}

		client := NewClientWithOptions(
			WithRetryConfig(customRetry),
			WithMaxRetries(10),
		)

		if client.retryConfig.MaxRetries != 10 {
			t.Errorf("MaxRetries = %d, want 10", client.retryConfig.MaxRetries)
		}
		// Verify other fields preserved
		if client.retryConfig.InitialBackoff != 100*time.Millisecond {
			t.Errorf("InitialBackoff = %v, want 100ms (should be preserved)", client.retryConfig.InitialBackoff)
		}
	})

	t.Run("without existing retry config", func(t *testing.T) {
		client := NewClientWithOptions(
			WithMaxRetries(7),
		)

		if client.retryConfig.MaxRetries != 7 {
			t.Errorf("MaxRetries = %d, want 7", client.retryConfig.MaxRetries)
		}
		// Should use default values for other fields
		defaultRetry := DefaultRetryConfig()
		if client.retryConfig.InitialBackoff != defaultRetry.InitialBackoff {
			t.Errorf("InitialBackoff = %v, want %v (default)", client.retryConfig.InitialBackoff, defaultRetry.InitialBackoff)
		}
	})
}
