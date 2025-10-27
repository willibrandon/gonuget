package restore

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
	"github.com/willibrandon/gonuget/core"
)

// BenchmarkRestore_V2_NewtonsoftJson benchmarks V2 restore for Newtonsoft.Json
// Compare with: dotnet restore --source https://www.nuget.org/api/v2
func BenchmarkRestore_V2_NewtonsoftJson(b *testing.B) {
	// Create temp project
	tmpDir := b.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	proj, err := project.LoadProject(projPath)
	if err != nil {
		b.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")
	opts := &Options{
		PackagesFolder: packagesFolder,
		Sources:        []string{"https://www.nuget.org/api/v2"},
	}

	packageRefs := proj.GetPackageReferences()

	b.ResetTimer()
	for b.Loop() {
		// Clear packages folder for each iteration
		_ = os.RemoveAll(packagesFolder)

		console := &mockConsole{}
		restorer := NewRestorer(opts, console)

		_, err := restorer.Restore(context.Background(), proj, packageRefs)
		if err != nil {
			b.Fatalf("Restore failed: %v", err)
		}
	}
}

// BenchmarkDotnetRestore_V2_NewtonsoftJson benchmarks dotnet restore with V2 source
// This provides direct comparison data
func BenchmarkDotnetRestore_V2_NewtonsoftJson(b *testing.B) {
	// Create temp project
	tmpDir := b.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")

	b.ResetTimer()
	for b.Loop() {
		// Clear packages and obj folders for each iteration
		_ = os.RemoveAll(packagesFolder)
		_ = os.RemoveAll(filepath.Join(tmpDir, "obj"))
		_ = os.RemoveAll(filepath.Join(tmpDir, "bin"))

		cmd := exec.Command("dotnet", "restore",
			"--source", "https://www.nuget.org/api/v2",
			"--packages", packagesFolder,
			projPath)
		cmd.Dir = tmpDir

		if err := cmd.Run(); err != nil {
			b.Fatalf("dotnet restore failed: %v", err)
		}
	}
}

// Standalone timing comparison (not a benchmark, for manual testing)
func TestCompareRestorePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison in short mode")
	}

	// Clear protocol cache to ensure clean state
	// The fast-path optimization requires clearing stale cache entries
	_ = core.ClearProtocolCache()

	// Clear both gonuget and dotnet HTTP caches for fair comparison
	// dotnet stores cache at ~/.local/share/NuGet/http-cache on macOS/Linux
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		_ = os.RemoveAll(filepath.Join(homeDir, ".gonuget"))
		_ = os.RemoveAll(filepath.Join(homeDir, ".local", "share", "NuGet", "http-cache"))
	}

	tmpDir := t.TempDir()
	projPath := filepath.Join(tmpDir, "test.csproj")
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	if err := os.WriteFile(projPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	packagesFolder := filepath.Join(tmpDir, "packages")

	// Benchmark gonuget (5 runs)
	t.Log("=== gonuget restore (V2) ===")
	var gonugetTimes []time.Duration
	for i := range 5 {
		_ = os.RemoveAll(packagesFolder)

		proj, _ := project.LoadProject(projPath)
		opts := &Options{
			PackagesFolder: packagesFolder,
			Sources:        []string{"https://www.nuget.org/api/v2"},
		}
		restorer := NewRestorer(opts, &mockConsole{})
		packageRefs := proj.GetPackageReferences()

		start := time.Now()
		_, err := restorer.Restore(context.Background(), proj, packageRefs)
		elapsed := time.Since(start)

		if err != nil {
			t.Logf("  Run %d: FAILED - %v", i+1, err)
			continue
		}

		gonugetTimes = append(gonugetTimes, elapsed)
		t.Logf("  Run %d: %v", i+1, elapsed)

		time.Sleep(time.Second) // Avoid rate limiting
	}

	// Benchmark dotnet (5 runs)
	t.Log("\n=== dotnet restore (V2) ===")
	var dotnetTimes []time.Duration
	for i := range 5 {
		_ = os.RemoveAll(packagesFolder)
		_ = os.RemoveAll(filepath.Join(tmpDir, "obj"))
		_ = os.RemoveAll(filepath.Join(tmpDir, "bin"))

		cmd := exec.Command("dotnet", "restore",
			"--source", "https://www.nuget.org/api/v2",
			"--packages", packagesFolder,
			projPath)
		cmd.Dir = tmpDir

		start := time.Now()
		err := cmd.Run()
		elapsed := time.Since(start)

		if err != nil {
			t.Logf("  Run %d: FAILED - %v", i+1, err)
			continue
		}

		dotnetTimes = append(dotnetTimes, elapsed)
		t.Logf("  Run %d: %v", i+1, elapsed)

		time.Sleep(time.Second) // Avoid rate limiting
	}

	// Calculate averages
	var gonugetSum, dotnetSum time.Duration
	for _, d := range gonugetTimes {
		gonugetSum += d
	}
	for _, d := range dotnetTimes {
		dotnetSum += d
	}

	if len(gonugetTimes) > 0 && len(dotnetTimes) > 0 {
		gonugetAvg := gonugetSum / time.Duration(len(gonugetTimes))
		dotnetAvg := dotnetSum / time.Duration(len(dotnetTimes))

		t.Log("\n=== Results ===")
		t.Logf("gonuget average: %v", gonugetAvg)
		t.Logf("dotnet average:  %v", dotnetAvg)

		ratio := float64(gonugetAvg) / float64(dotnetAvg)
		if ratio < 1.0 {
			t.Logf("gonuget is %.2fx FASTER than dotnet", 1.0/ratio)
		} else {
			t.Logf("gonuget is %.2fx slower than dotnet", ratio)
		}
	}
}
