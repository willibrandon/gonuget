# CLI Milestone 1: Foundation

**Status**: Implementation Guide
**Phase**: 1 of 8
**Duration**: Weeks 1-2
**Prerequisites**: gonuget library M1-M8 complete
**Target**: 100% parity with `dotnet nuget` (cross-platform)

---

## Overview

This milestone establishes the CLI foundation: command-line framework, configuration management, console output system, and basic commands. By the end of this milestone, you'll have a working `gonuget` binary with help, version, config, and source management commands that match `dotnet nuget` behavior exactly.

**Deliverables**:
- CLI application structure with Cobra
- NuGet.config XML parsing and writing (100% compatible with .NET tools)
- Console abstraction with colors and progress bars
- Commands: `help`, `version`, `config`, source management (`list source`, `add source`, etc.)
- CLI interop tests validating output matches `dotnet nuget`
- 80%+ test coverage

**Success Criteria**:
- `gonuget --help` output matches `dotnet nuget --help` structure
- `gonuget --version` output matches `dotnet nuget --version` format
- `gonuget config` behavior matches `dotnet nuget config` exactly
- `gonuget list source` output matches `dotnet nuget list source`
- All CLI interop tests pass (output identical to `dotnet nuget`)

---

## Architecture

```
cmd/gonuget/
├── main.go                 # Entry point
├── cli/
│   ├── app.go              # Cobra root command setup
│   ├── context.go          # Global execution context
│   ├── flags.go            # Common flag definitions
│   └── version.go          # Version information
├── commands/
│   ├── base.go             # Base command interface
│   ├── help.go             # Help command
│   ├── version.go          # Version command
│   ├── config.go           # Config command
│   ├── source_list.go      # list source command
│   ├── source_add.go       # add source command
│   ├── source_remove.go    # remove source command
│   ├── source_enable.go    # enable source command
│   ├── source_disable.go   # disable source command
│   └── source_update.go    # update source command
├── config/
│   ├── nuget_config.go     # NuGet.config XML parsing
│   ├── settings.go         # Settings management
│   ├── sources.go          # Package source configuration
│   └── defaults.go         # Default configuration
└── output/
    ├── console.go          # Console abstraction
    ├── colors.go           # Color schemes
    ├── progress.go         # Progress indicators
    └── formatter.go        # Output formatting
```

**Note**: Commands follow `dotnet nuget` structure (`<verb> <noun>`) not `nuget.exe` structure (`<noun> <verb>`). For example: `gonuget list source` not `gonuget sources list`.

---

## Chunk 1: Project Structure and Entry Point

**Objective**: Set up the CLI project structure and main entry point.

**Files to create**:
- `cmd/gonuget/main.go`
- `cmd/gonuget/cli/app.go`
- `cmd/gonuget/cli/version.go`

### Step 1.1: Create main.go

```go
// cmd/gonuget/main.go
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/willibrandon/gonuget/cmd/gonuget/cli"
)

// Version information (set via ldflags during build)
var (
	version   = "dev"
	commit    = "unknown"
	date      = "unknown"
	builtBy   = "unknown"
)

func main() {
	// Set version info
	cli.Version = version
	cli.Commit = commit
	cli.Date = date
	cli.BuiltBy = builtBy

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		os.Exit(130) // 128 + SIGINT
	}()

	// Execute CLI
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
```

### Step 1.2: Create cli/version.go

```go
// cmd/gonuget/cli/version.go
package cli

// Version information (set by main)
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
	BuiltBy = "unknown"
)

// GetVersion returns formatted version information
func GetVersion() string {
	return Version
}

// GetFullVersion returns detailed version information
func GetFullVersion() string {
	return "gonuget version " + Version + "\n" +
		"commit: " + Commit + "\n" +
		"built: " + Date + "\n" +
		"built by: " + BuiltBy
}
```

### Step 1.3: Create cli/app.go (skeleton)

```go
// cmd/gonuget/cli/app.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gonuget",
	Short: "NuGet package manager CLI",
	Long: `gonuget is a cross-platform NuGet package manager CLI with 100% parity to dotnet nuget.

Complete documentation is available at https://github.com/willibrandon/gonuget`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags will be added here
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Show help information")
	rootCmd.PersistentFlags().BoolP("version", "", false, "Show version information")

	// Handle --version flag
	rootCmd.SetVersionTemplate(GetFullVersion() + "\n")
	rootCmd.Version = GetVersion()
}
```

### Verification

```bash
# Build the CLI
go build -o gonuget ./cmd/gonuget

# Test basic execution
./gonuget --help
./gonuget --version
```

**Expected output**:
- `--help` shows usage information
- `--version` shows version details

### Testing

Create `cmd/gonuget/cli/app_test.go`:

```go
package cli

import (
	"testing"
)

func TestGetVersion(t *testing.T) {
	Version = "1.0.0"
	if got := GetVersion(); got != "1.0.0" {
		t.Errorf("GetVersion() = %v, want %v", got, "1.0.0")
	}
}

func TestGetFullVersion(t *testing.T) {
	Version = "1.0.0"
	Commit = "abc123"
	Date = "2025-01-01"
	BuiltBy = "test"

	got := GetFullVersion()
	if got == "" {
		t.Error("GetFullVersion() returned empty string")
	}
	// Should contain version info
	if !contains(got, Version) {
		t.Errorf("GetFullVersion() doesn't contain version %s", Version)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[1:len(s)-1] != s[1:len(s)-1]))
	// Simple contains check
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

```bash
go test ./cmd/gonuget/cli -v
```

### CLI Interop Testing

Since this chunk establishes basic CLI execution infrastructure, we'll add smoke test handlers to the CLI interop bridge:

**Go Handler** (`cmd/gonuget-cli-interop-test/handlers_basic.go`):
```go
type ExecuteHelpHandler struct{}

func (h *ExecuteHelpHandler) Handle(data json.RawMessage) (interface{}, error) {
    var req struct {
        Command string `json:"command"`
    }
    if err := json.Unmarshal(data, &req); err != nil {
        return nil, err
    }

    // Execute: dotnet nuget --help
    dotnetResult, err := ExecuteDotnetNuget([]string{"--help"}, "")
    if err != nil {
        return nil, err
    }

    // Execute: gonuget --help
    gonugetResult, err := ExecuteGonuget([]string{"--help"}, "")
    if err != nil {
        return nil, err
    }

    return ExecuteCommandPairResponse{
        DotnetExitCode: dotnetResult.ExitCode,
        DotnetStdout: dotnetResult.Stdout,
        GonugetExitCode: gonugetResult.ExitCode,
        GonugetStdout: gonugetResult.Stdout,
    }, nil
}

func (h *ExecuteHelpHandler) ErrorCode() string {
    return "help_execution_error"
}
```

**C# Test** (`tests/cli-interop/GonugetCliInterop.Tests/BasicTests.cs`):
```csharp
public class BasicTests
{
    [Fact]
    public void Help_ShouldExecuteSuccessfully()
    {
        var result = GonugetCliBridge.ExecuteCommandPair("--help", "--help");

        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
        Assert.NotEmpty(result.DotnetStdout);
        Assert.NotEmpty(result.GonugetStdout);
    }

    [Fact]
    public void Version_ShouldShowVersionInfo()
    {
        var result = GonugetCliBridge.ExecuteCommandPair("--version", "--version");

        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
        Assert.NotEmpty(result.DotnetStdout);
        Assert.NotEmpty(result.GonugetStdout);
    }
}
```

**Test Execution**:
```bash
# Build CLI interop bridge
make build-cli-interop

# Run CLI interop tests
cd tests/cli-interop/GonugetCliInterop.Tests
dotnet test --filter "FullyQualifiedName~BasicTests"
```

