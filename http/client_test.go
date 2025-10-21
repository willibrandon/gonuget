package http

import (
	"context"
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
