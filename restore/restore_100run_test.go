package restore

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
	"github.com/willibrandon/gonuget/core"
	nugethttp "github.com/willibrandon/gonuget/http"
)

// Test100Runs verifies consistent performance over 100 restore operations
// Goal: Prove gonuget beats dotnet consistently, not just occasionally
func Test100Runs_ConsistentPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 100-run test in short mode")
	}

	// Reset global caches to start fresh
	core.ResetGlobalRepositoryCache()
	nugethttp.ResetGlobalClient()

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

	// Run gonuget 100 times
	t.Log("=== gonuget restore (100 runs) ===")
	var gonugetTimes []time.Duration
	var gonugetFailures int

	for i := range 100 {
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
			gonugetFailures++
			t.Logf("  Run %d: FAILED - %v", i+1, err)
			continue
		}

		gonugetTimes = append(gonugetTimes, elapsed)

		if i < 10 || i >= 90 {
			t.Logf("  Run %d: %v", i+1, elapsed)
		}

		// Brief pause to avoid hammering the server
		if i < 99 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Run dotnet 100 times
	t.Log("\n=== dotnet restore (100 runs) ===")
	var dotnetTimes []time.Duration
	var dotnetFailures int

	for i := range 100 {
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
			dotnetFailures++
			t.Logf("  Run %d: FAILED - %v", i+1, err)
			continue
		}

		dotnetTimes = append(dotnetTimes, elapsed)

		if i < 10 || i >= 90 {
			t.Logf("  Run %d: %v", i+1, elapsed)
		}

		// Brief pause to avoid hammering the server
		if i < 99 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Calculate statistics
	if len(gonugetTimes) == 0 || len(dotnetTimes) == 0 {
		t.Fatal("Not enough successful runs to compare")
	}

	gonugetStats := calculateStats(gonugetTimes)
	dotnetStats := calculateStats(dotnetTimes)

	t.Log("\n=== Results ===")
	t.Logf("gonuget: %d/%d successful (%.1f%% success rate)", len(gonugetTimes), 100, float64(len(gonugetTimes))/100*100)
	t.Logf("  P50: %v", gonugetStats.p50)
	t.Logf("  P95: %v", gonugetStats.p95)
	t.Logf("  P99: %v", gonugetStats.p99)
	t.Logf("  Min: %v", gonugetStats.min)
	t.Logf("  Max: %v", gonugetStats.max)
	t.Logf("  Avg: %v", gonugetStats.avg)

	t.Logf("\ndotnet: %d/%d successful (%.1f%% success rate)", len(dotnetTimes), 100, float64(len(dotnetTimes))/100*100)
	t.Logf("  P50: %v", dotnetStats.p50)
	t.Logf("  P95: %v", dotnetStats.p95)
	t.Logf("  P99: %v", dotnetStats.p99)
	t.Logf("  Min: %v", dotnetStats.min)
	t.Logf("  Max: %v", dotnetStats.max)
	t.Logf("  Avg: %v", dotnetStats.avg)

	t.Log("\n=== Comparison ===")
	p50Ratio := float64(gonugetStats.p50) / float64(dotnetStats.p50)
	p95Ratio := float64(gonugetStats.p95) / float64(dotnetStats.p95)

	if p50Ratio < 1.0 {
		t.Logf("P50: gonuget is %.2fx FASTER", 1.0/p50Ratio)
	} else {
		t.Logf("P50: gonuget is %.2fx slower", p50Ratio)
	}

	if p95Ratio < 1.0 {
		t.Logf("P95: gonuget is %.2fx FASTER", 1.0/p95Ratio)
	} else {
		t.Logf("P95: gonuget is %.2fx slower", p95Ratio)
	}

	// Victory condition: P50 faster than dotnet
	if gonugetStats.p50 < dotnetStats.p50 {
		t.Log("\n🏆 VICTORY: gonuget P50 beats dotnet!")
	} else {
		t.Logf("\n⚠️  gonuget P50 is %v slower than dotnet (target: match or beat)", gonugetStats.p50-dotnetStats.p50)
	}
}

type stats struct {
	p50 time.Duration
	p95 time.Duration
	p99 time.Duration
	min time.Duration
	max time.Duration
	avg time.Duration
}

func calculateStats(times []time.Duration) stats {
	if len(times) == 0 {
		return stats{}
	}

	sorted := make([]time.Duration, len(times))
	copy(sorted, times)
	slices.Sort(sorted)

	var sum time.Duration
	for _, t := range times {
		sum += t
	}

	return stats{
		p50: sorted[len(sorted)*50/100],
		p95: sorted[len(sorted)*95/100],
		p99: sorted[len(sorted)*99/100],
		min: sorted[0],
		max: sorted[len(sorted)-1],
		avg: sum / time.Duration(len(times)),
	}
}