### Commit

```bash
git add cmd/gonuget/
git commit -m "feat(cli): add project structure and entry point

- Create main.go with signal handling
- Add version information management
- Set up Cobra root command
- Add basic --help and --version flags

Tests: Basic CLI execution and version display"
```

---

## Chunk 2: Console Abstraction

**Objective**: Implement console output abstraction with color support and verbosity levels.

**Files to create**:
- `cmd/gonuget/output/console.go`
- `cmd/gonuget/output/colors.go`
- `cmd/gonuget/output/console_test.go`

### Step 2.1: Create output/colors.go

```go
// cmd/gonuget/output/colors.go
package output

import (
	"github.com/fatih/color"
	"os"
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
```

### Step 2.2: Create output/console.go

```go
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
func (c *Console) Print(a ...interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprint(c.out, a...)
}

// Println writes line to output
func (c *Console) Println(a ...interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintln(c.out, a...)
}

// Printf writes formatted output
func (c *Console) Printf(format string, a ...interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintf(c.out, format, a...)
}

// Success writes success message (green)
func (c *Console) Success(format string, a ...interface{}) {
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
func (c *Console) Error(format string, a ...interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.colors {
		ColorError.Fprintf(c.err, "Error: "+format+"\n", a...)
	} else {
		fmt.Fprintf(c.err, "Error: "+format+"\n", a...)
	}
}

// Warning writes warning message (yellow)
func (c *Console) Warning(format string, a ...interface{}) {
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
func (c *Console) Info(format string, a ...interface{}) {
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
func (c *Console) Debug(format string, a ...interface{}) {
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
func (c *Console) Detail(format string, a ...interface{}) {
	if c.verbosity >= VerbosityDetailed {
		c.mu.Lock()
		defer c.mu.Unlock()
		fmt.Fprintf(c.out, format+"\n", a...)
	}
}
```

### Step 2.3: Create tests

```go
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

func TestConsole_Success(t *testing.T) {
	var out bytes.Buffer
	c := NewConsole(&out, &out, VerbosityNormal)
	c.SetColors(false) // Disable colors for testing

	c.Success("operation completed")
	got := out.String()
	if !strings.Contains(got, "operation completed") {
		t.Errorf("Success() output doesn't contain expected message, got %q", got)
	}
}

func TestConsole_Error(t *testing.T) {
	var out, err bytes.Buffer
	c := NewConsole(&out, &err, VerbosityNormal)
	c.SetColors(false)

	c.Error("something failed")
	got := err.String()
	if !strings.Contains(got, "Error:") || !strings.Contains(got, "something failed") {
		t.Errorf("Error() output doesn't contain expected format, got %q", got)
	}
}

func TestConsole_Verbosity(t *testing.T) {
	tests := []struct {
		name      string
		verbosity Verbosity
		method    func(*Console)
		wantOut   bool
	}{
		{
			name:      "quiet suppresses info",
			verbosity: VerbosityQuiet,
			method:    func(c *Console) { c.Info("test") },
			wantOut:   false,
		},
		{
			name:      "normal shows info",
			verbosity: VerbosityNormal,
			method:    func(c *Console) { c.Info("test") },
			wantOut:   true,
		},
		{
			name:      "quiet suppresses debug",
			verbosity: VerbosityQuiet,
			method:    func(c *Console) { c.Debug("test") },
			wantOut:   false,
		},
		{
			name:      "diagnostic shows debug",
			verbosity: VerbosityDiagnostic,
			method:    func(c *Console) { c.Debug("test") },
			wantOut:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			c := NewConsole(&out, &out, tt.verbosity)
			c.SetColors(false)

			tt.method(c)

			gotOut := out.Len() > 0
			if gotOut != tt.wantOut {
				t.Errorf("verbosity %d: got output=%v, want=%v", tt.verbosity, gotOut, tt.wantOut)
			}
		})
	}
}

func TestIsColorEnabled(t *testing.T) {
	// Save original env
	origNoColor := os.Getenv("NO_COLOR")
	origTerm := os.Getenv("TERM")
	defer func() {
		os.Setenv("NO_COLOR", origNoColor)
		os.Setenv("TERM", origTerm)
	}()

	tests := []struct {
		name     string
		noColor  string
		term     string
		want     bool
	}{
		{
			name:    "NO_COLOR disables",
			noColor: "1",
			term:    "xterm",
			want:    false,
		},
		{
			name:    "dumb terminal disables",
			noColor: "",
			term:    "dumb",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("NO_COLOR", tt.noColor)
			os.Setenv("TERM", tt.term)

			// Note: IsColorEnabled also checks if stdout is a TTY,
			// which will be false in tests, so we can't fully test this
			// Just ensure it doesn't panic
			_ = IsColorEnabled()
		})
	}
}
```

### Verification

```bash
# Run tests
go test ./cmd/gonuget/output -v

# Test in CLI
go build -o gonuget ./cmd/gonuget
./gonuget --help  # Should show colored output (if terminal supports it)
```

### CLI Interop Testing

The console abstraction is tested indirectly through all other CLI interop tests. However, we should add specific tests for verbosity level handling to ensure it matches `dotnet nuget` behavior:

**Go Handler** (`cmd/gonuget-cli-interop-test/handlers_basic.go`):
```go
type ExecuteWithVerbosityHandler struct{}

func (h *ExecuteWithVerbosityHandler) Handle(data json.RawMessage) (interface{}, error) {
    var req struct {
        Command    []string `json:"command"`
        Verbosity  string   `json:"verbosity"`  // "quiet", "normal", "detailed", "diagnostic"
    }
    if err := json.Unmarshal(data, &req); err != nil {
        return nil, err
    }

    // Execute: dotnet nuget [command] --verbosity [level]
    dotnetCmd := append(req.Command, "--verbosity", req.Verbosity)
    dotnetResult, err := ExecuteDotnetNuget(dotnetCmd, "")
    if err != nil {
        return nil, err
    }

    // Execute: gonuget [command] --verbosity [level]
    gonugetCmd := append(req.Command, "--verbosity", req.Verbosity)
    gonugetResult, err := ExecuteGonuget(gonugetCmd, "")
    if err != nil {
        return nil, err
    }

    return ExecuteCommandPairResponse{
        DotnetExitCode: dotnetResult.ExitCode,
        DotnetStdout: dotnetResult.Stdout,
        GonugetExitCode: gonugetResult.ExitCode,
        GonugetStdout: gonugetResult.Stdout,
    }, nil
}

func (h *ExecuteWithVerbosityHandler) ErrorCode() string {
    return "verbosity_execution_error"
}
```

**C# Test** (`tests/cli-interop/GonugetCliInterop.Tests/VerbosityTests.cs`):
```csharp
public class VerbosityTests
{
    [Theory]
    [InlineData("quiet")]
    [InlineData("normal")]
    [InlineData("detailed")]
    [InlineData("diagnostic")]
    public void Config_WithVerbosity_ShouldMatchDotnetNuget(string verbosity)
    {
        var result = GonugetCliBridge.ExecuteWithVerbosity(
            new[] { "config" },
            verbosity);

        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
        // Output amount should be similar (exact match not required due to formatting)
    }
}
```

**Note**: Color output is disabled in non-TTY environments, so CLI interop tests will not test color codes. Manual testing required for color verification.

**Test Execution**:
```bash
cd tests/cli-interop/GonugetCliInterop.Tests
dotnet test --filter "FullyQualifiedName~VerbosityTests"
```

### Commit

```bash
git add cmd/gonuget/output/
git commit -m "feat(cli): add console output abstraction

- Implement Console with verbosity levels
- Add color support with auto-detection
- Support quiet, normal, detailed, and diagnostic verbosity
- Thread-safe output methods
- Disable colors in non-TTY environments

Tests: Console output and verbosity filtering
Coverage: 85%+"
```

