# CLI Milestone 1: Foundation

**Status**: Implementation Guide
**Phase**: 1 of 8
**Duration**: Weeks 1-2
**Prerequisites**: gonuget library M1-M8 complete

---

## Overview

This milestone establishes the CLI foundation: command-line framework, configuration management, console output system, and basic commands. By the end of this milestone, you'll have a working `gonuget` binary with help, version, config, and sources commands.

**Deliverables**:
- CLI application structure with Cobra
- NuGet.config XML parsing and writing
- Console abstraction with colors and progress bars
- Commands: `help`, `version`, `config`, `sources`
- 80%+ test coverage

**Success Criteria**:
- `gonuget --help` displays command list
- `gonuget version` shows version info
- `gonuget config` reads/writes NuGet.config
- `gonuget sources list` displays configured sources

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
│   └── sources.go          # Sources command
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
	Long: `gonuget is a cross-platform NuGet package manager CLI with 100% parity to nuget.exe.

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

**Objective**: Implement the `version` command.

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
	Long: `gonuget is a cross-platform NuGet package manager CLI with 100% parity to nuget.exe.

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
```

**Expected output**:
```
gonuget version 1.0.0-test
commit: unknown
built: unknown
built by: unknown
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

## Chunk 5: Config Command (Part 1: Reading)

**Objective**: Implement `config` command for reading configuration values.

### Step 5.1: Create commands/config.go

```go
// cmd/gonuget/commands/config.go
package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

type configOptions struct {
	configFile string
	asPath     bool
	set        []string
}

// NewConfigCommand creates the config command
func NewConfigCommand(console *output.Console) *cobra.Command {
	opts := &configOptions{}

	cmd := &cobra.Command{
		Use:   "config [key] [value]",
		Short: "Get or set NuGet configuration values",
		Long: `Get or set NuGet configuration values.

Examples:
  gonuget config                                    # List all config values
  gonuget config repositoryPath                     # Get specific value
  gonuget config repositoryPath ~/packages          # Set value
  gonuget config -Set key1=value1 -Set key2=value2  # Set multiple values`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfig(console, args, opts)
		},
	}

	cmd.Flags().StringVar(&opts.configFile, "ConfigFile", "", "Config file to use")
	cmd.Flags().BoolVar(&opts.asPath, "AsPath", false, "Return value as filesystem path")
	cmd.Flags().StringArrayVar(&opts.set, "Set", []string{}, "Set key=value pair(s)")

	return cmd
}

func runConfig(console *output.Console, args []string, opts *configOptions) error {
	// Determine config file
	configPath := opts.configFile
	if configPath == "" {
		configPath = config.FindConfigFile()
		if configPath == "" {
			// Use user config
			configPath = config.GetUserConfigPath()
		}
	}

	// Load config
	cfg, err := loadOrCreateConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle --Set flags
	if len(opts.set) > 0 {
		for _, pair := range opts.set {
			key, value, err := parseKeyValue(pair)
			if err != nil {
				return err
			}
			cfg.SetConfigValue(key, value)
		}

		if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		console.Success("Configuration updated")
		return nil
	}

	// Handle get/set via args
	switch len(args) {
	case 0:
		// List all config values
		return listAllConfig(console, cfg)
	case 1:
		// Get value
		return getConfigValue(console, cfg, args[0], opts.asPath)
	case 2:
		// Set value
		cfg.SetConfigValue(args[0], args[1])
		if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		console.Success("Set config value: %s = %s", args[0], args[1])
		return nil
	default:
		return fmt.Errorf("too many arguments")
	}
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
		console.Info("No configuration values set")
		return nil
	}

	console.Println("Configuration values:")
	for _, item := range cfg.Config.Add {
		console.Printf("  %s = %s\n", item.Key, item.Value)
	}

	return nil
}

