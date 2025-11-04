// Package cli provides the gonuget CLI application framework.
package cli

import "github.com/willibrandon/gonuget/cmd/gonuget/version"

// GetVersion returns formatted version information
func GetVersion() string {
	return version.Version
}

// GetFullVersion returns detailed version information
func GetFullVersion() string {
	return version.FullInfo()
}
