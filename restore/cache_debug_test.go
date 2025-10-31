package restore

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// TestCacheDebug helps debug why cache doesn't hit in CI
func TestCacheDebug(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping debug test in short mode")
	}

	cwd, _ := os.Getwd()
	projectPath := filepath.Join(filepath.Dir(cwd), "tests", "test-scenarios", "complex", "test.csproj")
	cachePath := filepath.Join(filepath.Dir(cwd), "tests", "test-scenarios", "complex", "obj", "project.nuget.cache")

	// Step 1: Run dotnet restore
	dotnetCmd := exec.Command("dotnet", "restore", projectPath, "-v:quiet")
	if err := dotnetCmd.Run(); err != nil {
		t.Fatalf("dotnet restore failed: %v", err)
	}

	// Step 2: Read cache file to see dotnet's hash
	cacheData, _ := os.ReadFile(cachePath)
	var dotnetCache struct {
		DgSpecHash string `json:"dgSpecHash"`
	}
	json.Unmarshal(cacheData, &dotnetCache)
	t.Logf("Dotnet hash: %s", dotnetCache.DgSpecHash)

	// Step 3: Calculate gonuget hash
	proj, _ := project.LoadProject(projectPath)
	cfg, _ := DiscoverDgSpecConfig(proj)

	t.Logf("Config paths: %v", cfg.ConfigPaths)
	t.Logf("Sources: %v", cfg.Sources)
	t.Logf("Fallback folders: %v", cfg.FallbackFolders)
	t.Logf("RuntimeID path: %s", cfg.RuntimeIDPath)
	t.Logf("SdkAnalysisLevel: %s", cfg.SdkAnalysisLevel)

	gonugetHash, _ := CalculateDgSpecHash(proj)
	t.Logf("Gonuget hash: %s", gonugetHash)

	if gonugetHash != dotnetCache.DgSpecHash {
		t.Errorf("Hash mismatch! Dotnet: %s, Gonuget: %s", dotnetCache.DgSpecHash, gonugetHash)
	}
}