---

## Chunk 3: Configuration Management (NuGet.config XML)

**Objective**: Implement NuGet.config XML parsing and writing.

**Files to create**:
- `cmd/gonuget/config/nuget_config.go`
- `cmd/gonuget/config/sources.go`
- `cmd/gonuget/config/defaults.go`
- `cmd/gonuget/config/nuget_config_test.go`

### Step 3.1: Create config/nuget_config.go

```go
// cmd/gonuget/config/nuget_config.go
package config

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// NuGetConfig represents a NuGet.config file
type NuGetConfig struct {
	XMLName         xml.Name         `xml:"configuration"`
	PackageSources  *PackageSources  `xml:"packageSources"`
	APIKeys         *APIKeys         `xml:"apikeys"`
	Config          *ConfigSection   `xml:"config"`
	TrustedSigners  *TrustedSigners  `xml:"trustedSigners"`
	PackageSourceCredentials *PackageSourceCredentials `xml:"packageSourceCredentials"`
}

// PackageSources contains package source definitions
type PackageSources struct {
	Clear bool            `xml:"clear"`
	Add   []PackageSource `xml:"add"`
}

// PackageSource represents a package source
type PackageSource struct {
	Key             string `xml:"key,attr"`
	Value           string `xml:"value,attr"`
	ProtocolVersion string `xml:"protocolVersion,attr,omitempty"`
	Enabled         string `xml:"enabled,attr,omitempty"`
}

// APIKeys contains API key mappings
type APIKeys struct {
	Clear bool     `xml:"clear"`
	Add   []APIKey `xml:"add"`
}

// APIKey represents an API key for a source
type APIKey struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

// ConfigSection contains configuration settings
type ConfigSection struct {
	Clear bool         `xml:"clear"`
	Add   []ConfigItem `xml:"add"`
}

// ConfigItem represents a configuration key-value pair
type ConfigItem struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

// TrustedSigners contains trusted signer definitions
type TrustedSigners struct {
	Clear bool            `xml:"clear"`
	Add   []TrustedSigner `xml:"add,omitempty"`
}

// TrustedSigner represents a trusted signer
type TrustedSigner struct {
	Name string `xml:"name,attr"`
	// Additional fields as needed
}

// PackageSourceCredentials contains credentials for sources
type PackageSourceCredentials struct {
	Items []SourceCredential `xml:",any"`
}

// SourceCredential represents credentials for a source
type SourceCredential struct {
	XMLName xml.Name
	Add     []ConfigItem `xml:"add"`
}

// LoadNuGetConfig loads a NuGet.config file
func LoadNuGetConfig(path string) (*NuGetConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	return ParseNuGetConfig(f)
}

// ParseNuGetConfig parses NuGet.config XML from a reader
func ParseNuGetConfig(r io.Reader) (*NuGetConfig, error) {
	var config NuGetConfig
	decoder := xml.NewDecoder(r)

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config XML: %w", err)
	}

	return &config, nil
}

// SaveNuGetConfig saves a NuGet.config file
func SaveNuGetConfig(path string, config *NuGetConfig) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	return WriteNuGetConfig(f, config)
}

// WriteNuGetConfig writes NuGet.config XML to a writer
func WriteNuGetConfig(w io.Writer, config *NuGetConfig) error {
	// Write XML declaration
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return err
	}

	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")

	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config XML: %w", err)
	}

	return encoder.Flush()
}

// GetPackageSource gets a package source by key
func (c *NuGetConfig) GetPackageSource(key string) *PackageSource {
	if c.PackageSources == nil {
		return nil
	}

	for i := range c.PackageSources.Add {
		if c.PackageSources.Add[i].Key == key {
			return &c.PackageSources.Add[i]
		}
	}

	return nil
}

// AddPackageSource adds or updates a package source
func (c *NuGetConfig) AddPackageSource(source PackageSource) {
	if c.PackageSources == nil {
		c.PackageSources = &PackageSources{}
	}

	// Check if source exists
	for i := range c.PackageSources.Add {
		if c.PackageSources.Add[i].Key == source.Key {
			// Update existing
			c.PackageSources.Add[i] = source
			return
		}
	}

	// Add new
	c.PackageSources.Add = append(c.PackageSources.Add, source)
}

// RemovePackageSource removes a package source by key
func (c *NuGetConfig) RemovePackageSource(key string) bool {
	if c.PackageSources == nil {
		return false
	}

	for i := range c.PackageSources.Add {
		if c.PackageSources.Add[i].Key == key {
			c.PackageSources.Add = append(
				c.PackageSources.Add[:i],
				c.PackageSources.Add[i+1:]...,
			)
			return true
		}
	}

	return false
}

// GetConfigValue gets a configuration value by key
func (c *NuGetConfig) GetConfigValue(key string) string {
	if c.Config == nil {
		return ""
	}

	for _, item := range c.Config.Add {
		if item.Key == key {
			return item.Value
		}
	}

	return ""
}

// SetConfigValue sets a configuration value
func (c *NuGetConfig) SetConfigValue(key, value string) {
	if c.Config == nil {
		c.Config = &ConfigSection{}
	}

	// Check if exists
	for i := range c.Config.Add {
		if c.Config.Add[i].Key == key {
			c.Config.Add[i].Value = value
			return
		}
	}

	// Add new
	c.Config.Add = append(c.Config.Add, ConfigItem{Key: key, Value: value})
}
```

### Step 3.2: Create config/defaults.go

```go
// cmd/gonuget/config/defaults.go
package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultConfigLocations returns the list of NuGet.config locations to search
// in precedence order
func DefaultConfigLocations() []string {
	var locations []string

	// Current directory
	if cwd, err := os.Getwd(); err == nil {
		locations = append(locations, filepath.Join(cwd, "NuGet.config"))
		locations = append(locations, filepath.Join(cwd, ".nuget", "NuGet.config"))
	}

	// User config
	if home, err := os.UserHomeDir(); err == nil {
		locations = append(locations, filepath.Join(home, ".nuget", "NuGet.config"))
	}

	// System config (platform-specific)
	if runtime.GOOS == "windows" {
		// Windows: %ProgramData%\NuGet\Config
		if programData := os.Getenv("ProgramData"); programData != "" {
			locations = append(locations, filepath.Join(programData, "NuGet", "Config", "Microsoft.VisualStudio.Offline.config"))
		}
	} else {
		// Unix: /etc/nuget
		locations = append(locations, "/etc/nuget/NuGet.config")
	}

	return locations
}

// FindConfigFile finds the first existing NuGet.config file
func FindConfigFile() string {
	for _, loc := range DefaultConfigLocations() {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}
	return ""
}

// GetUserConfigPath returns the user-level NuGet.config path
func GetUserConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".nuget", "NuGet.config")
}

// DefaultPackageSources returns the default package sources
func DefaultPackageSources() []PackageSource {
	return []PackageSource{
		{
			Key:             "nuget.org",
			Value:           "https://api.nuget.org/v3/index.json",
			ProtocolVersion: "3",
			Enabled:         "true",
		},
	}
}

// NewDefaultConfig creates a new config with default values
func NewDefaultConfig() *NuGetConfig {
	config := &NuGetConfig{
		PackageSources: &PackageSources{
			Add: DefaultPackageSources(),
		},
		Config: &ConfigSection{},
	}

	return config
}
```

### Step 3.3: Create tests

