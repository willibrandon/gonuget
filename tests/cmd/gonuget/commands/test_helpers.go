// Package commands provides test helpers for CLI command testing.
package commands

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// BuildBinary builds the gonuget binary for testing and returns its path.
// This is used by all command tests to ensure the binary exists before running tests.
func BuildBinary(t *testing.T) string {
	t.Helper()

	binaryName := "gonuget"
	if runtime.GOOS == "windows" {
		binaryName = "gonuget.exe"
	}

	binPath := filepath.Join(t.TempDir(), binaryName)
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/willibrandon/gonuget/cmd/gonuget")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	return binPath
}

// BuildBinaryForBenchmark builds the gonuget binary for benchmarking and returns its path.
// This is used by all command benchmarks to ensure the binary exists before running benchmarks.
func BuildBinaryForBenchmark(b *testing.B) string {
	b.Helper()

	binaryName := "gonuget"
	if runtime.GOOS == "windows" {
		binaryName = "gonuget.exe"
	}

	binPath := filepath.Join(b.TempDir(), binaryName)
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/willibrandon/gonuget/cmd/gonuget")
	if err := cmd.Run(); err != nil {
		b.Fatalf("failed to build binary: %v", err)
	}

	return binPath
}
