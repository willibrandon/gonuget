//go:build benchmark
// +build benchmark

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// Benchmark CLI startup time (target: <50ms P50)
func BenchmarkStartup(b *testing.B) {
	binPath := buildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		cmd := exec.Command(binPath, "version")
		if err := cmd.Run(); err != nil {
			b.Fatalf("command failed: %v", err)
		}
		elapsed := time.Since(start)

		b.ReportMetric(float64(elapsed.Microseconds()), "μs/op")
	}
}

// Benchmark version command performance
func BenchmarkVersionCommand(b *testing.B) {
	binPath := buildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binPath, "version")
		cmd.Stdout = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			b.Fatalf("version command failed: %v", err)
		}
	}
}

// Benchmark config read operations
// Matches: dotnet nuget config get <key>
func BenchmarkConfigRead(b *testing.B) {
	binPath := buildBinary(b)

	// Create temp directory with config file
	tempDir := b.TempDir()
	configFile := filepath.Join(tempDir, "NuGet.config")
	config := `<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <config>
    <add key="globalPackagesFolder" value="/tmp/packages" />
  </config>
</configuration>`
	if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
		b.Fatalf("failed to write config: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binPath, "config", "get", "globalPackagesFolder", "--working-directory", tempDir)
		cmd.Stdout = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			b.Fatalf("config command failed: %v", err)
		}
	}
}

// Benchmark config write operations
// Matches: dotnet nuget config set <key> <value>
func BenchmarkConfigWrite(b *testing.B) {
	binPath := buildBinary(b)
	tempDir := b.TempDir()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		configFile := filepath.Join(tempDir, "config-"+string(rune('A'+i%26))+".xml")
		cmd := exec.Command(binPath, "config", "set", "repositoryPath", "/tmp/packages", "--configfile", configFile)
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			b.Fatalf("config write failed: %v", err)
		}
	}
}

// Benchmark list source operation
// Matches: dotnet nuget list source
func BenchmarkListSource(b *testing.B) {
	binPath := buildBinary(b)
	configFile := setupTestConfigWithSources(b, 10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binPath, "source", "list", "--configfile", configFile)
		cmd.Stdout = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			b.Fatalf("list source failed: %v", err)
		}
	}
}

// Benchmark add source operation
// Matches: dotnet nuget add source <url> --name <name>
func BenchmarkAddSource(b *testing.B) {
	binPath := buildBinary(b)
	tempDir := b.TempDir()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use unique config file for each iteration to avoid conflicts
		configFile := filepath.Join(tempDir, fmt.Sprintf("source-%d.xml", i))
		cmd := exec.Command(binPath, "source", "add",
			"https://test.example.com/v3/index.json",
			"--configfile", configFile,
			"--name", "TestFeed")
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			stderr := cmd.Stderr.(*bytes.Buffer).String()
			b.Fatalf("add source failed: %v, stderr: %s", err, stderr)
		}
	}
}

// Benchmark help command
func BenchmarkHelpCommand(b *testing.B) {
	binPath := buildBinary(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binPath, "--help")
		cmd.Stdout = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			b.Fatalf("help command failed: %v", err)
		}
	}
}

// Benchmark dotnet nuget version command for comparison
func BenchmarkDotnetVersion(b *testing.B) {
	// Check if dotnet is available
	if _, err := exec.LookPath("dotnet"); err != nil {
		b.Skip("dotnet not available")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command("dotnet", "nuget", "--version")
		cmd.Stdout = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			b.Fatalf("dotnet command failed: %v", err)
		}
	}
}

// Benchmark dotnet nuget config get for comparison
func BenchmarkDotnetConfigGet(b *testing.B) {
	if _, err := exec.LookPath("dotnet"); err != nil {
		b.Skip("dotnet not available")
	}

	// Create a temp directory with NuGet.config
	tempDir := b.TempDir()
	configFile := filepath.Join(tempDir, "NuGet.config")
	config := `<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <config>
    <add key="globalPackagesFolder" value="/tmp/packages" />
  </config>
</configuration>`
	if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
		b.Fatalf("failed to write config: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command("dotnet", "nuget", "config", "get", "globalPackagesFolder", "--working-directory", tempDir)
		cmd.Stdout = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			b.Fatalf("dotnet config failed: %v", err)
		}
	}
}

// Benchmark dotnet nuget list source for comparison
func BenchmarkDotnetListSource(b *testing.B) {
	if _, err := exec.LookPath("dotnet"); err != nil {
		b.Skip("dotnet not available")
	}

	// Create temp directory with NuGet.config containing sources
	tempDir := b.TempDir()
	configFile := filepath.Join(tempDir, "NuGet.config")

	config := `<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <packageSources>`
	for i := 0; i < 10; i++ {
		config += fmt.Sprintf("\n    <add key=\"Feed%c\" value=\"https://feed%c.com/v3/index.json\" />", 'A'+i, 'A'+i)
	}
	config += `
  </packageSources>
</configuration>`

	if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
		b.Fatalf("failed to write config: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command("dotnet", "nuget", "list", "source", "--configfile", configFile)
		cmd.Stdout = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			b.Fatalf("dotnet list source failed: %v", err)
		}
	}
}

// Helper: Build binary for benchmarks
func buildBinary(b *testing.B) string {
	b.Helper()

	binaryName := "gonuget"
	if runtime.GOOS == "windows" {
		binaryName = "gonuget.exe"
	}

	binPath := filepath.Join(b.TempDir(), binaryName)
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	if err := cmd.Run(); err != nil {
		b.Fatalf("failed to build binary: %v", err)
	}

	return binPath
}

// Helper: Setup test config with multiple sources
func setupTestConfigWithSources(b *testing.B, count int) string {
	b.Helper()

	configFile := filepath.Join(b.TempDir(), "NuGet.config")

	config := `<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <packageSources>`

	for i := 0; i < count; i++ {
		config += "\n    <add key=\"Feed" + string(rune('A'+i)) + "\" value=\"https://feed" + string(rune('A'+i)) + ".com/v3/index.json\" />"
	}

	config += `
  </packageSources>
</configuration>`

	if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
		b.Fatalf("failed to write config: %v", err)
	}

	return configFile
}
