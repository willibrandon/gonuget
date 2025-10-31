package restore

import (
	"io"
	"os"

	"golang.org/x/term"
)

// TTYDetector detects whether an io.Writer is a terminal (TTY)
// and gets its dimensions. This interface allows mocking in tests.
type TTYDetector interface {
	// IsTTY returns true if w is a terminal (not piped/redirected)
	IsTTY(w io.Writer) bool

	// GetSize returns the terminal width and height (columns, rows)
	// Returns an error if w is not a terminal or size cannot be determined
	GetSize(w io.Writer) (width, height int, err error)
}

// RealTTYDetector uses golang.org/x/term to detect real terminals
type RealTTYDetector struct{}

// IsTTY returns true if w is a terminal
func (d *RealTTYDetector) IsTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// GetSize returns the terminal dimensions
func (d *RealTTYDetector) GetSize(w io.Writer) (width, height int, err error) {
	if f, ok := w.(*os.File); ok {
		return term.GetSize(int(f.Fd()))
	}
	return 0, 0, os.ErrInvalid
}

// DefaultTTYDetector is the default detector used in production
var DefaultTTYDetector TTYDetector = &RealTTYDetector{}
