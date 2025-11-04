// Package version provides build-time version information for the gonuget CLI.
// Version information is injected at build time using -ldflags.
package version

import (
	"fmt"
	"runtime"
)

// Build-time variables injected via -ldflags -X
var (
	// Version is the semantic version (e.g., "v0.1.0" or "dev")
	Version = "dev"

	// Commit is the git commit SHA (short form, e.g., "a1b2c3d" or "none")
	Commit = "none"

	// Date is the build timestamp (ISO 8601 format, e.g., "2025-11-04T12:00:00Z" or "unknown")
	Date = "unknown"

	// GoVersion is the Go version used to build the binary (optional)
	GoVersion = runtime.Version()
)

// Info returns a formatted string with version information suitable for display.
// Example output: "gonuget version v0.1.0 (commit: a1b2c3d, built: 2025-11-04T12:00:00Z)"
func Info() string {
	return fmt.Sprintf("gonuget version %s (commit: %s, built: %s)",
		Version, Commit, Date)
}

// FullInfo returns detailed version information including Go version.
// Example output: "gonuget version v0.1.0 (commit: a1b2c3d, built: 2025-11-04T12:00:00Z, go: go1.23.1)"
func FullInfo() string {
	return fmt.Sprintf("gonuget version %s (commit: %s, built: %s, go: %s)",
		Version, Commit, Date, GoVersion)
}
