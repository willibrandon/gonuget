package cli

import (
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/version"
)

func TestGetVersion(t *testing.T) {
	// Test that GetVersion returns the version package's Version
	got := GetVersion()
	if got == "" {
		t.Error("GetVersion() returned empty string")
	}
	if got != version.Version {
		t.Errorf("GetVersion() = %v, want %v", got, version.Version)
	}
}

func TestGetFullVersion(t *testing.T) {
	// Test that GetFullVersion returns a non-empty string
	got := GetFullVersion()
	if got == "" {
		t.Error("GetFullVersion() returned empty string")
	}
}
