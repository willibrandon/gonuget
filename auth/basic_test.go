package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBasicAuthenticator_Authenticate(t *testing.T) {
	username := "testuser"
	password := "testpass"
	auth := NewBasicAuthenticator(username, password)

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	got := req.Header.Get("Authorization")
	if got == "" {
		t.Fatal("Authorization header not set")
	}

	// Should start with "Basic "
	if !strings.HasPrefix(got, "Basic ") {
		t.Errorf("Authorization = %q, want prefix 'Basic '", got)
	}

	// Decode and verify
	encoded := strings.TrimPrefix(got, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}

	want := username + ":" + password
	if string(decoded) != want {
		t.Errorf("decoded credentials = %q, want %q", decoded, want)
	}
}

func TestBasicAuthenticator_EmptyCredentials(t *testing.T) {
	auth := NewBasicAuthenticator("", "")

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should still set header even if empty (edge case)
	// Some implementations might want this behavior
	got := req.Header.Get("Authorization")
	if got != "" {
		// Verify it's properly encoded empty credentials
		encoded := strings.TrimPrefix(got, "Basic ")
		decoded, _ := base64.StdEncoding.DecodeString(encoded)
		if string(decoded) != ":" {
			t.Errorf("decoded = %q, want ':'", decoded)
		}
	}
}

func TestBasicAuthenticator_OnlyUsername(t *testing.T) {
	auth := NewBasicAuthenticator("testuser", "")

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	got := req.Header.Get("Authorization")
	encoded := strings.TrimPrefix(got, "Basic ")
	decoded, _ := base64.StdEncoding.DecodeString(encoded)

	want := "testuser:"
	if string(decoded) != want {
		t.Errorf("decoded = %q, want %q", decoded, want)
	}
}

func TestBasicAuthenticator_Type(t *testing.T) {
	auth := NewBasicAuthenticator("user", "pass")

	if auth.Type() != AuthTypeBasic {
		t.Errorf("Type() = %q, want %q", auth.Type(), AuthTypeBasic)
	}
}

func TestBasicAuthenticator_RealRequest(t *testing.T) {
	username := "testuser"
	password := "testpass"
	auth := NewBasicAuthenticator(username, password)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth := r.Header.Get("Authorization")

		// Verify format
		if !strings.HasPrefix(gotAuth, "Basic ") {
			t.Error("Authorization header missing 'Basic ' prefix")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Decode and verify credentials
		encoded := strings.TrimPrefix(gotAuth, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Errorf("base64 decode error = %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		want := username + ":" + password
		if string(decoded) != want {
			t.Errorf("credentials = %q, want %q", decoded, want)
			w.WriteHeader(http.StatusUnauthorized)
			return
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}
