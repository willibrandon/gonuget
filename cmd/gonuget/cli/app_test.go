package cli

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	Version = "1.0.0"
	if got := GetVersion(); got != "1.0.0" {
		t.Errorf("GetVersion() = %v, want %v", got, "1.0.0")
	}
}

func TestGetFullVersion(t *testing.T) {
	Version = "1.0.0"
	Commit = "abc123"
	Date = "2025-01-01"
	BuiltBy = "test"

	got := GetFullVersion()
	if got == "" {
		t.Error("GetFullVersion() returned empty string")
	}
	// Should contain version info
	if !strings.Contains(got, Version) {
		t.Errorf("GetFullVersion() doesn't contain version %s", Version)
	}
	if !strings.Contains(got, Commit) {
		t.Errorf("GetFullVersion() doesn't contain commit %s", Commit)
	}
	if !strings.Contains(got, Date) {
		t.Errorf("GetFullVersion() doesn't contain date %s", Date)
	}
	if !strings.Contains(got, BuiltBy) {
		t.Errorf("GetFullVersion() doesn't contain builtBy %s", BuiltBy)
	}
}
