package main

import (
	"bytes"
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testEnv represents an isolated test environment
type testEnv struct {
	t       *testing.T
	tempDir string
	binPath string
	homeDir string
	oldHome string
}

// newTestEnv creates a new isolated test environment
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Create temp directory
	tempDir := t.TempDir()

	// Build binary
	binPath := filepath.Join(tempDir, "gonuget")

	// Get the repo root (two directories up from cmd/gonuget)
	repoRoot := filepath.Join(getCwd(t), "..", "..")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/gonuget")
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, output)
	}

	// Create fake home directory
	homeDir := filepath.Join(tempDir, "home")
	if err := os.MkdirAll(filepath.Join(homeDir, ".nuget"), 0755); err != nil {
		t.Fatalf("failed to create home directory: %v", err)
	}

	// Save old HOME
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", homeDir)

	return &testEnv{
		t:       t,
		tempDir: tempDir,
		binPath: binPath,
		homeDir: homeDir,
		oldHome: oldHome,
	}
}

// cleanup restores the original environment
func (e *testEnv) cleanup() {
	_ = os.Setenv("HOME", e.oldHome)
}

// run executes the gonuget binary with the given arguments
func (e *testEnv) run(args ...string) (stdout, stderr string, exitCode int) {
	e.t.Helper()

	cmd := exec.Command(e.binPath, args...)
	cmd.Dir = e.tempDir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return
}