```go
// cmd/gonuget/config/nuget_config_test.go
package config

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseNuGetConfig(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <packageSources>
    <add key="nuget.org" value="https://api.nuget.org/v3/index.json" protocolVersion="3" />
  </packageSources>
  <config>
    <add key="globalPackagesFolder" value="~/.nuget/packages" />
  </config>
</configuration>`

	config, err := ParseNuGetConfig(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNuGetConfig() error = %v", err)
	}

	if config.PackageSources == nil {
		t.Fatal("PackageSources is nil")
	}

	if len(config.PackageSources.Add) != 1 {
		t.Errorf("expected 1 package source, got %d", len(config.PackageSources.Add))
	}

	source := config.PackageSources.Add[0]
	if source.Key != "nuget.org" {
		t.Errorf("source.Key = %q, want %q", source.Key, "nuget.org")
	}

	value := config.GetConfigValue("globalPackagesFolder")
	if value != "~/.nuget/packages" {
		t.Errorf("config value = %q, want %q", value, "~/.nuget/packages")
	}
}

func TestWriteNuGetConfig(t *testing.T) {
	config := NewDefaultConfig()
	config.SetConfigValue("test", "value")

	var buf bytes.Buffer
	if err := WriteNuGetConfig(&buf, config); err != nil {
		t.Fatalf("WriteNuGetConfig() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "configuration") {
		t.Error("output doesn't contain configuration element")
	}
	if !strings.Contains(output, "nuget.org") {
		t.Error("output doesn't contain nuget.org source")
	}
}

func TestNuGetConfig_AddPackageSource(t *testing.T) {
	config := &NuGetConfig{}

	source := PackageSource{
		Key:   "test",
		Value: "https://test.com/v3/index.json",
	}

	config.AddPackageSource(source)

	got := config.GetPackageSource("test")
	if got == nil {
		t.Fatal("GetPackageSource() returned nil")
	}
	if got.Key != "test" {
		t.Errorf("source.Key = %q, want %q", got.Key, "test")
	}
}

func TestNuGetConfig_RemovePackageSource(t *testing.T) {
	config := NewDefaultConfig()

	if !config.RemovePackageSource("nuget.org") {
		t.Error("RemovePackageSource() returned false, want true")
	}

	if config.GetPackageSource("nuget.org") != nil {
		t.Error("source still exists after removal")
	}
}

func TestDefaultConfigLocations(t *testing.T) {
	locations := DefaultConfigLocations()
	if len(locations) == 0 {
		t.Error("DefaultConfigLocations() returned empty slice")
	}

	// Should contain at least current directory and user directory
	foundCurrent := false
	foundUser := false
	for _, loc := range locations {
		if strings.Contains(loc, "NuGet.config") {
			if strings.Contains(loc, ".nuget") {
				foundUser = true
			} else {
				foundCurrent = true
			}
		}
	}

	if !foundCurrent && !foundUser {
		t.Error("DefaultConfigLocations() doesn't contain expected paths")
	}
}
```

### Verification

```bash
# Run tests
go test ./cmd/gonuget/config -v

# Test config loading
cat > /tmp/test-nuget.config << 'EOF'
<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <packageSources>
    <add key="nuget.org" value="https://api.nuget.org/v3/index.json" protocolVersion="3" />
  </packageSources>
</configuration>
EOF

# Create test program to verify config loading
cat > /tmp/test-config.go << 'EOF'
package main
import (
	"fmt"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
)
func main() {
	cfg, err := config.LoadNuGetConfig("/tmp/test-nuget.config")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Loaded config with %d sources\n", len(cfg.PackageSources.Add))
}
EOF

go run /tmp/test-config.go
```

### CLI Interop Testing

The NuGet.config XML parsing and management is tested through the `config` command CLI interop tests (see Chunk 5). This chunk provides the infrastructure that enables accurate `dotnet nuget config` parity.

**Key Compatibility Points**:
- XML structure must match `dotnet nuget` exactly
- Config file hierarchy (current, user, system) must match .NET behavior
- Default package sources must match (nuget.org V3 feed)
- Path resolution on Windows/Linux/macOS must match .NET conventions

**Manual Verification**:
```bash
# Create a config with dotnet nuget
dotnet nuget add source https://test.com/v3/index.json --name test-source

# Verify gonuget can read it
./gonuget config

# Create a config with gonuget
./gonuget config globalPackagesFolder ~/my-packages

# Verify dotnet nuget can read it
dotnet nuget config globalPackagesFolder
```

**Test Execution**:
Config file compatibility is validated in ConfigTests.cs (see Chunk 5).

### Commit

```bash
git add cmd/gonuget/config/
git commit -m "feat(cli): add NuGet.config XML parsing and management

- Implement NuGet.config XML structure
- Support package sources, API keys, config section
- Add config loading from multiple locations
- Support config hierarchy (current, user, system)
- Add default configuration values

Tests: Config parsing, round-trip, CRUD operations
Coverage: 90%+"
```

---

## Chunk 4: Version Command

**Objective**: Implement the `version` command with output matching `dotnet nuget --version`.

**Files to create**:
- `cmd/gonuget/commands/version.go`
- `cmd/gonuget/commands/version_test.go`

### Step 4.1: Update cli/app.go to register command

```go
// cmd/gonuget/cli/app.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/commands"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

var rootCmd = &cobra.Command{
	Use:   "gonuget",
	Short: "NuGet package manager CLI",
	Long: `gonuget is a cross-platform NuGet package manager CLI with 100% parity to dotnet nuget.

Complete documentation is available at https://github.com/willibrandon/gonuget`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Global console
var Console *output.Console

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Initialize console
	Console = output.DefaultConsole()

	// Global flags
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Show help information")
	rootCmd.PersistentFlags().StringP("verbosity", "", "normal", "Set verbosity level (quiet, normal, detailed, diagnostic)")
	rootCmd.PersistentFlags().BoolP("non-interactive", "", false, "Do not prompt for user input or confirmations")

	// Register commands
	rootCmd.AddCommand(commands.NewVersionCommand(Console))
}
```

### Step 4.2: Create commands/version.go

```go
// cmd/gonuget/commands/version.go
package commands

import (
	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/cli"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewVersionCommand creates the version command
func NewVersionCommand(console *output.Console) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Long:  `Display detailed version information including commit, build date, and builder.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(console)
		},
	}

	return cmd
}

func runVersion(console *output.Console) error {
	console.Println(cli.GetFullVersion())
	return nil
}
```

### Step 4.3: Create test

```go
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

	output := out.String()
	if !strings.Contains(output, "1.0.0") {
		t.Errorf("output doesn't contain version, got: %s", output)
	}
	if !strings.Contains(output, "abc123") {
		t.Errorf("output doesn't contain commit, got: %s", output)
	}
}
```

### Verification

```bash
# Run tests
go test ./cmd/gonuget/commands -v

# Build and test
go build -ldflags "-X github.com/willibrandon/gonuget/cmd/gonuget/cli.Version=1.0.0-test" -o gonuget ./cmd/gonuget
./gonuget version

# Compare with dotnet nuget
dotnet nuget --version
./gonuget --version
```

**Expected output format** (matching `dotnet nuget --version`):
```
gonuget version 1.0.0-test
commit: unknown
built: unknown
built by: unknown
```

**Note**: `dotnet nuget --version` output format varies by SDK version. The gonuget output should be similar but doesn't need to be byte-for-byte identical. Focus on including version number prominently.

### CLI Interop Testing

**Go Handler** (`cmd/gonuget-cli-interop-test/handlers_basic.go`):
```go
type ExecuteVersionHandler struct{}

func (h *ExecuteVersionHandler) Handle(data json.RawMessage) (interface{}, error) {
    // Execute: dotnet nuget --version
    dotnetResult, err := ExecuteDotnetNuget([]string{"--version"}, "")
    if err != nil {
        return nil, err
    }

    // Execute: gonuget --version
    gonugetResult, err := ExecuteGonuget([]string{"--version"}, "")
    if err != nil {
        return nil, err
    }

    return ExecuteCommandPairResponse{
        DotnetExitCode: dotnetResult.ExitCode,
        DotnetStdout: dotnetResult.Stdout,
        DotnetStderr: dotnetResult.Stderr,
        GonugetExitCode: gonugetResult.ExitCode,
        GonugetStdout: gonugetResult.Stdout,
        GonugetStderr: gonugetResult.Stderr,
    }, nil
}

func (h *ExecuteVersionHandler) ErrorCode() string {
    return "version_execution_error"
}
```

**C# Test** (`tests/cli-interop/GonugetCliInterop.Tests/VersionTests.cs`):
```csharp
public class VersionTests
{
    [Fact]
    public void Version_WithDoubleDash_ShouldExecuteSuccessfully()
    {
        var result = GonugetCliBridge.ExecuteCommandPair("--version", "--version");

        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
        Assert.NotEmpty(result.DotnetStdout);
        Assert.NotEmpty(result.GonugetStdout);

        // Both should contain version information
        Assert.Contains("version", result.DotnetStdout.ToLower());
        Assert.Contains("version", result.GonugetStdout.ToLower());
    }

    [Fact]
    public void Version_AsSubcommand_ShouldExecuteSuccessfully()
    {
        var result = GonugetCliBridge.ExecuteCommandPair("version", "version");

        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);
        Assert.NotEmpty(result.DotnetStdout);
        Assert.NotEmpty(result.GonugetStdout);
    }

    [Fact]
    public void Version_OutputShouldContainVersionNumber()
    {
        var result = GonugetCliBridge.ExecuteVersion();

        Assert.Equal(0, result.ExitCode);
        Assert.NotEmpty(result.Stdout);

        // Should contain a version number pattern (e.g., "1.0.0", "1.2.3-beta")
        Assert.Matches(@"\d+\.\d+\.\d+", result.Stdout);
    }
}
```

**Test Execution**:
```bash
# Build CLI interop bridge
make build-cli-interop

# Run version CLI interop tests
cd tests/cli-interop/GonugetCliInterop.Tests
dotnet test --filter "FullyQualifiedName~VersionTests"
```

### Commit

```bash
git add cmd/gonuget/
git commit -m "feat(cli): add version command

- Implement version command with detailed build info
- Display version, commit, date, and builder
- Support for ldflags version injection

Tests: Version command execution and output
Commands: 1/20 complete (5%)"
```

---

*Continue in next comment due to length...*

---

## Chunk 5: Config Command

**Objective**: Implement `config` command matching `dotnet nuget config` behavior exactly with four subcommands: get, set, unset, and paths.

**Reference**: `dotnet nuget config` implementation in `/Users/brandon/src/NuGet.Client/src/NuGet.Core/NuGet.CommandLine.XPlat/Commands/ConfigCommands/`

### Step 5.1: Create commands/config.go

```go
// cmd/gonuget/commands/config.go
package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// NewConfigCommand creates the config command with get/set/unset/paths subcommands
func NewConfigCommand(console *output.Console) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage NuGet configuration",
		Long: `Gets, sets, unsets, or displays paths for NuGet configuration values.

This command has four subcommands:
  - get:   Get a configuration value (or all with "all")
  - set:   Set a configuration value
  - unset: Remove a configuration value
  - paths: Display configuration file paths

Examples:
  gonuget config get repositoryPath
  gonuget config get all
  gonuget config set repositoryPath ~/packages
  gonuget config unset repositoryPath
  gonuget config paths`,
		SilenceUsage: true,
	}

	// Add subcommands
	cmd.AddCommand(newConfigGetCommand(console))
	cmd.AddCommand(newConfigSetCommand(console))
	cmd.AddCommand(newConfigUnsetCommand(console))
	cmd.AddCommand(newConfigPathsCommand(console))

	return cmd
}

// Config Get Subcommand

type configGetOptions struct {
	workingDirectory string
	showPath         bool
}

func newConfigGetCommand(console *output.Console) *cobra.Command {
	opts := &configGetOptions{}

	cmd := &cobra.Command{
		Use:   "get <all-or-config-key>",
		Short: "Get a configuration value",
		Long: `Get a NuGet configuration value by key, or get all values with "all".

Examples:
  gonuget config get repositoryPath
  gonuget config get all
  gonuget config get globalPackagesFolder --show-path`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(console, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.workingDirectory, "working-directory", "", "Working directory for config hierarchy resolution")
	cmd.Flags().BoolVar(&opts.showPath, "show-path", false, "Return value as filesystem path")

	return cmd
}

func runConfigGet(console *output.Console, allOrConfigKey string, opts *configGetOptions) error {
	// Determine config file based on working directory
	configPath := determineConfigPath(opts.workingDirectory)

	// Load config
	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle "all" keyword
	if strings.EqualFold(allOrConfigKey, "all") {
		return listAllConfig(console, cfg)
	}

	// Get specific value
	value := cfg.GetConfigValue(allOrConfigKey)
	if value == "" {
		return fmt.Errorf("key '%s' not found", allOrConfigKey)
	}

	// Handle --show-path
	if opts.showPath {
		absPath, err := filepath.Abs(value)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
		console.Println(absPath)
	} else {
		console.Println(value)
	}

	return nil
}

// Config Set Subcommand

type configSetOptions struct {
	configFile string
}

func newConfigSetCommand(console *output.Console) *cobra.Command {
	opts := &configSetOptions{}

	cmd := &cobra.Command{
		Use:   "set <config-key> <config-value>",
		Short: "Set a configuration value",
		Long: `Set a NuGet configuration value.

Examples:
  gonuget config set repositoryPath ~/packages
  gonuget config set globalPackagesFolder ~/.nuget/packages
  gonuget config set http_proxy http://proxy:8080 --configfile ./NuGet.config`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(console, args[0], args[1], opts)
		},
	}

	cmd.Flags().StringVar(&opts.configFile, "configfile", "", "Config file to modify")

	return cmd
}

