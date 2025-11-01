package commands_test

import (
	"os/exec"
	"testing"
	"time"
)

// TestErrorMessageTiming validates that error messages are returned within 50ms
// Per spec VR-014: "Error message timing: <50ms from command execution to error display"
func TestErrorMessageTiming(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"add package", []string{"add", "package", "Newtonsoft.Json"}},
		{"list source", []string{"list", "source"}},
		{"enable", []string{"enable", "test"}},
		{"unknown command", []string{"nonexistent", "command"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			cmd := exec.Command(getGonugetPath(), tt.args...)
			_, _ = cmd.CombinedOutput() // We expect errors, so ignore the error return

			elapsed := time.Since(start)

			// Per spec VR-014: error messages must be displayed within 50ms
			maxDuration := 50 * time.Millisecond
			if elapsed > maxDuration {
				t.Errorf("Error message took %v, expected <%v (VR-014 requirement)", elapsed, maxDuration)
			}
		})
	}
}

// TestHelpOutputTiming validates that help output is returned within 50ms
// Per spec VR-014: "Help output timing: <50ms"
func TestHelpOutputTiming(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"root help", []string{"--help"}},
		{"package help", []string{"package", "--help"}},
		{"source help", []string{"source", "--help"}},
		{"package add help", []string{"package", "add", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			cmd := exec.Command(getGonugetPath(), tt.args...)
			_, err := cmd.CombinedOutput()

			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("Help command failed: %v", err)
			}

			// Per spec VR-014: help output must be displayed within 50ms
			maxDuration := 50 * time.Millisecond
			if elapsed > maxDuration {
				t.Errorf("Help output took %v, expected <%v (VR-014 requirement)", elapsed, maxDuration)
			}
		})
	}
}

// BenchmarkErrorDetection measures the performance of error detection
func BenchmarkErrorDetection(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		cmd := exec.Command(getGonugetPath(), "add", "package", "Test")
		_, _ = cmd.CombinedOutput()
	}
}

// BenchmarkHelpOutput measures the performance of help output
func BenchmarkHelpOutput(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		cmd := exec.Command(getGonugetPath(), "package", "--help")
		_, _ = cmd.CombinedOutput()
	}
}
