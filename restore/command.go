// Package restore implements NuGet package restore operations.
// It provides functionality to restore packages from project files,
// resolve dependencies, and manage the package cache.
package restore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/willibrandon/gonuget/cmd/gonuget/project"
)

// Run executes the restore operation (entry point called from CLI).
func Run(ctx context.Context, args []string, opts *Options, console Console) error {
	start := time.Now()
	// Show detailed summary messages for both detailed and diagnostic verbosity
	isDetailed := opts.Verbosity == "detailed" || opts.Verbosity == "diagnostic"
	isQuiet := opts.Verbosity == "quiet" || opts.Verbosity == "q"
	isMinimal := !isQuiet // minimal includes minimal, normal, detailed, diagnostic

	// 1. Find project file
	projectPath, err := findProjectFile(args)
	if err != nil {
		return err
	}

	// Note: indent removed - Terminal Logger doesn't use internal MSBuild message indentation
	_ = isDetailed // Suppress unused warning

	// 2. Load project
	proj, err := project.LoadProject(projectPath)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	// 3. Get package references
	packageRefs := proj.GetPackageReferences()

	// 4. Create restorer (no messages yet - dotnet prints summary first, then details)
	restorer := NewRestorer(opts, console)

	// Diagnostic: Show project analysis
	isDiagnostic := opts.Verbosity == "diagnostic" || opts.Verbosity == "diag"
	if isDiagnostic {
		// Get target frameworks
		var targetFrameworks []string
		if len(proj.TargetFrameworks) > 0 {
			targetFrameworks = proj.TargetFrameworks
		} else if proj.TargetFramework != "" {
			targetFrameworks = []string{proj.TargetFramework}
		}

		// Get SDK (if available)
		sdk := ""
		if proj.Root != nil {
			sdk = proj.Root.Sdk
		}

		restorer.tracer.TraceProjectAnalysis(proj.Path, sdk, targetFrameworks, len(packageRefs))

		// Show package sources
		if len(opts.Sources) > 0 {
			restorer.tracer.TracePackageSources(opts.Sources)
		}
	}

	if len(packageRefs) == 0 {
		if !isQuiet {
			console.Printf("Nothing to restore\n")
		}
		return nil
	}

	// 5. Execute restore (track separate timing for per-project message)
	// Start terminal status updater (30Hz right-aligned status)
	projectName := filepath.Base(proj.Path)
	termStatus := NewTerminalStatus(console.Output(), projectName, nil)
	defer termStatus.Stop()

	restoreStart := time.Now()
	result, err := restorer.Restore(ctx, proj, packageRefs)
	restoreElapsed := time.Since(restoreStart)

	// Stop terminal status before printing results
	termStatus.Stop()
	if err != nil {
		// Print NuGet errors in correct format (if any)
		// DON'T print "Determining projects to restore..." on error path (matches dotnet)
		if result != nil && len(result.Errors) > 0 {
			// Detect TTY for colorization (dotnet doesn't colorize when piped)
			isTTY := termStatus.IsTTY()

			for _, nugetErr := range result.Errors {
				// NU1102 and NU1103 require multi-line format with per-source version info
				if nugetErr.Code == ErrorCodePackageVersionNotFound || nugetErr.Code == ErrorCodePackageDownloadFailed {
					// Format version-not-found errors (NU1102/NU1103)
					// Dotnet always uses prefix format (each line with full path)
					errorMsg := FormatVersionNotFoundError(
						nugetErr.ProjectPath,
						nugetErr.PackageID,
						nugetErr.Constraint,
						nugetErr.VersionInfos,
						nugetErr.Code,
						isTTY, // Colorize only for TTY output
					)
					console.Printf("%s\n", errorMsg)
				} else {
					// Use single-line format for other errors (NU1101)
					errorMsg := nugetErr.FormatError(isTTY) // Colorize only for TTY output
					// In quiet mode, remove indent from error messages (dotnet doesn't indent in quiet mode)
					if isQuiet {
						errorMsg = strings.TrimPrefix(errorMsg, "    ")
					}
					console.Printf("%s\n", errorMsg)
				}
			}

			// In non-quiet mode, print "Restore failed" summary (dotnet doesn't show this in quiet mode)
			if !isQuiet {
				elapsed := time.Since(start)
				errorCount := len(result.Errors)

				// Add blank line before summary (dotnet has spacing)
				console.Printf("\n")

				// Format: "Restore failed with N error(s) in X.Xs" with red on "failed with N error(s)"
				// Colorize only for TTY output (dotnet doesn't colorize when piped)
				if isTTY {
					// ANSI color codes (use bright red like error codes)
					const (
						red   = "\033[1;31m"
						reset = "\033[0m"
					)
					console.Printf("Restore %sfailed with %d error(s)%s in %.1fs\n",
						red, errorCount, reset, elapsed.Seconds())
				} else {
					// Plain text for piped output
					console.Printf("Restore failed with %d error(s) in %.1fs\n",
						errorCount, elapsed.Seconds())
				}
			}

			// Return a clean error without wrapping (main.go will add "Error: " prefix)
			return fmt.Errorf("")
		}
		return err
	}

	// 6. Generate lock file (project.assets.json) - only if not cache hit
	// Note: Terminal Logger hides all MSBuild internal messages (dg file, MSBuild files, assets, cache, etc.)
	// We match Terminal Logger behavior: clean output, no internal spam
	var assetsInfo *AssetsInfo
	if !result.CacheHit {
		lockFile := NewLockFileBuilder().Build(proj, result)
		objDir := filepath.Join(filepath.Dir(proj.Path), "obj")
		assetsPath := filepath.Join(objDir, "project.assets.json")

		if err := lockFile.Save(assetsPath); err != nil {
			return fmt.Errorf("failed to save project.assets.json: %w", err)
		}

		// Diagnostic: Collect assets information
		if isDiagnostic {
			assetsInfo = &AssetsInfo{
				ProjectAssetsFile: assetsPath,
				PackageCount:      len(result.DirectPackages) + len(result.TransitivePackages),
				TargetFrameworks:  proj.GetTargetFrameworks(),
			}

			// Get file size
			if fileInfo, err := os.Stat(assetsPath); err == nil {
				assetsInfo.ProjectAssetsSize = fileInfo.Size()
			}

			// Get cache file info
			cachePath := GetCacheFilePath(proj.Path)
			if fileInfo, err := os.Stat(cachePath); err == nil {
				assetsInfo.CacheFile = cachePath
				assetsInfo.CacheFileSize = fileInfo.Size()

				// Read cache file to get dgspec hash
				if cache, err := LoadCacheFile(cachePath); err == nil {
					assetsInfo.DgSpecHash = cache.DgSpecHash
				}
			}
		}
	}

	// 7. Report summary (matches MSBuild Terminal Logger format)
	elapsed := time.Since(start)

	// Diagnostic: Show assets generation
	if isDiagnostic && assetsInfo != nil {
		restorer.tracer.TraceAssetsGeneration(assetsInfo)
	}

	// Diagnostic: Show performance breakdown
	if isDiagnostic && result != nil && result.PerformanceTiming != nil {
		restorer.tracer.TracePerformanceBreakdown(result.PerformanceTiming)
	}

	// Quiet mode: No output on success
	if isQuiet {
		return nil
	}

	// Detect if output is TTY or piped (Console Logger vs Terminal Logger)
	isTTY := termStatus.IsTTY()

	// Terminal Logger (TTY) - clean output for interactive terminals
	if isTTY {
		// Print "Restore complete" summary first (matches dotnet Terminal Logger)
		console.Printf("Restore complete (%.1fs)\n", elapsed.Seconds())

		// Detailed mode: Print breakdown of what happened (indented with 4 spaces)
		// Skip in diagnostic mode - we already have comprehensive diagnostic output
		if isDetailed && !isDiagnostic {
			console.Printf("    Determining projects to restore...\n")
			// Terminal Logger: Show "All projects are up-to-date" for cache hits
			// Show "Restored /path (in X ms)" only for actual restores
			if result.CacheHit {
				console.Printf("    All projects are up-to-date for restore.\n")
			} else {
				console.Printf("    Restored %s (in %d ms).\n", proj.Path, restoreElapsed.Milliseconds())
			}
		}

		// Add blank line and success message (matches dotnet's "Build succeeded" but says "Restore succeeded")
		// ANSI green color for "succeeded" (color 32 then ;1 for bright to match MSBuild exactly)
		const (
			green = "\033[32;1m"
			reset = "\033[0m"
		)
		console.Printf("\nRestore %ssucceeded%s in %.1fs\n", green, reset, elapsed.Seconds())
	} else {
		// Console Logger (piped) - matches dotnet when output is redirected

		// Minimal mode: Show basic restore status (matches dotnet minimal verbosity)
		if isMinimal && !isDetailed && !isDiagnostic {
			console.Printf("  Determining projects to restore...\n")
			if result.CacheHit {
				console.Printf("  All projects are up-to-date for restore.\n")
			} else {
				console.Printf("  Restored %s (in %d ms).\n", proj.Path, restoreElapsed.Milliseconds())
			}
		}

		// Detailed mode: Show verbose restore details (matches dotnet Console Logger detailed verbosity)
		if isDetailed && !isDiagnostic {
			// Show "Committing restore..." at LogVerbose level (detailed only)
			console.Printf("  Committing restore...\n")

			// Show file write operations or cache status
			objDir := filepath.Join(filepath.Dir(proj.Path), "obj")
			assetsPath := filepath.Join(objDir, "project.assets.json")
			cachePath := GetCacheFilePath(proj.Path)

			if !result.CacheHit {
				// Files were written - show write messages
				dgSpecPath := filepath.Join(objDir, filepath.Base(proj.Path)+".nuget.dgspec.json")
				console.Printf("  Writing assets file to disk. Path: %s\n", assetsPath)
				console.Printf("  Writing cache file to disk. Path: %s\n", cachePath)
				console.Printf("  Persisting dg to %s\n", dgSpecPath)
			} else {
				// Cache hit - show that assets file and cache were not updated
				console.Printf("  Assets file has not changed. Skipping assets file writing. Path: %s\n", assetsPath)
				console.Printf("  No-Op restore. The cache will not be updated. Path: %s\n", cachePath)
			}

			// Always show "Restored /path (in X ms)." for successful restores
			// For cache hits: logged at LogLevel.Information (normal/detailed/diagnostic)
			// For actual restores: logged at LogLevel.Minimal (minimal/normal/detailed/diagnostic)
			console.Printf("  Restored %s (in %d ms).\n", proj.Path, restoreElapsed.Milliseconds())

			console.Printf("\n")

			// Show NuGet config files used
			console.Printf("  NuGet Config files used:\n")
			// Get user config path
			if home, err := os.UserHomeDir(); err == nil {
				userConfigPath := filepath.Join(home, ".nuget", "NuGet", "NuGet.Config")
				if _, err := os.Stat(userConfigPath); err == nil {
					console.Printf("      %s\n", userConfigPath)
				}
			}

			console.Printf("\n")

			// Show feeds used
			console.Printf("  Feeds used:\n")
			if len(opts.Sources) > 0 {
				for _, source := range opts.Sources {
					console.Printf("      %s\n", source)
				}
			}

			// Show "All projects are up-to-date" only for cache hits (no-op restores)
			// This matches NuGet.Client's RestoreSummary.cs behavior
			if result.CacheHit {
				console.Printf("  All projects are up-to-date for restore.\n")
			}
		}

		// Diagnostic mode: Always show completion status (after all diagnostic output)
		if isDiagnostic {
			if result.CacheHit {
				console.Printf("  All projects are up-to-date for restore.\n")
			} else {
				console.Printf("  Restored %s (in %d ms).\n", proj.Path, restoreElapsed.Milliseconds())
			}
		}
	}

	return nil
}

func findProjectFile(args []string) (string, error) {
	var projectPath string
	var err error

	if len(args) > 0 {
		projectPath = args[0]
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		projectPath, err = project.FindProjectFile(cwd)
		if err != nil {
			return "", err
		}
	}

	// Convert to absolute path WITHOUT resolving symlinks (matches dotnet behavior)
	// On macOS, /tmp is a symlink to /private/tmp, but dotnet preserves /tmp in output
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", err
	}

	return absPath, nil
}