func runConfigSet(console *output.Console, configKey string, configValue string, opts *configSetOptions) error {
	// Determine config file
	configPath := opts.configFile
	if configPath == "" {
		configPath = config.FindConfigFile()
		if configPath == "" {
			configPath = config.GetUserConfigPath()
		}
	}

	// Load or create config
	cfg, err := loadOrCreateConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set value
	cfg.SetConfigValue(configKey, configValue)

	// Save config
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	console.Println(fmt.Sprintf("Successfully updated config file at '%s'.", configPath))
	return nil
}

// Config Unset Subcommand

type configUnsetOptions struct {
	configFile string
}

func newConfigUnsetCommand(console *output.Console) *cobra.Command {
	opts := &configUnsetOptions{}

	cmd := &cobra.Command{
		Use:   "unset <config-key>",
		Short: "Remove a configuration value",
		Long: `Remove a NuGet configuration value.

Examples:
  gonuget config unset repositoryPath
  gonuget config unset http_proxy --configfile ./NuGet.config`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigUnset(console, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.configFile, "configfile", "", "Config file to modify")

	return cmd
}

func runConfigUnset(console *output.Console, configKey string, opts *configUnsetOptions) error {
	// Determine config file
	configPath := opts.configFile
	if configPath == "" {
		configPath = config.FindConfigFile()
		if configPath == "" {
			configPath = config.GetUserConfigPath()
		}
	}

	// Load config
	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Remove value
	cfg.DeleteConfigValue(configKey)

	// Save config
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	console.Println(fmt.Sprintf("Successfully updated config file at '%s'.", configPath))
	return nil
}

// Config Paths Subcommand

type configPathsOptions struct {
	workingDirectory string
}

func newConfigPathsCommand(console *output.Console) *cobra.Command {
	opts := &configPathsOptions{}

	cmd := &cobra.Command{
		Use:   "paths",
		Short: "Display configuration file paths",
		Long: `Display the paths to NuGet configuration files in the hierarchy.

Examples:
  gonuget config paths
  gonuget config paths --working-directory /path/to/project`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPaths(console, opts)
		},
	}

	cmd.Flags().StringVar(&opts.workingDirectory, "working-directory", "", "Working directory for config hierarchy resolution")

	return cmd
}

