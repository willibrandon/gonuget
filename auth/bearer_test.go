package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBearerAuthenticator_Authenticate(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.signature"
	auth := NewBearerAuthenticator(token)

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	got := req.Header.Get("Authorization")
	want := "Bearer " + token
	if got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestBearerAuthenticator_EmptyToken(t *testing.T) {
	auth := NewBearerAuthenticator("")

	req := httptest.NewRequest("GET", "https://api.nuget.org/v3/index.json", nil)

	err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should not set header if token is empty
	got := req.Header.Get("Authorization")
	if got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
}

func TestBearerAuthenticator_Type(t *testing.T) {
	auth := NewBearerAuthenticator("test-token")

	if auth.Type() != AuthTypeBearer {
		t.Errorf("Type() = %q, want %q", auth.Type(), AuthTypeBearer)
	}
}

func TestBearerAuthenticator_RealRequest(t *testing.T) {
	token := "test-bearer-token-12345"
	auth := NewBearerAuthenticator(token)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth := r.Header.Get("Authorization")
		wantAuth := "Bearer " + token

		if gotAuth != wantAuth {
			t.Errorf("Authorization = %q, want %q", gotAuth, wantAuth)
		}

		// Verify it starts with "Bearer "
		if !strings.HasPrefix(gotAuth, "Bearer ") {
			t.Error("Authorization header missing 'Bearer ' prefix")
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
