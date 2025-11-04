package restore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockTTYDetector allows simulating TTY or piped mode for tests
type mockTTYDetector struct {
	isTTY  bool
	width  int
	height int
}

func (m *mockTTYDetector) IsTTY(w io.Writer) bool {
	return m.isTTY
}

func (m *mockTTYDetector) GetSize(w io.Writer) (width, height int, err error) {
	if !m.isTTY {
		return 0, 0, os.ErrInvalid
	}
	return m.width, m.height, nil
}

// safeBuffer wraps bytes.Buffer with mutex protection for concurrent access
type safeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (sb *safeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *safeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

// mockConsoleForOutputMode extends mockConsole to support Output()
type mockConsoleForOutputMode struct {
	output   *safeBuffer
	messages []string
	mu       sync.Mutex
}

func (m *mockConsoleForOutputMode) Printf(format string, args ...any) {
	var msg string
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	} else {
		msg = format
	}
	m.mu.Lock()
	m.messages = append(m.messages, msg)
	m.mu.Unlock()
	m.output.Write([]byte(msg + "\n"))
}

func (m *mockConsoleForOutputMode) Error(format string, args ...any) {
	m.Printf(format, args...)
}

func (m *mockConsoleForOutputMode) Warning(format string, args ...any) {
	m.Printf(format, args...)
}

func (m *mockConsoleForOutputMode) Output() io.Writer {
	return m.output
}

func TestTerminalStatus_TTYMode(t *testing.T) {
	t.Parallel()

	// Create mock TTY detector that simulates a terminal
	detector := &mockTTYDetector{
		isTTY:  true,
		width:  120,
		height: 24,
	}

	var output bytes.Buffer
	status := NewTerminalStatus(&output, "test.csproj", detector)

	// Give it time to update a few times (30Hz = ~33ms per update)
	time.Sleep(100 * time.Millisecond)

	status.Stop()

	// In TTY mode, we expect:
	// 1. ANSI escape sequences for cursor positioning
	// 2. Status messages like "Restore (0.1s)"
	// 3. Clear line escape sequence at the end

	outputStr := output.String()

	// Check for ANSI escape sequences (cursor hide/show, positioning)
	if !strings.Contains(outputStr, "\x1B[") {
		t.Error("Expected ANSI escape sequences in TTY mode output")
	}

	// Check for "Restore" status messages
	if !strings.Contains(outputStr, "Restore") {
		t.Error("Expected 'Restore' status messages in TTY mode output")
	}

	// Check that IsTTY returns true
	if !status.IsTTY() {
		t.Error("Expected IsTTY() to return true in TTY mode")
	}
}

func TestTerminalStatus_PipedMode(t *testing.T) {
	t.Parallel()

	// Create mock TTY detector that simulates piped output (not a TTY)
	detector := &mockTTYDetector{
		isTTY:  false,
		width:  0,
		height: 0,
	}

	var output bytes.Buffer
	status := NewTerminalStatus(&output, "test.csproj", detector)

	// Give it time to see if any updates happen
	time.Sleep(100 * time.Millisecond)

	status.Stop()

	// In piped mode, we expect:
	// 1. NO ANSI escape sequences
	// 2. NO status updates (ticker should not be running)
	// 3. Empty or minimal output

	outputStr := output.String()

	// Check that no ANSI escape sequences are present
	if strings.Contains(outputStr, "\x1B[") {
		t.Errorf("Expected NO ANSI escape sequences in piped mode, got: %q", outputStr)
	}

	// Check that IsTTY returns false
	if status.IsTTY() {
		t.Error("Expected IsTTY() to return false in piped mode")
	}
}

func TestRun_TTYMode_Output(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup: Create temp directory with test project
	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	// Create minimal .csproj file
	csprojContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(csprojContent), 0644); err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	// Create mock console with TTY detector
	console := &mockConsoleForOutputMode{
		output:   &safeBuffer{},
		messages: []string{},
	}

	ttyDetector := &mockTTYDetector{
		isTTY:  true,
		width:  120,
		height: 24,
	}

	// Temporarily replace DefaultTTYDetector for this test
	oldDetector := DefaultTTYDetector
	DefaultTTYDetector = ttyDetector
	defer func() { DefaultTTYDetector = oldDetector }()

	// Run restore with TTY mode
	ctx := context.Background()
	opts := &Options{
		Sources:   []string{"https://api.nuget.org/v3/index.json"},
		Verbosity: "detailed", // Enable detailed output to see "Determining projects" message
	}

	err := Run(ctx, []string{projPath}, opts, console)
	if err != nil {
		t.Logf("Restore output: %s", console.output.String())
		t.Logf("Messages: %v", console.messages)
		// Restore might fail due to network issues - not the focus of this test
		t.Skipf("Restore failed (may be network): %v", err)
	}

	// In TTY mode, we expect:
	// 1. "Determining projects to restore..." message
	// 2. Live status updates (via TerminalStatus)
	// 3. Final "Restore succeeded" message with timing

	foundDetermining := false
	foundSucceeded := false

	for _, msg := range console.messages {
		if strings.Contains(msg, "Determining projects to restore") {
			foundDetermining = true
		}
		if strings.Contains(msg, "Restore") && strings.Contains(msg, "succeeded") {
			foundSucceeded = true
		}
	}

	if !foundDetermining {
		t.Errorf("Expected 'Determining projects to restore' message in TTY mode, got messages: %v", console.messages)
	}

	if !foundSucceeded {
		t.Errorf("Expected 'Restore succeeded' message in TTY mode, got messages: %v", console.messages)
	}

	// Check that output contains ANSI sequences (from TerminalStatus)
	outputStr := console.output.String()
	if !strings.Contains(outputStr, "\x1B[") {
		t.Error("Expected ANSI escape sequences in TTY mode restore output")
	}
}

