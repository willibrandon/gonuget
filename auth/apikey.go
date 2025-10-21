package auth

import (
	"net/http"
)

// APIKeyAuthenticator implements API key authentication.
type APIKeyAuthenticator struct {
	apiKey string
}

// NewAPIKeyAuthenticator creates a new API key authenticator.
func NewAPIKeyAuthenticator(apiKey string) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		apiKey: apiKey,
	}
}

// Authenticate adds the X-NuGet-ApiKey header to the request.
func (a *APIKeyAuthenticator) Authenticate(req *http.Request) error {
	if a.apiKey != "" {
		req.Header.Set("X-NuGet-ApiKey", a.apiKey)
	}
	return nil
}

// Type returns the authentication type.
func (a *APIKeyAuthenticator) Type() AuthType {
	return AuthTypeAPIKey
}
