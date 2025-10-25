// cmd/gonuget/output/console_test.go
package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestConsole_Print(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityNormal)
	c.Print("hello")
	if got := out.String(); got != "hello" {
		t.Errorf("Print() = %q, want %q", got, "hello")
	}
}

func TestConsole_Println(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityNormal)
	c.Println("hello")
	if got := out.String(); got != "hello\n" {
		t.Errorf("Println() = %q, want %q", got, "hello\n")
	}
}

func TestConsole_Printf(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityNormal)
	c.Printf("hello %s", "world")
	if got := out.String(); got != "hello world" {
		t.Errorf("Printf() = %q, want %q", got, "hello world")
	}
}

func TestConsole_Success(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityNormal)
	c.SetColors(false) // Disable colors for testing
	c.Success("operation succeeded")
	if !strings.Contains(out.String(), "operation succeeded") {
		t.Errorf("Success() output doesn't contain expected message")
	}
}

func TestConsole_Error(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	c := NewConsole(&outBuf, &errBuf, VerbosityNormal)
	c.SetColors(false) // Disable colors for testing
	c.Error("operation failed")
	got := errBuf.String()
	if !strings.Contains(got, "Error:") || !strings.Contains(got, "operation failed") {
		t.Errorf("Error() output doesn't contain expected message, got: %q", got)
	}
}

func TestConsole_Warning(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityNormal)
	c.SetColors(false) // Disable colors for testing
	c.Warning("something is wrong")
	got := out.String()
	if !strings.Contains(got, "Warning:") || !strings.Contains(got, "something is wrong") {
		t.Errorf("Warning() output doesn't contain expected message, got: %q", got)
	}
}

func TestConsole_Info(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityNormal)
	c.SetColors(false) // Disable colors for testing
	c.Info("information message")
	if !strings.Contains(out.String(), "information message") {
		t.Errorf("Info() output doesn't contain expected message")
	}
}

func TestConsole_Debug(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityDiagnostic)
	c.SetColors(false) // Disable colors for testing
	c.Debug("debug information")
	got := out.String()
	if !strings.Contains(got, "[DEBUG]") || !strings.Contains(got, "debug information") {
		t.Errorf("Debug() output doesn't contain expected message, got: %q", got)
	}
}

func TestConsole_Detail(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityDetailed)
	c.Detail("detailed information")
	if !strings.Contains(out.String(), "detailed information") {
		t.Errorf("Detail() output doesn't contain expected message")
	}
}

func TestConsole_VerbosityQuiet(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityQuiet)
	c.SetColors(false)

	// Normal messages should not appear in quiet mode
	c.Success("success message")
	c.Warning("warning message")
	c.Info("info message")
	c.Detail("detail message")
	c.Debug("debug message")

	if out.Len() != 0 {
		t.Errorf("Quiet mode should not output normal messages, got: %q", out.String())
	}

	// Errors should still appear in quiet mode
	var errBuf bytes.Buffer
	c = NewConsole(&out, &errBuf, VerbosityQuiet)
	c.SetColors(false)
	c.Error("error message")
	if !strings.Contains(errBuf.String(), "error message") {
		t.Errorf("Quiet mode should output error messages")
	}
}

func TestConsole_VerbosityNormal(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityNormal)
	c.SetColors(false)

	// Success, warning, info should appear
	c.Success("success")
	c.Warning("warning")
	c.Info("info")

	got := out.String()
	if !strings.Contains(got, "success") {
		t.Errorf("Normal mode should show success messages")
	}
	if !strings.Contains(got, "warning") {
		t.Errorf("Normal mode should show warning messages")
	}
	if !strings.Contains(got, "info") {
		t.Errorf("Normal mode should show info messages")
	}

	// Detail and debug should not appear
	out.Reset()
	c.Detail("detail")
	c.Debug("debug")
	if out.Len() != 0 {
		t.Errorf("Normal mode should not show detail/debug messages, got: %q", out.String())
	}
}

func TestConsole_VerbosityDetailed(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityDetailed)
	c.SetColors(false)

	// Detail should appear
	c.Detail("detail message")
	if !strings.Contains(out.String(), "detail message") {
		t.Errorf("Detailed mode should show detail messages")
	}

	// Debug should not appear
	out.Reset()
	c.Debug("debug")
	if out.Len() != 0 {
		t.Errorf("Detailed mode should not show debug messages, got: %q", out.String())
	}
}

func TestConsole_VerbosityDiagnostic(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityDiagnostic)
	c.SetColors(false)

	// All messages should appear
	c.Success("success")
	c.Warning("warning")
	c.Info("info")
	c.Detail("detail")
	c.Debug("debug")

	got := out.String()
	if !strings.Contains(got, "success") {
		t.Errorf("Diagnostic mode should show success messages")
	}
	if !strings.Contains(got, "detail") {
		t.Errorf("Diagnostic mode should show detail messages")
	}
	if !strings.Contains(got, "debug") {
		t.Errorf("Diagnostic mode should show debug messages")
	}
}

func TestConsole_SetGetVerbosity(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityNormal)

	if c.GetVerbosity() != VerbosityNormal {
		t.Errorf("GetVerbosity() = %v, want %v", c.GetVerbosity(), VerbosityNormal)
	}

	c.SetVerbosity(VerbosityDetailed)
	if c.GetVerbosity() != VerbosityDetailed {
		t.Errorf("After SetVerbosity(Detailed), GetVerbosity() = %v, want %v", c.GetVerbosity(), VerbosityDetailed)
	}
}

func TestConsole_SetColors(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityNormal)

	// Test enabling colors
	c.SetColors(true)
	// Colors are set, but we can't easily test the actual color output
	// We just verify the call doesn't panic

	// Test disabling colors
	c.SetColors(false)
	// Verify the call doesn't panic
}

func TestDefaultConsole(t *testing.T) {
	c := DefaultConsole()
	if c == nil {
		t.Error("DefaultConsole() returned nil")
	}
	if c.GetVerbosity() != VerbosityNormal {
		t.Errorf("DefaultConsole() verbosity = %v, want %v", c.GetVerbosity(), VerbosityNormal)
	}
}

func TestIsColorEnabled(t *testing.T) {
	// This test just verifies the function doesn't panic
	// Actual behavior depends on terminal state
	_ = IsColorEnabled()
}