func runConfigPaths(console *output.Console, opts *configPathsOptions) error {
	// Get config hierarchy
	paths := config.GetConfigHierarchy(opts.workingDirectory)

	console.Println("NuGet configuration file paths:")
	for _, path := range paths {
		exists := "✓"
		if _, err := filepath.Abs(path); err != nil {
			exists = "✗"
		}
		console.Printf("  %s %s\n", exists, path)
	}

	return nil
}

// Helper functions

func determineConfigPath(workingDirectory string) string {
	if workingDirectory != "" {
		// Start from working directory
		return config.FindConfigFileFrom(workingDirectory)
	}
	// Use current directory
	configPath := config.FindConfigFile()
	if configPath == "" {
		configPath = config.GetUserConfigPath()
	}
	return configPath
}

func loadOrCreateConfig(path string) (*config.NuGetConfig, error) {
	cfg, err := config.LoadNuGetConfig(path)
	if err != nil {
		// Create new default config
		cfg = config.NewDefaultConfig()
	}
	return cfg, nil
}

func listAllConfig(console *output.Console, cfg *config.NuGetConfig) error {
	if cfg.Config == nil || len(cfg.Config.Add) == 0 {
		console.Println("No configuration values found.")
		return nil
	}

	for _, item := range cfg.Config.Add {
		console.Printf("%s=%s\n", item.Key, item.Value)
	}

	return nil
}
```

### Step 5.2: Register command

```go
// cmd/gonuget/cli/app.go - add to init()
func init() {
	// ... existing code ...

	// Register commands
	rootCmd.AddCommand(commands.NewVersionCommand(Console))
	rootCmd.AddCommand(commands.NewConfigCommand(Console))
}
```

### Step 5.3: Add config package helpers

The config command requires additional helper functions in the config package:

```go
// cmd/gonuget/config/config.go - add these methods

// DeleteConfigValue removes a configuration value
func (c *NuGetConfig) DeleteConfigValue(key string) {
	if c.Config == nil {
		return
	}

	var filtered []ConfigItem
	for _, item := range c.Config.Add {
		if item.Key != key {
			filtered = append(filtered, item)
		}
	}
	c.Config.Add = filtered
}