// runExpectSuccess runs a command and expects exit code 0
func (e *testEnv) runExpectSuccess(args ...string) (stdout string) {
	e.t.Helper()

	stdout, stderr, exitCode := e.run(args...)
	if exitCode != 0 {
		e.t.Fatalf("command failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	return stdout
}

// runExpectError runs a command and expects non-zero exit code
func (e *testEnv) runExpectError(args ...string) (stderr string) {
	e.t.Helper()

	stdout, stderr, exitCode := e.run(args...)
	if exitCode == 0 {
		e.t.Fatalf("command succeeded but expected failure\nstdout: %s\nstderr: %s",
			stdout, stderr)
	}

	return stderr
}

// configPath returns the path to the user NuGet.config
func (e *testEnv) configPath() string {
	return filepath.Join(e.homeDir, ".nuget", "NuGet", "NuGet.Config")
}

// readConfig reads and parses the NuGet.config file
func (e *testEnv) readConfig() map[string]any {
	e.t.Helper()

	data, err := os.ReadFile(e.configPath())
	if err != nil {
		e.t.Fatalf("failed to read config: %v", err)
	}

	var config struct {
		PackageSources struct {
			Add []struct {
				Key   string `xml:"key,attr"`
				Value string `xml:"value,attr"`
			} `xml:"add"`
		} `xml:"packageSources"`
		DisabledPackageSources struct {
			Add []struct {
				Key   string `xml:"key,attr"`
				Value string `xml:"value,attr"`
			} `xml:"add"`
		} `xml:"disabledPackageSources"`
		Config struct {
			Add []struct {
				Key   string `xml:"key,attr"`
				Value string `xml:"value,attr"`
			} `xml:"add"`
		} `xml:"config"`
	}

	if err := xml.Unmarshal(data, &config); err != nil {
		e.t.Fatalf("failed to parse config: %v", err)
	}

	result := make(map[string]any)
	result["packageSources"] = config.PackageSources.Add
	result["disabledPackageSources"] = config.DisabledPackageSources.Add
	result["config"] = config.Config.Add

	return result
}

// getCwd returns the current working directory
func getCwd(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	return cwd
}

func TestVersionCommand(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	t.Run("version command", func(t *testing.T) {
		stdout := env.runExpectSuccess("version")

		// Should contain version number
		if !strings.Contains(stdout, "gonuget version") {
			t.Errorf("version output should contain 'gonuget version', got: %s", stdout)
		}
	})

	t.Run("version flag", func(t *testing.T) {
		stdout := env.runExpectSuccess("--version")

		// Should contain version number
		if !strings.Contains(stdout, "gonuget version") && !strings.Contains(stdout, "0.0.0-dev") {
			t.Errorf("--version output should contain version, got: %s", stdout)
		}
	})
}

func TestConfigCommand(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create a NuGet.config by adding a source first
	env.runExpectSuccess("add", "source", "https://api.nuget.org/v3/index.json", "--name", "nuget.org")

	t.Run("set and get config value", func(t *testing.T) {
		// Set a value (use valid config key)
		// Matches: dotnet nuget config set repositoryPath ~/test/packages
		env.runExpectSuccess("config", "set", "repositoryPath", "~/test/packages")

		// Get the value
		// Matches: dotnet nuget config get repositoryPath
		stdout := env.runExpectSuccess("config", "get", "repositoryPath")

		if !strings.Contains(stdout, "~/test/packages") {
			t.Errorf("config get should return '~/test/packages', got: %s", stdout)
		}

		// Verify in config file
		config := env.readConfig()
		configItems := config["config"].([]struct {
			Key   string `xml:"key,attr"`
			Value string `xml:"value,attr"`
		})

		found := false
		for _, item := range configItems {
			if item.Key == "repositoryPath" && item.Value == "~/test/packages" {
				found = true
				break
			}
		}

		if !found {
			t.Error("config value not found in NuGet.config")
		}
	})

	t.Run("list all config values", func(t *testing.T) {
		// Set multiple values (use valid config keys)
		env.runExpectSuccess("config", "set", "globalPackagesFolder", "~/.nuget/packages")
		env.runExpectSuccess("config", "set", "http_proxy", "http://proxy.example.com:8080")

		// List all
		// Matches: dotnet nuget config get all
		stdout := env.runExpectSuccess("config", "get", "all")

		if !strings.Contains(stdout, "globalPackagesFolder") || !strings.Contains(stdout, "~/.nuget/packages") {
			t.Errorf("config list should show globalPackagesFolder=~/.nuget/packages, got: %s", stdout)
		}

		if !strings.Contains(stdout, "http_proxy") || !strings.Contains(stdout, "http://proxy.example.com:8080") {
			t.Errorf("config list should show http_proxy=http://proxy.example.com:8080, got: %s", stdout)
		}
	})

	t.Run("explicit config file", func(t *testing.T) {
		customConfig := filepath.Join(env.tempDir, "custom.config")

		// Set value in custom config using --configfile
		// Matches: dotnet nuget config set --configfile custom.config repositoryPath ~/custom/packages
		env.runExpectSuccess("config", "set", "--configfile", customConfig, "repositoryPath", "~/custom/packages")

		// Verify custom config was created
		if _, err := os.Stat(customConfig); os.IsNotExist(err) {
			t.Error("custom config file was not created")
		}

		// Get value from custom config using --working-directory
		// Note: dotnet nuget config get doesn't have --configfile, it uses --working-directory
		// So we verify by checking the file directly
		data, err := os.ReadFile(customConfig)
		if err != nil {
			t.Fatalf("failed to read custom config: %v", err)
		}

		if !strings.Contains(string(data), "repositoryPath") || !strings.Contains(string(data), "~/custom/packages") {
			t.Errorf("custom config should contain repositoryPath=~/custom/packages, got: %s", string(data))
		}
	})
}

func TestSourceCommands(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	t.Run("add source", func(t *testing.T) {
		// Add a source
		// Matches: dotnet nuget add source https://test.example.com/v3/index.json --name TestFeed
		stdout := env.runExpectSuccess("add", "source",
			"https://test.example.com/v3/index.json",
			"--name", "TestFeed")

		if !strings.Contains(stdout, "added successfully") {
			t.Errorf("add source should report success, got: %s", stdout)
		}

		// Verify in config
		config := env.readConfig()
		sources := config["packageSources"].([]struct {
			Key   string `xml:"key,attr"`
			Value string `xml:"value,attr"`
		})

		found := false
		for _, source := range sources {
			if source.Key == "TestFeed" && source.Value == "https://test.example.com/v3/index.json" {
				found = true
				break
			}
		}

		if !found {
			t.Error("source not found in config")
		}
	})

	t.Run("list source", func(t *testing.T) {
		// Add multiple sources
		env.runExpectSuccess("add", "source", "https://feed1.com/v3/index.json", "--name", "Feed1")
		env.runExpectSuccess("add", "source", "https://feed2.com/v3/index.json", "--name", "Feed2")

		// List sources
		// Matches: dotnet nuget list source
		stdout := env.runExpectSuccess("list", "source")

		if !strings.Contains(stdout, "Feed1") {
			t.Errorf("list source should show Feed1, got: %s", stdout)
		}

		if !strings.Contains(stdout, "Feed2") {
			t.Errorf("list source should show Feed2, got: %s", stdout)
		}

		if !strings.Contains(stdout, "https://feed1.com/v3/index.json") {
			t.Errorf("list source should show Feed1 URL, got: %s", stdout)
		}
	})

	t.Run("disable and enable source", func(t *testing.T) {
		// Add a source
		env.runExpectSuccess("add", "source", "https://toggle.com/v3/index.json", "--name", "ToggleFeed")

		// Disable it
		// Matches: dotnet nuget disable source ToggleFeed
		stdout := env.runExpectSuccess("disable", "source", "ToggleFeed")
		if !strings.Contains(stdout, "disabled successfully") {
			t.Errorf("disable should report success, got: %s", stdout)
		}

		// List should show disabled
		stdout = env.runExpectSuccess("list", "source")
		if !strings.Contains(stdout, "Disabled") {
			t.Errorf("list source should show Disabled status, got: %s", stdout)
		}

		// Enable it
		// Matches: dotnet nuget enable source ToggleFeed
		stdout = env.runExpectSuccess("enable", "source", "ToggleFeed")
		if !strings.Contains(stdout, "enabled successfully") {
			t.Errorf("enable should report success, got: %s", stdout)
		}

		// List should show enabled
		stdout = env.runExpectSuccess("list", "source")
		if !strings.Contains(stdout, "Enabled") {
			t.Errorf("list source should show Enabled status, got: %s", stdout)
		}
	})

	t.Run("update source", func(t *testing.T) {
		// Add a source
		env.runExpectSuccess("add", "source", "https://old.com/v3/index.json", "--name", "UpdateFeed")

		// Update it
		// Matches: dotnet nuget update source UpdateFeed --source https://new.com/v3/index.json
		stdout := env.runExpectSuccess("update", "source", "UpdateFeed",
			"--source", "https://new.com/v3/index.json")

		if !strings.Contains(stdout, "updated successfully") {
			t.Errorf("update should report success, got: %s", stdout)
		}

		// Verify new URL
		stdout = env.runExpectSuccess("list", "source")
		if !strings.Contains(stdout, "https://new.com/v3/index.json") {
			t.Errorf("list source should show updated URL, got: %s", stdout)
		}

		if strings.Contains(stdout, "https://old.com/v3/index.json") {
			t.Errorf("list source should not show old URL, got: %s", stdout)
		}
	})

	t.Run("remove source", func(t *testing.T) {
		// Add a source
		env.runExpectSuccess("add", "source", "https://remove.com/v3/index.json", "--name", "RemoveFeed")

		// Remove it
		// Matches: dotnet nuget remove source RemoveFeed
		stdout := env.runExpectSuccess("remove", "source", "RemoveFeed")
		if !strings.Contains(stdout, "removed successfully") {
			t.Errorf("remove should report success, got: %s", stdout)
		}

		// Verify it's gone
		stdout = env.runExpectSuccess("list", "source")
		if strings.Contains(stdout, "RemoveFeed") {
			t.Errorf("list source should not show removed feed, got: %s", stdout)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		// Add duplicate source
		env.runExpectSuccess("add", "source", "https://dup.com/v3/index.json", "--name", "DupFeed")
		stderr := env.runExpectError("add", "source", "https://dup2.com/v3/index.json", "--name", "DupFeed")
		if !strings.Contains(stderr, "already exists") {
			t.Errorf("duplicate source should error, got: %s", stderr)
		}

		// Remove non-existent source
		stderr = env.runExpectError("remove", "source", "NonExistent")
		if !strings.Contains(stderr, "not found") {
			t.Errorf("remove non-existent should error, got: %s", stderr)
		}

		// Enable non-existent source
		stderr = env.runExpectError("enable", "source", "NonExistent")
		if !strings.Contains(stderr, "not found") {
			t.Errorf("enable non-existent should error, got: %s", stderr)
		}
	})
}

func TestHelpCommand(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	t.Run("general help", func(t *testing.T) {
		// Matches: dotnet nuget --help
		stdout := env.runExpectSuccess("--help")

		// Should list usage
		if !strings.Contains(stdout, "Usage:") {
			t.Errorf("help should show usage, got: %s", stdout)
		}

		if !strings.Contains(stdout, "Commands:") {
			t.Errorf("help should list commands, got: %s", stdout)
		}

		if !strings.Contains(stdout, "config") {
			t.Errorf("help should list config command, got: %s", stdout)
		}

		// Should list source commands (add, list, remove, etc.)
		if !strings.Contains(stdout, "add") || !strings.Contains(stdout, "list") {
			t.Errorf("help should list add and list commands, got: %s", stdout)
		}
	})

	t.Run("help flag", func(t *testing.T) {
		stdout := env.runExpectSuccess("--help")

		if !strings.Contains(stdout, "Commands:") {
			t.Errorf("--help should show help, got: %s", stdout)
		}
	})

	t.Run("config command help flag", func(t *testing.T) {
		// Matches: dotnet nuget config --help
		stdout := env.runExpectSuccess("config", "--help")

		if !strings.Contains(stdout, "config") {
			t.Errorf("config --help should show help, got: %s", stdout)
		}
	})
}

func TestEndToEndWorkflow(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// 1. Check version
	// Matches: dotnet nuget --version (or gonuget --version)
	stdout := env.runExpectSuccess("version")
	if !strings.Contains(stdout, "gonuget version") {
		t.Fatal("version command failed")
	}

	// 2. Add multiple package sources (this creates NuGet.config)
	// Matches: dotnet nuget add source <url> --name <name>
	env.runExpectSuccess("add", "source", "https://api.nuget.org/v3/index.json", "--name", "nuget.org")
	env.runExpectSuccess("add", "source", "https://www.myget.org/F/myfeed/api/v3/index.json", "--name", "myget")
	env.runExpectSuccess("add", "source", "/var/packages", "--name", "local")

	// 3. Set configuration
	// Matches: dotnet nuget config set <key> <value>
	env.runExpectSuccess("config", "set", "globalPackagesFolder", "~/.nuget/packages")
	env.runExpectSuccess("config", "set", "http_proxy", "http://proxy.example.com:8080")

	// 4. Disable one source
	// Matches: dotnet nuget disable source <name>
	env.runExpectSuccess("disable", "source", "local")

	// 5. List sources
	// Matches: dotnet nuget list source
	stdout = env.runExpectSuccess("list", "source")
	if !strings.Contains(stdout, "nuget.org") || !strings.Contains(stdout, "myget") {
		t.Fatal("list source failed")
	}

	// 6. Update a source
	// Matches: dotnet nuget update source <name> --source <url>
	env.runExpectSuccess("update", "source", "myget", "--source", "https://www.myget.org/F/newfeed/api/v3/index.json")

	// 7. List config values
	// Matches: dotnet nuget config get all
	stdout = env.runExpectSuccess("config", "get", "all")
	if !strings.Contains(stdout, "globalPackagesFolder") || !strings.Contains(stdout, "http_proxy") {
		t.Fatal("config list failed")
	}

	// 8. Get specific config value
	// Matches: dotnet nuget config get <key>
	stdout = env.runExpectSuccess("config", "get", "globalPackagesFolder")
	if !strings.Contains(stdout, "~/.nuget/packages") {
		t.Fatal("config get failed")
	}

	// 9. Remove a source
	// Matches: dotnet nuget remove source <name>
	env.runExpectSuccess("remove", "source", "local")

	// 10. Verify final state
	stdout = env.runExpectSuccess("list", "source")
	if strings.Contains(stdout, "local") {
		t.Fatal("source removal failed")
	}

	// 11. Check help
	// Matches: dotnet nuget --help
	stdout = env.runExpectSuccess("--help")
	if !strings.Contains(stdout, "Commands:") {
		t.Fatal("help command failed")
	}

	t.Log("✓ End-to-end workflow completed successfully (dotnet nuget parity)")
}

// createBasicProject creates a minimal SDK-style project for testing
func createBasicProject(t *testing.T, path string) {
	t.Helper()
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
}

// packageRef represents a package reference for testing
type packageRef struct {
	ID      string
	Version string
}

// createProjectWithPackages creates a project with specified PackageReferences
func createProjectWithPackages(t *testing.T, path string, packages []packageRef) {
	t.Helper()
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>`

	if len(packages) > 0 {
		content += "\n  <ItemGroup>"
		for _, pkg := range packages {
			content += "\n    <PackageReference Include=\"" + pkg.ID + "\" Version=\"" + pkg.Version + "\" />"
		}
		content += "\n  </ItemGroup>"
	}

	content += "\n</Project>"

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
}
