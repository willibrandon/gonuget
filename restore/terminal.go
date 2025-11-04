package restore

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// TerminalStatus displays live restore status with right-aligned timer
// Matches MSBuild Terminal Logger behavior:
// - Updates at 30Hz (every ~33ms)
// - Right-aligned "Restore (X.Xs)" status
// - Hides cursor during updates to prevent flicker
type TerminalStatus struct {
	output      io.Writer
	isTTY       bool
	width       int
	ticker      *time.Ticker
	start       time.Time
	done        chan struct{}
	projectName string
	stopped     bool
	mu          sync.Mutex // Protects concurrent writes to output
}

// NewTerminalStatus creates a new terminal status updater
// If detector is nil, DefaultTTYDetector is used
func NewTerminalStatus(output io.Writer, projectName string, detector TTYDetector) *TerminalStatus {
	// Use default detector if none provided
	if detector == nil {
		detector = DefaultTTYDetector
	}

	// Check if output is a TTY
	isTTY := detector.IsTTY(output)
	width := 120
	if isTTY {
		if w, _, err := detector.GetSize(output); err == nil && w > 0 {
			width = w
		}
	}

	t := &TerminalStatus{
		output:      output,
		isTTY:       isTTY,
		width:       width,
		start:       time.Now(),
		done:        make(chan struct{}),
		projectName: projectName,
	}

	if isTTY {
		t.ticker = time.NewTicker(33 * time.Millisecond) // 30Hz
		go t.updateLoop()
	}

	return t
}

// updateLoop runs in background, updating status at 30Hz
func (t *TerminalStatus) updateLoop() {
	for {
		select {
		case <-t.ticker.C:
			t.updateStatus()
		case <-t.done:
			return
		}
	}
}

// updateStatus writes the right-aligned status to terminal
func (t *TerminalStatus) updateStatus() {
	elapsed := time.Since(t.start).Seconds()

	// Format: "Restore (X.Xs)"
	status := fmt.Sprintf("Restore (%.1fs)", elapsed)

	// Calculate positioning:
	// 1. Move cursor to column 120 (or terminal width)
	// 2. Move backward by status length
	// 3. Write status
	// 4. Carriage return to beginning of line

	column := min(t.width, 120) // MSBuild uses max 120

	backwardCount := len(status)

	// Hide cursor, position, write, show cursor
	t.mu.Lock()
	_, _ = fmt.Fprintf(t.output, "\x1B[?25l\x1B[%dG\x1B[%dD%s\r\x1B[?25h",
		column, backwardCount, status)
	t.mu.Unlock()
}

// Stop stops the status updater and clears the status line
// Safe to call multiple times
func (t *TerminalStatus) Stop() {
	if t.stopped {
		return
	}
	t.stopped = true

	if t.ticker != nil {
		t.ticker.Stop()
		close(t.done)
	}

	if t.isTTY {
		// Clear to end of line
		t.mu.Lock()
		_, _ = fmt.Fprint(t.output, "\x1B[K")
		t.mu.Unlock()
	}
}

// Elapsed returns the elapsed time since start
func (t *TerminalStatus) Elapsed() time.Duration {
	return time.Since(t.start)
}

// IsTTY returns true if output is a terminal (not piped/redirected)
func (t *TerminalStatus) IsTTY() bool {
	return t.isTTY
}
