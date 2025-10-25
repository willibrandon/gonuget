// cmd/gonuget/commands/version_test.go
package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/cli"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

func TestVersionCommand(t *testing.T) {
	// Set version info for test
	cli.Version = "1.0.0"
	cli.Commit = "abc123"
	cli.Date = "2025-01-01"
	cli.BuiltBy = "test"

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewVersionCommand(console)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "1.0.0") {
		t.Errorf("output doesn't contain version, got: %s", result)
	}
	if !strings.Contains(result, "abc123") {
		t.Errorf("output doesn't contain commit, got: %s", result)
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