func TestRun_PipedMode_Output(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup: Create temp directory with test project
	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")

	// Create minimal .csproj file
	csprojContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(csprojContent), 0644); err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	// Create mock console with piped mode detector
	console := &mockConsoleForOutputMode{
		output:   &safeBuffer{},
		messages: []string{},
	}

	pipedDetector := &mockTTYDetector{
		isTTY:  false,
		width:  0,
		height: 0,
	}

	// Temporarily replace DefaultTTYDetector for this test
	oldDetector := DefaultTTYDetector
	DefaultTTYDetector = pipedDetector
	defer func() { DefaultTTYDetector = oldDetector }()

	// Run restore with piped mode
	ctx := context.Background()
	opts := &Options{
		Sources:   []string{"https://api.nuget.org/v3/index.json"},
		Verbosity: "detailed", // Enable detailed output to see "Committing restore" message
	}

	err := Run(ctx, []string{projPath}, opts, console)
	if err != nil {
		t.Logf("Restore output: %s", console.output.String())
		t.Logf("Messages: %v", console.messages)
		// Restore might fail due to network issues - not the focus of this test
		t.Skipf("Restore failed (may be network): %v", err)
	}

	// In piped mode with detailed verbosity, we expect:
	// 1. "Committing restore..." message (NOT "Determining projects to restore...")
	// 2. NO live status updates
	// 3. "Restored <project>" message OR "All projects are up-to-date" (for cache hits)
	// 4. NO ANSI escape sequences
	// Note: "Restore succeeded" is NOT shown in piped mode for fresh restores

	foundCommitting := false
	foundDetermining := false
	foundRestored := false

	for _, msg := range console.messages {
		if strings.Contains(msg, "Committing restore") {
			foundCommitting = true
		}
		if strings.Contains(msg, "Determining projects to restore") {
			foundDetermining = true
		}
		if strings.Contains(msg, "Restored") || strings.Contains(msg, "up-to-date") {
			foundRestored = true
		}
	}

	if !foundCommitting {
		t.Errorf("Expected 'Committing restore' message in piped mode, got messages: %v", console.messages)
	}

	if foundDetermining {
		t.Errorf("Expected NO 'Determining projects to restore' message in piped mode, got messages: %v", console.messages)
	}

	if !foundRestored {
		t.Errorf("Expected 'Restored' or 'up-to-date' message in piped mode, got messages: %v", console.messages)
	}

	// Check that output does NOT contain ANSI sequences
	outputStr := console.output.String()
	if strings.Contains(outputStr, "\x1B[") {
		t.Errorf("Expected NO ANSI escape sequences in piped mode, got output: %q", outputStr)
	}
}

func TestRealTTYDetector_WithFile(t *testing.T) {
	t.Parallel()

	detector := &RealTTYDetector{}

	// Test with os.Stdout (may or may not be a TTY depending on test environment)
	isTTY := detector.IsTTY(os.Stdout)

	// We can't assert true or false here because it depends on how tests are run
	// But we can check that the method doesn't panic
	t.Logf("os.Stdout is TTY: %v", isTTY)

	if isTTY {
		// If it's a TTY, GetSize should succeed
		width, height, err := detector.GetSize(os.Stdout)
		if err != nil {
			t.Errorf("GetSize failed on TTY: %v", err)
		}
		if width <= 0 || height <= 0 {
			t.Errorf("GetSize returned invalid dimensions: %dx%d", width, height)
		}
		t.Logf("Terminal size: %dx%d", width, height)
	} else {
		// If it's not a TTY, GetSize should fail
		_, _, err := detector.GetSize(os.Stdout)
		if err == nil {
			t.Error("GetSize should fail on non-TTY")
		}
	}
}

func TestRealTTYDetector_WithNonFile(t *testing.T) {
	t.Parallel()

	detector := &RealTTYDetector{}

	// Test with a bytes.Buffer (not a file, definitely not a TTY)
	var buf bytes.Buffer

	isTTY := detector.IsTTY(&buf)
	if isTTY {
		t.Error("bytes.Buffer should not be detected as TTY")
	}

	_, _, err := detector.GetSize(&buf)
	if err == nil {
		t.Error("GetSize should fail on bytes.Buffer")
	}
}

func TestDefaultTTYDetector(t *testing.T) {
	t.Parallel()

	// Verify DefaultTTYDetector is set and is a RealTTYDetector
	if DefaultTTYDetector == nil {
		t.Fatal("DefaultTTYDetector should not be nil")
	}

	if _, ok := DefaultTTYDetector.(*RealTTYDetector); !ok {
		t.Errorf("DefaultTTYDetector should be *RealTTYDetector, got: %T", DefaultTTYDetector)
	}
}
