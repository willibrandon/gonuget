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

// Type represents the type of authentication.
type Type string

const (
	// AuthTypeNone indicates no authentication is required.
	AuthTypeNone Type = "none"
	// AuthTypeAPIKey indicates API key authentication.
	AuthTypeAPIKey Type = "apikey"
	// AuthTypeBearer indicates bearer token authentication.
	AuthTypeBearer Type = "bearer"
	// AuthTypeBasic indicates HTTP basic authentication.
	AuthTypeBasic Type = "basic"
)
