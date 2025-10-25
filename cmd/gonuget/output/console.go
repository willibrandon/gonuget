// cmd/gonuget/output/console.go
package output

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Verbosity levels
type Verbosity int

const (
	// VerbosityQuiet shows errors only
	VerbosityQuiet Verbosity = iota
	// VerbosityNormal shows errors, warnings, and key operations (default)
	VerbosityNormal
	// VerbosityDetailed shows above + progress details
	VerbosityDetailed
	// VerbosityDiagnostic shows above + HTTP requests, cache hits, timing
	VerbosityDiagnostic
)

// Console provides output abstraction
type Console struct {
	out       io.Writer
	err       io.Writer
	verbosity Verbosity
	mu        sync.Mutex
	colors    bool
}

// NewConsole creates a new console
func NewConsole(out, err io.Writer, verbosity Verbosity) *Console {
	c := &Console{
		out:       out,
		err:       err,
		verbosity: verbosity,
		colors:    IsColorEnabled(),
	}

	if !c.colors {
		DisableColors()
	}

	return c
}

// DefaultConsole creates a console with stdout/stderr and normal verbosity
func DefaultConsole() *Console {
	return NewConsole(os.Stdout, os.Stderr, VerbosityNormal)
}

// SetVerbosity sets the verbosity level
func (c *Console) SetVerbosity(v Verbosity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.verbosity = v
}

// GetVerbosity returns the current verbosity level
func (c *Console) GetVerbosity() Verbosity {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.verbosity
}

// SetColors enables or disables color output
func (c *Console) SetColors(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.colors = enabled
	if enabled {
		EnableColors()
	} else {
		DisableColors()
	}
}

// Print writes to output
func (c *Console) Print(a ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprint(c.out, a...)
}

// Println writes line to output
func (c *Console) Println(a ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintln(c.out, a...)
}

// Printf writes formatted output
func (c *Console) Printf(format string, a ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintf(c.out, format, a...)
}

// Success writes success message (green)
func (c *Console) Success(format string, a ...any) {
	if c.verbosity >= VerbosityNormal {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.colors {
			ColorSuccess.Fprintf(c.out, format+"\n", a...)
		} else {
			fmt.Fprintf(c.out, format+"\n", a...)
		}
	}
}

// Error writes error message (red)
func (c *Console) Error(format string, a ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.colors {
		ColorError.Fprintf(c.err, "Error: "+format+"\n", a...)
	} else {
		fmt.Fprintf(c.err, "Error: "+format+"\n", a...)
	}
}

// Warning writes warning message (yellow)
func (c *Console) Warning(format string, a ...any) {
	if c.verbosity >= VerbosityNormal {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.colors {
			ColorWarning.Fprintf(c.out, "Warning: "+format+"\n", a...)
		} else {
			fmt.Fprintf(c.out, "Warning: "+format+"\n", a...)
		}
	}
}

// Info writes info message (cyan)
func (c *Console) Info(format string, a ...any) {
	if c.verbosity >= VerbosityNormal {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.colors {
			ColorInfo.Fprintf(c.out, format+"\n", a...)
		} else {
			fmt.Fprintf(c.out, format+"\n", a...)
		}
	}
}

// Debug writes debug message (white)
func (c *Console) Debug(format string, a ...any) {
	if c.verbosity >= VerbosityDiagnostic {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.colors {
			ColorDebug.Fprintf(c.out, "[DEBUG] "+format+"\n", a...)
		} else {
			fmt.Fprintf(c.out, "[DEBUG] "+format+"\n", a...)
		}
	}
}

// Detail writes detailed message
func (c *Console) Detail(format string, a ...any) {
	if c.verbosity >= VerbosityDetailed {
		c.mu.Lock()
		defer c.mu.Unlock()
		fmt.Fprintf(c.out, format+"\n", a...)
	}
}
