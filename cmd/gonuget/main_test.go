package main

import "testing"

// TestVersionDefaults ensures version variables are initialized
func TestVersionDefaults(t *testing.T) {
	// Verify version variables are initialized to defaults
	if version == "" {
		t.Error("version should have default value")
	}
	if commit == "" {
		t.Error("commit should have default value")
	}
	if date == "" {
		t.Error("date should have default value")
	}
	if builtBy == "" {
		t.Error("builtBy should have default value")
	}
}
