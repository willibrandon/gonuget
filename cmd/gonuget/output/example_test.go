// cmd/gonuget/output/example_test.go
package output_test

import (
	"os"

	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// Example demonstrating console usage
func ExampleConsole() {
	c := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityNormal)
	c.SetColors(false) // Disable for consistent output in examples

	c.Print("Basic print\n")
	c.Println("Print with newline")
	c.Printf("Formatted %s\n", "output")
	c.Success("Operation completed successfully")
	c.Info("Information message")
	c.Warning("Warning message")

	// Output:
	// Basic print
	// Print with newline
	// Formatted output
	// Operation completed successfully
	// Information message
	// Warning: Warning message
}

// Example showing verbosity levels
func ExampleConsole_verbosity() {
	// Quiet mode - only errors
	c := output.NewConsole(os.Stdout, os.Stderr, output.VerbosityQuiet)
	c.SetColors(false)
	c.Success("This won't appear")
	c.Info("This won't appear either")
	// (no output in quiet mode for success/info)

	// Normal mode - errors, warnings, info, success
	c.SetVerbosity(output.VerbosityNormal)
	c.Info("This will appear")

	// Detailed mode - adds detail messages
	c.SetVerbosity(output.VerbosityDetailed)
	c.Detail("Detailed information")

	// Output:
	// This will appear
	// Detailed information
}
