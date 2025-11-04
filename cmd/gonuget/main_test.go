package main

import (
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/version"
)

// TestVersionDefaults ensures version variables are initialized
func TestVersionDefaults(t *testing.T) {
	// Verify version package variables are initialized
	if version.Version == "" {
		t.Error("version.Version should have default value")
	}
}
