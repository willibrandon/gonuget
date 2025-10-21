package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyAuthenticator_Authenticate(t *testing.T) {
	apiKey := "test-api-key-12345"
	auth := NewAPIKeyAuthenticator(apiKey)

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	got := req.Header.Get("X-NuGet-ApiKey")
	if got != apiKey {
		t.Errorf("X-NuGet-ApiKey = %q, want %q", got, apiKey)
	}
}

func TestAPIKeyAuthenticator_EmptyKey(t *testing.T) {
	auth := NewAPIKeyAuthenticator("")

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should not set header if key is empty
	got := req.Header.Get("X-NuGet-ApiKey")
	if got != "" {
		t.Errorf("X-NuGet-ApiKey = %q, want empty", got)
	}
}

func TestAPIKeyAuthenticator_Type(t *testing.T) {
	auth := NewAPIKeyAuthenticator("test-key")

	if auth.Type() != AuthTypeAPIKey {
		t.Errorf("Type() = %q, want %q", auth.Type(), AuthTypeAPIKey)
	}
}

func TestAPIKeyAuthenticator_RealRequest(t *testing.T) {
	apiKey := "test-api-key"
	auth := NewAPIKeyAuthenticator(apiKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey := r.Header.Get("X-NuGet-ApiKey")
		if gotKey != apiKey {
			t.Errorf("X-NuGet-ApiKey = %q, want %q", gotKey, apiKey)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	err = auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}