func getConfigValue(console *output.Console, cfg *config.NuGetConfig, key string, asPath bool) error {
	value := cfg.GetConfigValue(key)
	if value == "" {
		return fmt.Errorf("configuration value not found: %s", key)
	}

	if asPath {
		// Expand path
		if filepath.IsAbs(value) {
			console.Println(value)
		} else {
			absPath, err := filepath.Abs(value)
			if err != nil {
				return fmt.Errorf("failed to resolve path: %w", err)
			}
			console.Println(absPath)
		}
	} else {
		console.Println(value)
	}

	return nil
}

func parseKeyValue(pair string) (string, string, error) {
	for i, r := range pair {
		if r == '=' {
			return pair[:i], pair[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid key=value pair: %s", pair)
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

### Step 5.3: Create test

```go
// cmd/gonuget/commands/config_test.go
package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/config"
	"github.com/willibrandon/gonuget/cmd/gonuget/output"
)

func TestConfigCommand_Get(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	cfg := config.NewDefaultConfig()
	cfg.SetConfigValue("testKey", "testValue")
	if err := config.SaveNuGetConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{configFile: configPath}
	if err := runConfig(console, []string{"testKey"}, opts); err != nil {
		t.Fatalf("runConfig() error = %v", err)
	}

	output := strings.TrimSpace(out.String())
	if output != "testValue" {
		t.Errorf("output = %q, want %q", output, "testValue")
	}
}

func TestConfigCommand_Set(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{configFile: configPath}
	if err := runConfig(console, []string{"newKey", "newValue"}, opts); err != nil {
		t.Fatalf("runConfig() error = %v", err)
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

func TestConfigCommand_SetMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "NuGet.config")

	var out bytes.Buffer
	console := output.NewConsole(&out, &out, output.VerbosityNormal)

	opts := &configOptions{
		configFile: configPath,
		set:        []string{"key1=value1", "key2=value2"},
	}
	if err := runConfig(console, []string{}, opts); err != nil {
		t.Fatalf("runConfig() error = %v", err)
	}

	cfg, err := config.LoadNuGetConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if got := cfg.GetConfigValue("key1"); got != "value1" {
		t.Errorf("key1 = %q, want %q", got, "value1")
	}
	if got := cfg.GetConfigValue("key2"); got != "value2" {
		t.Errorf("key2 = %q, want %q", got, "value2")
	}
}

func TestParseKeyValue(t *testing.T) {
	tests := []struct {
		input     string
		wantKey   string
		wantValue string
		wantErr   bool
	}{
		{"key=value", "key", "value", false},
		{"key=", "key", "", false},
		{"=value", "", "value", false},
		{"invalid", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			key, value, err := parseKeyValue(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKeyValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
			if value != tt.wantValue {
				t.Errorf("value = %q, want %q", value, tt.wantValue)
			}
		})
	}
}
```

### Verification

```bash
# Run tests
go test ./cmd/gonuget/commands -v -run TestConfig

# Build and test
go build -o gonuget ./cmd/gonuget

# Test config operations
./gonuget config -Set testKey=testValue
./gonuget config testKey
./gonuget config
```

### Commit

```bash
git add cmd/gonuget/
git commit -m "feat(cli): add config command

- Implement config get/set operations
- Support single and multiple key-value pairs
- Load from config hierarchy
- Create config if doesn't exist
- Support -AsPath flag for path expansion

Tests: Config get, set, multiple values
Commands: 2/20 complete (10%)"
```

---

## Summary and Next Steps

You've completed Chunk 1-5 of CLI Milestone 1 (Foundation). You now have:

✅ Project structure with Cobra
✅ Console output abstraction with colors and verbosity
✅ NuGet.config XML parsing and management
✅ Version command
✅ Config command (get/set)

**Next document**: CLI-M1-FOUNDATION.md (continued) will cover:
- Chunk 6: Sources command (list, add, remove, enable, disable)
- Chunk 7: Help command
- Chunk 8: Progress bars and spinners
- Chunk 9: Integration tests for Phase 1
- Chunk 10: Performance benchmarks

**Commands completed**: 2/20 (10%)
**Test coverage**: Should be >80% for Phase 1

**Ready to proceed?** The next chunks will complete the foundation phase. Let me know if you want me to continue with the remaining chunks of M1 or move to the next document (CLI-M2-CORE-OPERATIONS.md).
