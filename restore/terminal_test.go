package restore

import (
	"testing"
	"time"
)

func TestTerminalStatus_Elapsed(t *testing.T) {
	status := &TerminalStatus{
		start: time.Now(),
	}

	// Wait a tiny bit
	time.Sleep(time.Millisecond)

	elapsed := status.Elapsed()
	if elapsed <= 0 {
		t.Error("Elapsed time should be greater than 0")
	}

	if elapsed > time.Second {
		t.Errorf("Elapsed time seems too long: %v", elapsed)
	}
}

func TestTerminalStatus_IsTTY(t *testing.T) {
	tests := []struct {
		name   string
		isTTY  bool
		expect bool
	}{
		{
			name:   "TTY enabled",
			isTTY:  true,
			expect: true,
		},
		{
			name:   "TTY disabled",
			isTTY:  false,
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &TerminalStatus{
				isTTY: tt.isTTY,
			}

			if got := status.IsTTY(); got != tt.expect {
				t.Errorf("IsTTY() = %v, want %v", got, tt.expect)
			}
		})
	}
}