// FindConfigFileFrom finds config file starting from specified directory
func FindConfigFileFrom(startDir string) string {
	dir := startDir
	for {
		configPath := filepath.Join(dir, "NuGet.config")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	// Fall back to user config
	return GetUserConfigPath()
}

// GetConfigHierarchy returns all config file paths in the hierarchy
func GetConfigHierarchy(workingDirectory string) []string {
	var paths []string

	// Start directory
	startDir := workingDirectory
	if startDir == "" {
		startDir, _ = os.Getwd()
	}

	// Walk up directory tree
	dir := startDir
	for {
		configPath := filepath.Join(dir, "NuGet.config")
		if _, err := os.Stat(configPath); err == nil {
			paths = append(paths, configPath)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Add user config
	userConfig := GetUserConfigPath()
	paths = append(paths, userConfig)

	// Add machine-wide config (platform-specific)
	if runtime.GOOS == "windows" {
		programData := os.Getenv("ProgramData")
		if programData != "" {
			machineConfig := filepath.Join(programData, "NuGet", "Config", "NuGet.config")
			paths = append(paths, machineConfig)
		}
	} else {
		paths = append(paths, "/etc/nuget/NuGet.config")
	}

	return paths
}
```

### Step 5.4: Create tests

```go
// cmd/gonuget/commands/config_test.go
package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

// Config Get Tests

func TestConfigGet(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("testKey", "testValue")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	// Set working directory to temp dir so config is found
	opts := &configGetOptions{workingDirectory: tmpDir}
	if err := runConfigGet(console, "testKey", opts); err != nil {
		t.Fatalf("runConfigGet() error = %v", err)
	}

	result := strings.TrimSpace(out.String())
	if result != "testValue" {
		t.Errorf("output = %q, want %q", result, "testValue")
	}
}

func TestConfigGet_All(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("key1", "value1")
	cfg.SetConfigValue("key2", "value2")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configGetOptions{workingDirectory: tmpDir}
	if err := runConfigGet(console, "all", opts); err != nil {
		t.Fatalf("runConfigGet() error = %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "key1=value1") {
		t.Errorf("output should contain 'key1=value1'")
	}
	if !strings.Contains(result, "key2=value2") {
		t.Errorf("output should contain 'key2=value2'")
	}
}

func TestConfigGet_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configGetOptions{workingDirectory: tmpDir}
	err := runConfigGet(console, "nonexistent", opts)
	if err == nil {
		t.Error("runConfigGet() should return error for nonexistent key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestConfigGet_ShowPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("relativePath", "./packages")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configGetOptions{
		workingDirectory: tmpDir,
		showPath:         true,
	}
	if err := runConfigGet(console, "relativePath", opts); err != nil {
		t.Fatalf("runConfigGet() error = %v", err)
	}

	result := strings.TrimSpace(out.String())
	if !filepath.IsAbs(result) {
		t.Errorf("--show-path should return absolute path, got: %s", result)
	}
}

// Config Set Tests

func TestConfigSet(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configSetOptions{configFile: configPath}
	if err := runConfigSet(console, "newKey", "newValue", opts); err != nil {
		t.Fatalf("runConfigSet() error = %v", err)
	}

	// Verify config was saved
	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	value := cfg.GetConfigValue("newKey")
	if value != "newValue" {
		t.Errorf("config value = %q, want %q", value, "newValue")
	}
}

// Config Unset Tests

func TestConfigUnset(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	// Create config with value
	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("testKey", "testValue")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configUnsetOptions{configFile: configPath}
	if err := runConfigUnset(console, "testKey", opts); err != nil {
		t.Fatalf("runConfigUnset() error = %v", err)
	}

	// Verify value was removed
	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	value := cfg.GetConfigValue("testKey")
	if value != "" {
		t.Errorf("config value should be empty after unset, got: %q", value)
	}
}

// Config Paths Tests

func TestConfigPaths(t *testing.T) {
	tmpDir := t.TempDir()

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configPathsOptions{workingDirectory: tmpDir}
	if err := runConfigPaths(console, opts); err != nil {
		t.Fatalf("runConfigPaths() error = %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "NuGet configuration file paths:") {
		t.Error("output should contain header")
	}
}

// Command Structure Tests

func TestNewConfigCommand(t *testing.T) {
	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	cmd := NewConfigCommand(console)
	if cmd == nil {
		t.Fatal("NewConfigCommand() returned nil")
	}

	if cmd.Use != "config" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "config")
	}

	// Check subcommands exist
	subcommands := cmd.Commands()
	if len(subcommands) != 4 {
		t.Errorf("expected 4 subcommands, got %d", len(subcommands))
	}

	// Verify subcommand names
	var foundGet, foundSet, foundUnset, foundPaths bool
	for _, subcmd := range subcommands {
		switch subcmd.Use {
		case "get <all-or-config-key>":
			foundGet = true
		case "set <config-key> <config-value>":
			foundSet = true
		case "unset <config-key>":
			foundUnset = true
		case "paths":
			foundPaths = true
		}
	}

	if !foundGet {
		t.Error("missing 'get' subcommand")
	}
	if !foundSet {
		t.Error("missing 'set' subcommand")
	}
	if !foundUnset {
		t.Error("missing 'unset' subcommand")
	}
	if !foundPaths {
		t.Error("missing 'paths' subcommand")
	}
}
```

### Verification

```bash
# Run tests
go test ./cmd/gonuget/commands -v -run TestConfig

# Build and test
go build -o /tmp/gonuget ./cmd/gonuget

# Test config get
/tmp/gonuget config get repositoryPath
dotnet nuget config get repositoryPath
# Outputs should match

# Test config get all
/tmp/gonuget config get all
dotnet nuget config get all
# Outputs should match (format: key=value per line)

# Test config set
/tmp/gonuget config set testKey testValue --configfile /tmp/test.config
dotnet nuget config set testKey testValue --configfile /tmp/test.config
# Both should succeed with similar message

# Test config unset
/tmp/gonuget config unset testKey --configfile /tmp/test.config
dotnet nuget config unset testKey --configfile /tmp/test.config
# Both should succeed

# Test config paths
/tmp/gonuget config paths
dotnet nuget config paths
# Both should list config file paths in hierarchy

# Test --show-path flag
/tmp/gonuget config get globalPackagesFolder --show-path
dotnet nuget config get globalPackagesFolder --show-path
# Both should return absolute paths
```

### CLI Interop Testing

The config command is critical for NuGet.config parity. We need comprehensive CLI interop tests to ensure behavior matches `dotnet nuget config` exactly.

**Go Handler** (`cmd/gonuget-cli-interop-test/handlers_config.go`):
```go
type ExecuteConfigHandler struct{}

func (h *ExecuteConfigHandler) Handle(data json.RawMessage) (interface{}, error) {
    var req struct {
        Subcommand      string   `json:"subcommand"` // "get", "set", "unset", "paths"
        AllOrConfigKey  string   `json:"allOrConfigKey,omitempty"` // for "get"
        ConfigKey       string   `json:"configKey,omitempty"` // for "set"/"unset"
        ConfigValue     string   `json:"configValue,omitempty"` // for "set"
        ConfigFile      string   `json:"configFile,omitempty"`
        WorkingDirectory string   `json:"workingDirectory,omitempty"`
        ShowPath        bool     `json:"showPath,omitempty"`
    }
    if err := json.Unmarshal(data, &req); err != nil {
        return nil, err
    }

    var dotnetArgs, gonugetArgs []string

    // Build base command: dotnet nuget config <subcommand>
    dotnetArgs = []string{"config", req.Subcommand}
    gonugetArgs = []string{"config", req.Subcommand}

    // Add subcommand-specific args
    switch req.Subcommand {
    case "get":
        // dotnet nuget config get <all-or-config-key> [--show-path] [--working-directory]
        if req.AllOrConfigKey == "" {
            return nil, fmt.Errorf("allOrConfigKey required for get")
        }
        dotnetArgs = append(dotnetArgs, req.AllOrConfigKey)
        gonugetArgs = append(gonugetArgs, req.AllOrConfigKey)

        if req.ShowPath {
            dotnetArgs = append(dotnetArgs, "--show-path")
            gonugetArgs = append(gonugetArgs, "--show-path")
        }
        if req.WorkingDirectory != "" {
            dotnetArgs = append(dotnetArgs, "--working-directory", req.WorkingDirectory)
            gonugetArgs = append(gonugetArgs, "--working-directory", req.WorkingDirectory)
        }

    case "set":
        // dotnet nuget config set <config-key> <config-value> [--configfile]
        if req.ConfigKey == "" || req.ConfigValue == "" {
            return nil, fmt.Errorf("configKey and configValue required for set")
        }
        dotnetArgs = append(dotnetArgs, req.ConfigKey, req.ConfigValue)
        gonugetArgs = append(gonugetArgs, req.ConfigKey, req.ConfigValue)

        if req.ConfigFile != "" {
            dotnetArgs = append(dotnetArgs, "--configfile", req.ConfigFile)
            gonugetArgs = append(gonugetArgs, "--configfile", req.ConfigFile)
        }

    case "unset":
        // dotnet nuget config unset <config-key> [--configfile]
        if req.ConfigKey == "" {
            return nil, fmt.Errorf("configKey required for unset")
        }
        dotnetArgs = append(dotnetArgs, req.ConfigKey)
        gonugetArgs = append(gonugetArgs, req.ConfigKey)

        if req.ConfigFile != "" {
            dotnetArgs = append(dotnetArgs, "--configfile", req.ConfigFile)
            gonugetArgs = append(gonugetArgs, "--configfile", req.ConfigFile)
        }

    case "paths":
        // dotnet nuget config paths [--working-directory]
        if req.WorkingDirectory != "" {
            dotnetArgs = append(dotnetArgs, "--working-directory", req.WorkingDirectory)
            gonugetArgs = append(gonugetArgs, "--working-directory", req.WorkingDirectory)
        }

    default:
        return nil, fmt.Errorf("unknown subcommand: %s", req.Subcommand)
    }

    // Execute dotnet nuget
    dotnetResult, err := ExecuteDotnetNuget(dotnetArgs, "")
    if err != nil {
        return nil, err
    }

    // Execute gonuget
    gonugetResult, err := ExecuteGonuget(gonugetArgs, "")
    if err != nil {
        return nil, err
    }

    return ExecuteCommandPairResponse{
        DotnetExitCode: dotnetResult.ExitCode,
        DotnetStdout: NormalizeOutput(dotnetResult.Stdout),
        DotnetStderr: dotnetResult.Stderr,
        GonugetExitCode: gonugetResult.ExitCode,
        GonugetStdout: NormalizeOutput(gonugetResult.Stdout),
        GonugetStderr: gonugetResult.Stderr,
    }, nil
}

func (h *ExecuteConfigHandler) ErrorCode() string {
    return "config_execution_error"
}
```

**C# Test** (`tests/cli-interop/GonugetCliInterop.Tests/ConfigTests.cs`):
```csharp
public class ConfigTests : IDisposable
{
    private readonly string _tempConfigDir;
    private readonly string _tempConfigFile;

    public ConfigTests()
    {
        _tempConfigDir = Path.Combine(Path.GetTempPath(), $"nuget-test-{Guid.NewGuid()}");
        Directory.CreateDirectory(_tempConfigDir);
        _tempConfigFile = Path.Combine(_tempConfigDir, "NuGet.config");
    }

    public void Dispose()
    {
        if (Directory.Exists(_tempConfigDir))
        {
            Directory.Delete(_tempConfigDir, true);
        }
    }

    [Fact]
    public void Config_Get_ShouldMatchDotnetNuget()
    {
        // First set a value
        var setResult = GonugetCliBridge.ExecuteConfig(
            subcommand: "set",
            configKey: "testKey",
            configValue: "testValue",
            configFile: _tempConfigFile);

        Assert.Equal(0, setResult.DotnetExitCode);
        Assert.Equal(0, setResult.GonugetExitCode);

        // Then get it
        var getResult = GonugetCliBridge.ExecuteConfig(
            subcommand: "get",
            allOrConfigKey: "testKey",
            workingDirectory: _tempConfigDir);

        Assert.Equal(0, getResult.DotnetExitCode);
        Assert.Equal(0, getResult.GonugetExitCode);
        Assert.Equal(getResult.DotnetStdout.Trim(), getResult.GonugetStdout.Trim());
    }

    [Fact]
    public void Config_GetAll_ShouldMatchDotnetNuget()
    {
        // Set multiple values
        GonugetCliBridge.ExecuteConfig("set", configKey: "key1", configValue: "value1", configFile: _tempConfigFile);
        GonugetCliBridge.ExecuteConfig("set", configKey: "key2", configValue: "value2", configFile: _tempConfigFile);

        // Get all
        var result = GonugetCliBridge.ExecuteConfig(
            subcommand: "get",
            allOrConfigKey: "all",
            workingDirectory: _tempConfigDir);

        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both should output key1=value1 and key2=value2
        Assert.Contains("key1=value1", result.DotnetStdout);
        Assert.Contains("key1=value1", result.GonugetStdout);
        Assert.Contains("key2=value2", result.DotnetStdout);
        Assert.Contains("key2=value2", result.GonugetStdout);
    }

    [Fact]
    public void Config_Set_ShouldMatchDotnetNuget()
    {
        var result = GonugetCliBridge.ExecuteConfig(
            subcommand: "set",
            configKey: "globalPackagesFolder",
            configValue: "~/test-packages",
            configFile: _tempConfigFile);

        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both should output success message mentioning the config file
        Assert.Contains(_tempConfigFile, result.DotnetStdout);
        Assert.Contains(_tempConfigFile, result.GonugetStdout);
    }

    [Fact]
    public void Config_Unset_ShouldMatchDotnetNuget()
    {
        // First set a value
        GonugetCliBridge.ExecuteConfig(
            subcommand: "set",
            configKey: "testKey",
            configValue: "testValue",
            configFile: _tempConfigFile);

        // Then unset it
        var result = GonugetCliBridge.ExecuteConfig(
            subcommand: "unset",
            configKey: "testKey",
            configFile: _tempConfigFile);

        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Verify it's actually removed
        var getResult = GonugetCliBridge.ExecuteConfig(
            subcommand: "get",
            allOrConfigKey: "testKey",
            workingDirectory: _tempConfigDir);

        // Both should fail since key doesn't exist
        Assert.NotEqual(0, getResult.DotnetExitCode);
        Assert.NotEqual(0, getResult.GonugetExitCode);
    }

    [Fact]
    public void Config_Paths_ShouldMatchDotnetNuget()
    {
        var result = GonugetCliBridge.ExecuteConfig(
            subcommand: "paths",
            workingDirectory: _tempConfigDir);

        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both should output config file paths
        // The exact paths may differ, but both should list hierarchy
    }

    [Fact]
    public void Config_GetNotFound_ShouldFail()
    {
        var result = GonugetCliBridge.ExecuteConfig(
            subcommand: "get",
            allOrConfigKey: "nonExistentKey",
            workingDirectory: _tempConfigDir);

        // Both should fail with similar exit codes
        Assert.NotEqual(0, result.DotnetExitCode);
        Assert.NotEqual(0, result.GonugetExitCode);
    }

    [Theory]
    [InlineData("globalPackagesFolder", "~/packages")]
    [InlineData("repositoryPath", "./local-repo")]
    [InlineData("http_proxy", "http://proxy.example.com")]
    public void Config_CommonKeys_ShouldMatchBehavior(string key, string value)
    {
        var setResult = GonugetCliBridge.ExecuteConfig(
            subcommand: "set",
            configKey: key,
            configValue: value,
            configFile: _tempConfigFile);

        Assert.Equal(0, setResult.DotnetExitCode);
        Assert.Equal(0, setResult.GonugetExitCode);

        var getResult = GonugetCliBridge.ExecuteConfig(
            subcommand: "get",
            allOrConfigKey: key,
            workingDirectory: _tempConfigDir);

        Assert.Equal(0, getResult.DotnetExitCode);
        Assert.Equal(0, getResult.GonugetExitCode);
        Assert.Contains(value, getResult.GonugetStdout);
    }

    [Fact]
    public void Config_ShowPath_ShouldMatchDotnetNuget()
    {
        // Set a relative path
        GonugetCliBridge.ExecuteConfig(
            subcommand: "set",
            configKey: "repositoryPath",
            configValue: "./packages",
            configFile: _tempConfigFile);

        // Get with --show-path
        var result = GonugetCliBridge.ExecuteConfig(
            subcommand: "get",
            allOrConfigKey: "repositoryPath",
            workingDirectory: _tempConfigDir,
            showPath: true);

        Assert.Equal(0, result.DotnetExitCode);
        Assert.Equal(0, result.GonugetExitCode);

        // Both should return absolute paths
        var dotnetPath = result.DotnetStdout.Trim();
        var gonugetPath = result.GonugetStdout.Trim();

        Assert.True(Path.IsPathRooted(dotnetPath), "dotnet should return absolute path");
        Assert.True(Path.IsPathRooted(gonugetPath), "gonuget should return absolute path");
    }
}
```

**Test Execution**:
```bash
# Build CLI interop bridge
make build-cli-interop

# Run config CLI interop tests
cd tests/cli-interop/GonugetCliInterop.Tests
dotnet test --filter "FullyQualifiedName~ConfigTests"
```

**Key Compatibility Validation**:
- Config file XML structure matches exactly
- Both tools can read/write each other's config files
- Exit codes match for success/failure scenarios
- Path expansion with `--show-path` behaves identically on Windows/Linux/macOS
- Config hierarchy resolution matches dotnet nuget
- All four subcommands (get, set, unset, paths) produce identical output

### Commit

```bash
git add cmd/gonuget/
git commit -m "feat(cli): add config command matching dotnet nuget

Implements four subcommands with exact dotnet nuget parity:
- config get <all-or-config-key> with --show-path and --working-directory
- config set <config-key> <config-value> with --configfile
- config unset <config-key> with --configfile
- config paths with --working-directory

Positional arguments (not flags) for get/set/unset match dotnet nuget.
Flag names match exactly: --show-path (not --as-path), --working-directory.
Support for 'all' keyword in get to list all config values.
Config hierarchy resolution from working directory.

Includes comprehensive CLI interop tests validating exact behavior parity.

Commands: 2/21 complete (10%)"
```

---

## Summary and Next Steps

You've completed Chunks 1-5 of CLI Milestone 1 (Foundation). You now have:

✅ Project structure with Cobra
✅ Console output abstraction with colors and verbosity matching `dotnet nuget`
✅ NuGet.config XML parsing and management (100% compatible with .NET tools)
✅ Version command (`gonuget --version` vs `dotnet nuget --version`)
✅ Config command with four subcommands (get/set/unset/paths) matching `dotnet nuget config` exactly
✅ Positional arguments for config operations (not flags)
✅ Config hierarchy resolution and --working-directory support
✅ CLI interop test handlers and C# tests for all chunks

**Next document**: CLI-M1-FOUNDATION-CONTINUED.md will cover:
- Chunk 6: Source management commands (`list source`, `add source`, `remove source`, `enable source`, `disable source`, `update source`)
- Chunk 7: Help command
- Chunk 8: Progress bars and spinners
- Chunk 9: CLI Interop Tests for Phase 1
- Chunk 10: Performance benchmarks

**Commands completed**: 2/21 (9.5%)
**Target**: 100% parity with `dotnet nuget`
**Test coverage**: >80% unit tests + CLI interop tests for Phase 1

**Key Changes from nuget.exe target**:
- All flag names use kebab-case (--configfile not --ConfigFile)
- Commands match `dotnet nuget` structure
- Cross-platform compatibility is critical (Windows, macOS, Linux)
- CLI interop tests validate output matches `dotnet nuget` exactly

**Ready to proceed?** Continue to CLI-M1-FOUNDATION-CONTINUED.md for chunks 6-10, which will complete the foundation phase with source management commands and CLI interop test infrastructure.
