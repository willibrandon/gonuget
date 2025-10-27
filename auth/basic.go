package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

// BasicAuthenticator implements HTTP basic authentication.
type BasicAuthenticator struct {
	username string
	password string
}

// NewBasicAuthenticator creates a new basic auth authenticator.
func NewBasicAuthenticator(username, password string) *BasicAuthenticator {
	return &BasicAuthenticator{
		username: username,
		password: password,
	}
}

// Authenticate adds the Authorization: Basic header to the request.
func (a *BasicAuthenticator) Authenticate(req *http.Request) error {
	if a.username != "" || a.password != "" {
		// Encode username:password as base64
		credentials := fmt.Sprintf("%s:%s", a.username, a.password)
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		req.Header.Set("Authorization", fmt.Sprintf("Basic %s", encoded))
	}
	return nil
}

// Type returns the authentication type.
func (a *BasicAuthenticator) Type() Type {
	return AuthTypeBasic
}
