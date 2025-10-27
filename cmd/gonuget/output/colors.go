// Package output provides console output formatting and colorization.
package output

import (
	"os"

	"github.com/fatih/color"
)

// Color schemes
var (
	ColorSuccess = color.New(color.FgGreen)
	ColorError   = color.New(color.FgRed)
	ColorWarning = color.New(color.FgYellow)
	ColorInfo    = color.New(color.FgCyan)
	ColorDebug   = color.New(color.FgWhite)
	ColorHeader  = color.New(color.Bold, color.FgWhite)
)

// IsColorEnabled checks if color output should be enabled
func IsColorEnabled() bool {
	// Disable colors if not a TTY
	if !isTerminal(os.Stdout) {
		return false
	}

	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "dumb" || term == "" {
		return false
	}

	return true
}

// isTerminal checks if the file is a terminal
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// DisableColors disables all color output
func DisableColors() {
	color.NoColor = true
}

// EnableColors enables color output
func EnableColors() {
	color.NoColor = false
}
