// cmd/gonuget/commands/version_test.go
package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

func TestVersionCommand(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewVersionCommand(console)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	result := out.String()
	// Version command should output version information
	if result == "" {
		t.Error("version command produced no output")
	}
	if !strings.Contains(result, "gonuget version") {
		t.Errorf("output doesn't contain 'gonuget version', got: %s", result)
	}
}

func TestVersionCommand_NoArgs(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewVersionCommand(console)
	cmd.SetArgs([]string{"extraarg"})

	if err := cmd.Execute(); err == nil {
		t.Error("Execute() should return error for extra arguments")
	}
}

func TestNewVersionCommand(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewVersionCommand(console)
	if cmd == nil {
		t.Fatal("NewVersionCommand() returned nil")
	}

	if cmd.Use != "version" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "version")
	}

	if cmd.Short == "" {
		t.Error("cmd.Short is empty")
	}
}
