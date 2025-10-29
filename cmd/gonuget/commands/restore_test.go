package commands

import (
	"bytes"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

func TestNewRestoreCommand(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewRestoreCommand(console)
	if cmd == nil {
		t.Fatal("NewRestoreCommand() returned nil")
	}

	if cmd.Use != "restore [<PROJECT|SOLUTION>]" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "restore [<PROJECT|SOLUTION>]")
	}

	if cmd.Short == "" {
		t.Error("cmd.Short is empty")
	}

	if cmd.Long == "" {
		t.Error("cmd.Long is empty")
	}
}

func TestRestoreCommand_Flags(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewRestoreCommand(console)

	tests := []struct {
		name      string
		flagName  string
		shorthand string
	}{
		{"source flag", "source", "s"},
		{"packages flag", "packages", ""},
		{"configfile flag", "configfile", ""},
		{"force flag", "force", ""},
		{"no-cache flag", "no-cache", ""},
		{"no-dependencies flag", "no-dependencies", ""},
		{"verbosity flag", "verbosity", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)
				return
			}

			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", tt.flagName, flag.Shorthand, tt.shorthand)
			}
		})
	}
}

func TestRestoreCommand_MaxArgs(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewRestoreCommand(console)
	cmd.SetArgs([]string{"arg1", "arg2"})

	if err := cmd.Execute(); err == nil {
		t.Error("Execute() should return error for more than 1 argument")
	}
}

func TestRestoreCommand_FlagDefaults(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewRestoreCommand(console)

	tests := []struct {
		name         string
		flagName     string
		expectedVal  string
		isBoolFlag   bool
		expectedBool bool
	}{
		{"verbosity default", "verbosity", "minimal", false, false}, // dotnet restore default is minimal
		{"force default", "force", "", true, false},
		{"no-cache default", "no-cache", "", true, false},
		{"no-dependencies default", "no-dependencies", "", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found", tt.flagName)
			}

			if tt.isBoolFlag {
				val := flag.Value.String()
				if (val == "true") != tt.expectedBool {
					t.Errorf("flag %q default = %q, want %t", tt.flagName, val, tt.expectedBool)
				}
			} else if flag.DefValue != tt.expectedVal {
				t.Errorf("flag %q default = %q, want %q", tt.flagName, flag.DefValue, tt.expectedVal)
			}
		})
	}
}

func TestRestoreCommand_SourcesSliceFlag(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewRestoreCommand(console)

	// Verify sources flag accepts multiple values
	sourceFlag := cmd.Flags().Lookup("source")
	if sourceFlag == nil {
		t.Fatal("source flag not found")
	}

	// Check it's a StringSlice type
	if sourceFlag.Value.Type() != "stringSlice" {
		t.Errorf("source flag type = %q, want %q", sourceFlag.Value.Type(), "stringSlice")
	}
}
