package auth

import (
	"fmt"
	"net/http"
)

// BearerAuthenticator implements bearer token authentication.
type BearerAuthenticator struct {
	token string
}

// NewBearerAuthenticator creates a new bearer token authenticator.
func NewBearerAuthenticator(token string) *BearerAuthenticator {
	return &BearerAuthenticator{
		token: token,
	}
}

// Authenticate adds the Authorization: Bearer header to the request.
func (a *BearerAuthenticator) Authenticate(req *http.Request) error {
	if a.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
	}
	return nil
}

// Type returns the authentication type.
func (a *BearerAuthenticator) Type() Type {
	return AuthTypeBearer
}
