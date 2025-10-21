// Package auth provides authentication mechanisms for NuGet feeds.
package auth

import (
	"net/http"
)

// Authenticator is the interface for NuGet authentication.
type Authenticator interface {
	// Authenticate adds authentication to the request
	Authenticate(req *http.Request) error
}

// AuthType represents the type of authentication.
type AuthType string

const (
	AuthTypeNone   AuthType = "none"
	AuthTypeAPIKey AuthType = "apikey"
	AuthTypeBearer AuthType = "bearer"
	AuthTypeBasic  AuthType = "basic"
)
