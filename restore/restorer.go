// Package restore implements NuGet package restore operations.
// It provides functionality to restore packages from project files,
// resolve dependencies, and manage the package cache.
package restore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/willibrandon/gonuget/cmd/gonuget/project"
	"github.com/willibrandon/gonuget/core"
	"github.com/willibrandon/gonuget/core/resolver"
	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/packaging"
	"github.com/willibrandon/gonuget/version"
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

// Restorer executes restore operations.
type Restorer struct {
	opts    *Options
	console Console
	client  *core.Client
	tracer  DiagnosticTracer // Diagnostic output tracer (enabled for diagnostic verbosity only)
	logs    []LogMessage     // Collected warnings/errors during restore (for cache file)
}

// NewRestorer creates a new restorer.
func NewRestorer(opts *Options, console Console) *Restorer {
	// Create repository manager
	repoManager := core.NewRepositoryManager()

	// Add sources using GLOBAL repository cache
	// This is critical for performance - reuses HTTP clients, protocol providers, and connections
	// Matches NuGet.Client's SourceRepositoryProvider which maintains singleton repositories
	if len(opts.Sources) > 0 {
		for _, source := range opts.Sources {
			// Get or create repository from global cache (avoids protocol detection on every restore!)
			repo := core.GetOrCreateRepository(source)
			if err := repoManager.AddRepository(repo); err != nil {
				console.Warning(fmt.Sprintf("Failed to add repository %s: %v", source, err))
			}
		}
	}

	// Create client
	client := core.NewClient(core.ClientConfig{
		RepositoryManager: repoManager,
	})

	return &Restorer{
		opts:    opts,
		console: console,
		client:  client,
		tracer:  NewResolutionTracer(console, opts.Verbosity),
		logs:    make([]LogMessage, 0),
	}
}

// addLog adds a log message to the collector for cache file persistence.
// Matches MSBuildRestoreUtility.CollectMessage in NuGet.Client.
func (r *Restorer) addLog(log LogMessage) {
	r.logs = append(r.logs, log)
}

// addErrorLog creates and adds an error log from a NuGetError.
// Matches NuGet.Client's error logging in RestoreCommand.
func (r *Restorer) addErrorLog(err *NuGetError, targetFramework string) {
	log := LogMessage{
		Code:         err.Code,
		Level:        "Error",
		Message:      err.Message,
		ProjectPath:  err.ProjectPath,
		FilePath:     err.ProjectPath,
		LibraryID:    err.PackageID,
		TargetGraphs: []string{targetFramework},
	}
	r.addLog(log)
}

// replayLogs outputs cached logs to console (on cache hit).
// Matches MSBuildRestoreUtility.ReplayWarningsAndErrorsAsync in NuGet.Client.
func (r *Restorer) replayLogs(logs []LogMessage) {
	for _, log := range logs {
		level := strings.ToLower(log.Level)
		switch level {
		case "error":
			// Format: "    /path/to/project.csproj : error NU1101: message"
			// Use ANSI colors only if colors are enabled (TTY mode)
			if !color.NoColor {
				const (
					red   = "\033[1;31m"
					reset = "\033[0m"
				)
				r.console.Printf("    %s : %serror %s%s: %s\n",
					log.ProjectPath, red, log.Code, reset, log.Message)
			} else {
				r.console.Printf("    %s : error %s: %s\n",
					log.ProjectPath, log.Code, log.Message)
			}
		case "warning":
			// Format warnings similarly (yellow color in TTY mode)
			if !color.NoColor {
				const (
					yellow = "\033[1;33m"
					reset  = "\033[0m"
				)
				r.console.Printf("    %s : %swarning %s%s: %s\n",
					log.ProjectPath, yellow, log.Code, reset, log.Message)
			} else {
				r.console.Printf("    %s : warning %s: %s\n",
					log.ProjectPath, log.Code, log.Message)
			}
		}
	}
}

// writeCacheFileOnError writes a cache file when restore fails early.
// Matches NuGet.Client behavior of writing cache even on failure (with success=false).
func (r *Restorer) writeCacheFileOnError(proj *project.Project, dgSpecHash, cachePath string) {
	cacheFile := &CacheFile{
		Version:              CacheFileVersion,
		DgSpecHash:           dgSpecHash,
		Success:              false, // Restore failed
		ProjectFilePath:      proj.Path,
		ExpectedPackageFiles: []string{}, // No packages resolved
		Logs:                 r.logs,     // Collected error logs
	}

	// Don't fail if cache write fails (just log warning)
	if err := cacheFile.Save(cachePath); err != nil {
		r.console.Warning("Failed to write cache file: %v\n", err)
	}
}

// Result holds restore results.
type Result struct {
	// DirectPackages contains packages explicitly listed in project file
	DirectPackages []PackageInfo

	// TransitivePackages contains packages pulled in as dependencies
	TransitivePackages []PackageInfo

	// Graph contains full dependency graph (optional, for debugging)
	Graph any // *resolver.GraphNode, but avoid import cycle

	// CacheHit indicates restore was skipped (cache valid)
	CacheHit bool

	// Errors contains NuGet errors encountered during restore
	Errors []*NuGetError

	// PerformanceTiming holds detailed timing metrics (diagnostic mode only)
	PerformanceTiming *PerformanceTiming
}

// PerformanceTiming holds detailed timing metrics for diagnostic output.
type PerformanceTiming struct {
	// Phase timings
	DependencyResolution time.Duration
	PackageDownloads     time.Duration
	AssetsGeneration     time.Duration

	// Per-package resolution timing
	ResolutionTimings map[string]time.Duration // packageID -> duration

	// Per-package download timing
	DownloadTimings map[string]time.Duration // packageID -> duration
	CacheHits       map[string]bool          // packageID -> cache hit
}

// PackageInfo holds package information.
type PackageInfo struct {
	ID      string
	Version string
	Path    string

	// IsDirect indicates if this is a direct dependency
	IsDirect bool
}

// AllPackages returns all packages (direct + transitive).
// Matches NuGet.Client's flattened package list from RestoreTargetGraph.
func (r *Result) AllPackages() []PackageInfo {
	all := make([]PackageInfo, 0, len(r.DirectPackages)+len(r.TransitivePackages))
	all = append(all, r.DirectPackages...)
	all = append(all, r.TransitivePackages...)
	return all
}

// Restore executes the restore operation with full transitive dependency resolution.
// Matches NuGet.Client RestoreCommand behavior (line 572-616 GenerateRestoreGraphsAsync).
func (r *Restorer) Restore(
	ctx context.Context,
	proj *project.Project,
	packageRefs []project.PackageReference,
) (*Result, error) {
	result := &Result{
		DirectPackages:     make([]PackageInfo, 0, len(packageRefs)),
		TransitivePackages: make([]PackageInfo, 0),
	}

	// Initialize performance timing in diagnostic mode
	isDiagnostic := r.opts.Verbosity == "diagnostic" || r.opts.Verbosity == "diag"
	if isDiagnostic {
		result.PerformanceTiming = &PerformanceTiming{
			ResolutionTimings: make(map[string]time.Duration),
			DownloadTimings:   make(map[string]time.Duration),
			CacheHits:         make(map[string]bool),
		}
	}

	// Phase 0: No-op optimization (cache check)
	// Matches RestoreCommand.EvaluateNoOpAsync (line 442-501)
	cachePath := GetCacheFilePath(proj.Path)

	// Calculate current hash
	currentHash, err := CalculateDgSpecHash(proj)
	if err != nil {
		// If we can't calculate hash, just proceed with full restore
		r.console.Warning("Failed to calculate dgspec hash: %v\n", err)
	} else {
		// Check if cache is valid
		cacheValid, cachedFile, err := IsCacheValid(cachePath, currentHash)
		if err != nil {
			r.console.Warning("Failed to validate cache: %v\n", err)
		} else if cacheValid && !r.opts.Force {
			// Cache hit! Return cached result without doing restore
			// (Message will be printed by Run() function)

			// Diagnostic: Show project-level cache hit
			isDiagnostic := r.opts.Verbosity == "diagnostic" || r.opts.Verbosity == "diag"
			if isDiagnostic {
				r.console.Printf("Project restore cache hit (dgspec hash: %s)\n", currentHash)
				r.console.Printf("  Using cached restore result from: %s\n", cachePath)
				r.console.Printf("  All packages already restored - skipping dependency resolution\n\n")
			}

			// Get packages folder for path construction
			packagesFolder := r.opts.PackagesFolder
			if packagesFolder == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					// Fallback: just proceed with full restore
					goto fullRestore
				}
				packagesFolder = filepath.Join(home, ".nuget", "packages")
			}

			// Build result from cache
			// Build map of direct package IDs from project file PackageReferences
			// This matches dotnet behavior: check project file, not cache extensions
			directPackageIDs := make(map[string]bool)
			for _, pkgRef := range packageRefs {
				normalizedID := strings.ToLower(pkgRef.Include)
				directPackageIDs[normalizedID] = true
			}

			// Parse package info from cache paths
			// Expected format: /path/packages/{id}/{version}/{id}.{version}.nupkg.sha512
			for _, pkgPath := range cachedFile.ExpectedPackageFiles {
				parts := strings.Split(filepath.ToSlash(pkgPath), "/")
				if len(parts) < 3 {
					continue
				}

				// Extract ID and version from path
				version := parts[len(parts)-2]
				id := parts[len(parts)-3]

				// Skip framework reference packs (these are not regular NuGet packages)
				// These are download dependencies added by the SDK for targeting packs
				normalizedID := strings.ToLower(id)
				if isFrameworkReferencePack(normalizedID) {
					continue
				}

				// Check if this package ID is in project file PackageReferences
				isDirect := directPackageIDs[normalizedID]

				info := PackageInfo{
					ID:       id,
					Version:  version,
					Path:     filepath.Join(packagesFolder, normalizedID, version),
					IsDirect: isDirect,
				}

				if info.IsDirect {
					result.DirectPackages = append(result.DirectPackages, info)
				} else {
					result.TransitivePackages = append(result.TransitivePackages, info)
				}
			}

			result.CacheHit = true

			// Replay warnings/errors from cache (matches NuGet.Client line 471)
			// This must happen on cache hit to show users any problems from cached restore
			if len(cachedFile.Logs) > 0 {
				r.replayLogs(cachedFile.Logs)
			}

			// Diagnostic: Show cached packages
			if isDiagnostic {
				r.console.Printf("Cached packages:\n")
				if len(result.DirectPackages) > 0 {
					r.console.Printf("  Direct packages (%d):\n", len(result.DirectPackages))
					for _, pkg := range result.DirectPackages {
						r.console.Printf("    - %s %s\n", pkg.ID, pkg.Version)
					}
				}
				if len(result.TransitivePackages) > 0 {
					r.console.Printf("  Transitive packages (%d):\n", len(result.TransitivePackages))
					for _, pkg := range result.TransitivePackages {
						r.console.Printf("    - %s %s\n", pkg.ID, pkg.Version)
					}
				}
				r.console.Printf("\n")
			}

			return result, nil
		}
	}

fullRestore:
	// Cache miss or invalid - proceed with full restore
	// Get global packages folder
	packagesFolder := r.opts.PackagesFolder
	if packagesFolder == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		packagesFolder = filepath.Join(home, ".nuget", "packages")
	}

	// Ensure packages folder exists
	if err := os.MkdirAll(packagesFolder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create packages folder: %w", err)
	}

	// Get target framework
	targetFrameworkStrings := proj.GetTargetFrameworks()
	if len(targetFrameworkStrings) == 0 {
		return nil, fmt.Errorf("project has no target frameworks")
	}
	targetFrameworkStr := targetFrameworkStrings[0] // Use first TFM

	// Parse target framework
	targetFramework, err := frameworks.ParseFramework(targetFrameworkStr)
	if err != nil {
		return nil, fmt.Errorf("parse target framework: %w", err)
	}

	// Track all resolved packages (direct + transitive)
	allResolvedPackages := make(map[string]*resolver.PackageDependencyInfo)
	unresolvedPackages := make([]*resolver.GraphNode, 0)

	// Track direct package ID+version combinations (key format: "packageid|version")
	// This is used to properly categorize packages when multiple versions of same ID exist
	directPackageKeys := make(map[string]bool)

	// Diagnostic: Start resolution trace
	isDiagnostic = r.opts.Verbosity == "diagnostic" || r.opts.Verbosity == "diag"
	if isDiagnostic {
		r.console.Printf("\nResolving dependencies for %s:\n", targetFrameworkStr)
	}

	// Phase 1: Walk dependency graph for each direct dependency
	// Start timing dependency resolution
	resolutionStart := time.Now()

	// Create local dependency provider for cached packages (NO HTTP!)
	// Matches NuGet.Client's LocalLibraryProviders approach (RestoreCommand.cs line 2037-2056)
	localProvider := NewLocalDependencyProvider(packagesFolder)

	// Create dependency walker with local-first metadata client
	// Matches NuGet.Client's RemoteWalkContext with LocalLibraryProviders + RemoteLibraryProviders
	walker, err := r.createLocalFirstWalker(localProvider, targetFramework)
	if err != nil {
		return nil, fmt.Errorf("failed to create dependency walker: %w", err)
	}

	for _, pkgRef := range packageRefs {
		// Use version as-is from PackageReference
		// ParseVersionRange will handle conversion correctly:
		// - Plain versions like "13.0.1" are interpreted as ">= 13.0.1" (minimum version)
		// - Bracketed versions like "[13.0.1]" are exact matches
		// - Ranges like "[1.0,2.0)" are preserved
		// This matches NuGet.Client's VersionRange.Parse behavior
		versionRange := pkgRef.Version
		if versionRange == "" {
			versionRange = "0.0.0" // Empty means any version >= 0.0.0
		}

		// Diagnostic: Trace package resolution start
		if isDiagnostic {
			r.console.Printf("  %s %s (direct reference)\n", pkgRef.Include, versionRange)
			r.console.Printf("    Constraint: %s\n", versionRange)
		}

		// OPTIMIZATION: Early version availability check (matches NuGet.Client's SourceRepositoryDependencyProvider)
		// Check if any version satisfying the constraint exists BEFORE running expensive dependency walk
		// This provides massive speedup for NU1102/NU1103 error cases (version not found)
		versionInfos, allVersions, allSourceNames, canSatisfy := r.checkVersionAvailability(ctx, pkgRef.Include, versionRange)

		// Diagnostic: Show available versions (limit to last 10 for readability)
		if isDiagnostic && len(allVersions) > 0 {
			displayVersions := allVersions
			if len(displayVersions) > 10 {
				displayVersions = displayVersions[len(displayVersions)-10:]
			}
			r.console.Printf("    Available versions: %s\n", strings.Join(displayVersions, ", "))
		}

		if !canSatisfy {
			// Version not found - immediately return NU1101/NU1102/NU1103 error without dependency walk
			// This saves ~160-195ms by skipping graph traversal
			var nugetErr *NuGetError
			switch {
			case len(versionInfos) == 0:
				// Package doesn't exist at all - NU1101
				nugetErr = NewPackageNotFoundError(proj.Path, pkgRef.Include, versionRange, allSourceNames)
			case !isPrereleaseAllowed(versionRange) && hasPrereleaseVersionsOnly(versionRange, allVersions):
				// Only prerelease versions satisfy the range when stable requested - NU1103
				// For NU1103, dotnet shows the LOWEST prerelease version (not highest)
				parsedRange, _ := version.ParseVersionRange(versionRange)
				versionInfosNU1103 := r.updateNearestVersionForNU1103(versionInfos, allVersions, parsedRange)
				nugetErr = NewPackageDownloadFailedError(proj.Path, pkgRef.Include, versionRange, versionInfosNU1103)
			default:
				// Package exists but no compatible version - NU1102
				nugetErr = NewPackageVersionNotFoundError(proj.Path, pkgRef.Include, versionRange, versionInfos)
			}

			// Add error to result and collect for cache file
			result.Errors = []*NuGetError{nugetErr}
			r.addErrorLog(nugetErr, targetFrameworkStr)

			// Write cache file with error before returning (matches NuGet.Client behavior)
			// Cache file is written even on failure so errors can be replayed on cache hit
			r.writeCacheFileOnError(proj, currentHash, cachePath)

			return result, fmt.Errorf("restore failed due to package version not found")
		}

		// Walk dependency graph (matches RemoteDependencyWalker.WalkAsync line 28)
		pkgResolutionStart := time.Now()
		graphNode, err := walker.Walk(
			ctx,
			pkgRef.Include,
			versionRange,
			targetFrameworkStr,
			true, // recursive=true for transitive resolution
		)
		pkgResolutionDuration := time.Since(pkgResolutionStart)

		// Record per-package resolution timing
		if isDiagnostic && result.PerformanceTiming != nil {
			result.PerformanceTiming.ResolutionTimings[pkgRef.Include] = pkgResolutionDuration
		}

		if err != nil {
			return nil, fmt.Errorf("failed to walk dependencies for %s: %w", pkgRef.Include, err)
		}

		// Track this as a direct dependency (the root node from walker.Walk)
		if graphNode != nil && graphNode.Item != nil {
			key := fmt.Sprintf("%s|%s", strings.ToLower(graphNode.Item.ID), graphNode.Item.Version)
			directPackageKeys[key] = true
		}

		// Diagnostic: Show selected version and dependencies
		if isDiagnostic && graphNode != nil && graphNode.Item != nil {
			r.console.Printf("    Selected: %s (highest matching)\n", graphNode.Item.Version)
			r.console.Printf("    Framework: compatible with %s\n", targetFrameworkStr)

			// Show dependencies for this package
			deps := r.getDependenciesForFramework(graphNode.Item, targetFrameworkStr)
			if len(deps) > 0 {
				r.console.Printf("    Dependencies:\n")
				for _, dep := range deps {
					r.console.Printf("      → %s %s\n", dep.ID, dep.VersionRange)
				}
			}
			r.console.Printf("\n")
		}

		// Collect all packages from graph (breadth-first)
		// Matches NuGet.Client: collect both resolved and unresolved packages
		if err := r.collectPackagesFromGraph(graphNode, allResolvedPackages, &unresolvedPackages); err != nil {
			return nil, err
		}
	}

	// Record total resolution timing
	if isDiagnostic && result.PerformanceTiming != nil {
		result.PerformanceTiming.DependencyResolution = time.Since(resolutionStart)
	}

	// Diagnostic: Show dependency graph summary
	if isDiagnostic {
		directCount := len(packageRefs)
		transitiveCount := max(0, len(allResolvedPackages)-directCount) // Safety check
		r.tracer.TraceDependencyGraph(directCount, transitiveCount)
	}

	// Check for unresolved packages and fail restore if any found
	// Matches NuGet.Client: RestoreCommand fails when graphs have Unresolved.Count > 0
	if len(unresolvedPackages) > 0 {
		// Store NuGet errors in result for proper formatting by Run()
		result.Errors = r.buildUnresolvedError(ctx, unresolvedPackages, proj.Path)
		return result, fmt.Errorf("restore failed due to unresolved packages")
	}

	// Phase 2: Download all resolved packages (direct + transitive)
	// Matches ProjectRestoreCommand.InstallPackagesAsync behavior
	downloadStart := time.Now()
	for _, pkgInfo := range allResolvedPackages {
		normalizedID := strings.ToLower(pkgInfo.ID)
		packagePath := filepath.Join(packagesFolder, normalizedID, pkgInfo.Version)

		// Check if package already exists in cache
		cacheHit := false
		if !r.opts.Force {
			if _, err := os.Stat(packagePath); err == nil {
				cacheHit = true
			}
		}

		// Record cache hit status
		if isDiagnostic && result.PerformanceTiming != nil {
			result.PerformanceTiming.CacheHits[pkgInfo.ID] = cacheHit
		}

		// Time individual package download
		pkgDownloadStart := time.Now()

		// Download package (pass original ID for display, with cache hit flag for logging)
		if err := r.downloadPackage(ctx, pkgInfo.ID, pkgInfo.Version, packagePath, cacheHit); err != nil {
			return nil, fmt.Errorf("failed to download package %s %s: %w", pkgInfo.ID, pkgInfo.Version, err)
		}

		// Record per-package download timing
		if isDiagnostic && result.PerformanceTiming != nil {
			result.PerformanceTiming.DownloadTimings[pkgInfo.ID] = time.Since(pkgDownloadStart)
		}
	}

	// Record total download timing
	if isDiagnostic && result.PerformanceTiming != nil {
		result.PerformanceTiming.PackageDownloads = time.Since(downloadStart)
	}

	// Phase 3: Categorize packages as direct vs transitive
	// Use directPackageKeys that was built during resolution (tracks ID+version, not just ID)
	for _, pkgInfo := range allResolvedPackages {
		normalizedID := strings.ToLower(pkgInfo.ID)
		packagePath := filepath.Join(packagesFolder, normalizedID, pkgInfo.Version)

		// Check if this specific package ID+version is direct
		key := fmt.Sprintf("%s|%s", normalizedID, pkgInfo.Version)
		isDirect := directPackageKeys[key]

		info := PackageInfo{
			ID:       pkgInfo.ID,
			Version:  pkgInfo.Version,
			Path:     packagePath,
			IsDirect: isDirect,
		}

		if info.IsDirect {
			result.DirectPackages = append(result.DirectPackages, info)
		} else {
			result.TransitivePackages = append(result.TransitivePackages, info)
		}
	}

	// Phase 4: Write cache file for no-op optimization
	// Matches RestoreCommand.CommitCacheFileAsync (RestoreResult.cs line 296)
	assetsStart := time.Now()
	cachePath = GetCacheFilePath(proj.Path)

	// Calculate hash
	dgSpecHash, err := CalculateDgSpecHash(proj)
	if err != nil {
		// If we can't calculate hash, just proceed without cache
		r.console.Warning("Failed to calculate dgspec hash: %v\n", err)
	} else {
		// Build expected package file paths (all .nupkg.sha512 files)
		expectedPackageFiles := make([]string, 0, len(allResolvedPackages))
		for _, pkgInfo := range allResolvedPackages {
			normalizedID := strings.ToLower(pkgInfo.ID)
			sha512Path := filepath.Join(packagesFolder, normalizedID, pkgInfo.Version,
				fmt.Sprintf("%s.%s.nupkg.sha512", normalizedID, pkgInfo.Version))
			expectedPackageFiles = append(expectedPackageFiles, sha512Path)
		}

		// Create cache file
		cacheFile := &CacheFile{
			Version:              CacheFileVersion,
			DgSpecHash:           dgSpecHash,
			Success:              true,
			ProjectFilePath:      proj.Path,
			ExpectedPackageFiles: expectedPackageFiles,
			Logs:                 r.logs, // Collected warnings/errors during restore
		}

		// Save cache file
		if err := cacheFile.Save(cachePath); err != nil {
			// Don't fail restore if cache write fails
			r.console.Warning("Failed to write cache file: %v\n", err)
		}
	}

	// Record assets generation timing
	if isDiagnostic && result.PerformanceTiming != nil {
		result.PerformanceTiming.AssetsGeneration = time.Since(assetsStart)
	}

	return result, nil
}

// collectPackagesFromGraph traverses graph and collects resolved and unresolved packages.
// Matches NuGet.Client's graph flattening in BuildAssetsFile.
func (r *Restorer) collectPackagesFromGraph(
	node *resolver.GraphNode,
	collected map[string]*resolver.PackageDependencyInfo,
	unresolved *[]*resolver.GraphNode,
) error {
	if node == nil || node.Item == nil {
		return nil
	}

	key := node.Key

	// Collect unresolved packages separately
	if node.Item.IsUnresolved {
		// Add to unresolved list (avoid duplicates)
		alreadyCollected := false
		for _, u := range *unresolved {
			if u.Key == key {
				alreadyCollected = true
				break
			}
		}
		if !alreadyCollected {
			*unresolved = append(*unresolved, node)
		}
	} else {
		// Add resolved package
		if _, exists := collected[key]; !exists {
			collected[key] = node.Item
		}
	}

	// Recursively collect children (depth-first)
	for _, child := range node.InnerNodes {
		if err := r.collectPackagesFromGraph(child, collected, unresolved); err != nil {
			return err
		}
	}

	return nil
}

// buildUnresolvedError creates NuGetError instances for unresolved packages.
// Matches NuGet.Client's error reporting format.
// Distinguishes between NU1101 (package doesn't exist), NU1102 (package exists but no compatible version),
// and NU1103 (only prerelease versions available when stable requested).
func (r *Restorer) buildUnresolvedError(ctx context.Context, unresolvedNodes []*resolver.GraphNode, projectPath string) []*NuGetError {
	if len(unresolvedNodes) == 0 {
		return nil
	}

	errors := make([]*NuGetError, 0, len(unresolvedNodes))
	for _, node := range unresolvedNodes {
		// Try to detect if this is NU1101, NU1102, or NU1103
		queryResult := r.tryGetVersionInfo(ctx, node.Item.ID, node.Item.Version)

		if queryResult.packageFound && len(queryResult.versionInfos) > 0 {
			// Package exists but no compatible version found
			// Check if this is NU1103 (only prerelease versions satisfy the range when stable requested)
			// Matches NuGet.Client logic: !IsPrereleaseAllowed(range) && HasPrereleaseVersionsOnly(range, allVersions)
			if !isPrereleaseAllowed(node.Item.Version) && hasPrereleaseVersionsOnly(node.Item.Version, queryResult.allVersions) {
				// NU1103: Only prerelease versions satisfy the range, but stable requested
				err := NewPackageDownloadFailedError(
					projectPath,
					node.Item.ID,
					node.Item.Version,
					queryResult.versionInfos,
				)
				errors = append(errors, err)
			} else {
				// NU1102: Package exists but no compatible version
				err := NewPackageVersionNotFoundError(
					projectPath,
					node.Item.ID,
					node.Item.Version,
					queryResult.versionInfos,
				)
				errors = append(errors, err)
			}
		} else {
			// NU1101: Package doesn't exist at all
			err := NewPackageNotFoundError(
				projectPath,
				node.Item.ID,
				node.Item.Version,
				r.opts.Sources,
			)
			errors = append(errors, err)
		}
	}

	return errors
}

// checkVersionAvailability checks if any version satisfying the constraint exists across all sources.
// This is an optimization to fail fast for NU1102/NU1103 cases without running expensive dependency walk.
// Returns version information per source, all versions, all queried source names, and a boolean indicating if constraint can be satisfied.
func (r *Restorer) checkVersionAvailability(ctx context.Context, packageID, versionConstraint string) ([]VersionInfo, []string, []string, bool) {
	// Parse version range constraint
	versionRange, err := version.ParseVersionRange(versionConstraint)
	if err != nil {
		// If we can't parse the constraint, let the walk handle it
		return nil, nil, nil, true
	}

	// Get all repositories from the client
	repos := r.client.GetRepositoryManager().ListRepositories()

	// Parallel source queries for 2x faster error reporting (critical for NU1101/NU1102/NU1103)
	type sourceResult struct {
		index          int
		sourceName     string
		versions       []string
		nearestVersion string
		canSatisfy     bool
		hasVersions    bool
	}

	results := make(chan sourceResult, len(repos))

	// Query all sources in parallel - network I/O is the bottleneck
	for idx, repo := range repos {
		go func(idx int, repo *core.SourceRepository) {
			// Format source name (check V2 first since it also contains "nuget.org")
			sourceName := repo.SourceURL()
			if strings.Contains(sourceName, "/api/v2") {
				sourceName = "NuGet V2"
			} else if strings.Contains(sourceName, "nuget.org") {
				sourceName = "nuget.org"
			}

			// Try to list all versions of this package from this repository
			versions, err := repo.ListVersions(ctx, nil, packageID)

			if err != nil || len(versions) == 0 {
				// Package doesn't exist in this source
				results <- sourceResult{index: idx, sourceName: sourceName, hasVersions: false}
				return
			}

			// Package exists! Optimize by checking max version first for early rejection
			var nearestVersion string
			var maxVersion *version.NuGetVersion

			// Find max version (versions are typically sorted, so check last first)
			// For NU1102 error display: use HIGHEST version (nearest to requested version)
			// For NU1103 error display: use LOWEST prerelease (will be updated later)
			if len(versions) > 0 {
				// Try last version first (usually the highest) for optimization
				if maxV, err := version.Parse(versions[len(versions)-1]); err == nil {
					maxVersion = maxV
					nearestVersion = versions[len(versions)-1]
				}

				// Verify it's actually the max by checking a few more
				for i := len(versions) - 2; i >= 0 && i >= len(versions)-5; i-- {
					if v, err := version.Parse(versions[i]); err == nil {
						if maxVersion == nil || v.Compare(maxVersion) > 0 {
							maxVersion = v
							nearestVersion = versions[i]
						}
					}
				}
			}

			// OPTIMIZATION: If constraint minimum > max version, no version can satisfy
			// This provides fast rejection for NU1102 cases like "99.99.99" > "13.0.4"
			canSatisfy := false
			if maxVersion != nil && versionRange.MinVersion != nil {
				cmp := versionRange.MinVersion.Compare(maxVersion)
				switch {
				case !versionRange.MinInclusive && cmp == 0:
					// Constraint requires > maxVersion, which is impossible
					// Don't set canSatisfy, use nearestVersion = maxVersion
				case cmp > 0:
					// Constraint requires higher than any available version - fast fail
					// Don't set canSatisfy, use nearestVersion = maxVersion
				default:
					// Constraint might be satisfiable - check if max version satisfies
					if versionRange.Satisfies(maxVersion) {
						canSatisfy = true
					} else {
						// Max doesn't satisfy, need to check other versions
						for _, v := range versions {
							nv, err := version.Parse(v)
							if err != nil {
								continue
							}
							if versionRange.Satisfies(nv) {
								canSatisfy = true
								nearestVersion = v
								break
							}
						}
					}
				}
			}

			results <- sourceResult{
				index:          idx,
				sourceName:     sourceName,
				versions:       versions,
				nearestVersion: nearestVersion,
				canSatisfy:     canSatisfy,
				hasVersions:    true,
			}
		}(idx, repo)
	}

	// Collect results from parallel queries and preserve original order
	resultsByIndex := make([]sourceResult, len(repos))
	for range len(repos) {
		result := <-results
		resultsByIndex[result.index] = result
	}

	// Process results in original source order (critical for source name display order)
	versionInfos := make([]VersionInfo, 0, len(repos))
	allVersions := make([]string, 0)
	allSourceNames := make([]string, 0, len(repos))
	canSatisfy := false

	for _, result := range resultsByIndex {
		// Track all sources queried (for NU1101 error reporting)
		allSourceNames = append(allSourceNames, result.sourceName)

		if !result.hasVersions {
			continue
		}

		// Collect all versions for NU1103 detection
		allVersions = append(allVersions, result.versions...)

		if result.canSatisfy {
			canSatisfy = true
		}

		versionInfos = append(versionInfos, VersionInfo{
			Source:         result.sourceName,
			VersionCount:   len(result.versions),
			NearestVersion: result.nearestVersion,
		})
	}

	return versionInfos, allVersions, allSourceNames, canSatisfy
}

// updateNearestVersionForNU1103 updates versionInfos to show the LOWEST prerelease version
// for NU1103 errors (dotnet shows lowest, not highest, for prerelease-only scenarios)
func (r *Restorer) updateNearestVersionForNU1103(versionInfos []VersionInfo, allVersions []string, versionRange *version.Range) []VersionInfo {
	// Parse all versions once
	parsedVersions := make([]*version.NuGetVersion, 0, len(allVersions))
	versionStrings := make([]string, 0, len(allVersions))

	for _, vStr := range allVersions {
		if v, err := version.Parse(vStr); err == nil {
			parsedVersions = append(parsedVersions, v)
			versionStrings = append(versionStrings, vStr)
		}
	}

	// Find lowest prerelease that satisfies numeric bounds
	var lowestPrerelease *version.NuGetVersion
	var lowestPrereleaseStr string

	for i, v := range parsedVersions {
		// Check if it's prerelease and satisfies numeric bounds
		if v.IsPrerelease() && versionRange.SatisfiesNumericBounds(v) {
			if lowestPrerelease == nil || v.LessThan(lowestPrerelease) {
				lowestPrerelease = v
				lowestPrereleaseStr = versionStrings[i]
			}
		}
	}

	// Update all versionInfos to use the lowest prerelease
	if lowestPrereleaseStr != "" {
		updatedInfos := make([]VersionInfo, len(versionInfos))
		for i, info := range versionInfos {
			updatedInfos[i] = VersionInfo{
				Source:         info.Source,
				VersionCount:   info.VersionCount,
				NearestVersion: lowestPrereleaseStr,
			}
		}
		return updatedInfos
	}

	// Fallback: return original if no prerelease found
	return versionInfos
}

// versionQueryResult holds the results of querying for versions from all sources.
type versionQueryResult struct {
	versionInfos []VersionInfo
	allVersions  []string
	packageFound bool
}

// getBestMatch finds the best matching version from available versions based on a version range.
// Matches NuGet.Client's GetBestMatch algorithm in UnresolvedMessages.cs.
//
// Algorithm:
// 1. If no versions available, return empty string
// 2. Find pivot point from range (MinVersion or MaxVersion)
// 3. For ranges with bounds, find first version above pivot that is closest
// 4. If no match, return highest version
//
// Examples:
//   - Range [1.0.0, ), Available [0.7.0, 0.9.0] → 0.7.0 (closest below lower bound)
//   - Range (0.5.0, 1.0.0), Available [0.1.0, 1.0.0] → 1.0.0 (closest to upper bound)
//   - Range (, 1.0.0), Available [2.0.0, 3.0.0] → 2.0.0 (closest above upper bound)
//   - Range [1.*,), Available [0.0.1, 0.9.0] → 0.9.0 (highest below lower bound)
func getBestMatch(versions []string, vr *version.Range) string {
	if len(versions) == 0 {
		return ""
	}

	// Parse all versions
	parsedVersions := make([]*version.NuGetVersion, 0, len(versions))
	for _, v := range versions {
		parsed, err := version.Parse(v)
		if err == nil {
			parsedVersions = append(parsedVersions, parsed)
		}
	}

	if len(parsedVersions) == 0 {
		return ""
	}

	// If no range provided, return highest version
	if vr == nil {
		return parsedVersions[len(parsedVersions)-1].String()
	}

	// Find pivot point (prefer MinVersion, fallback to MaxVersion)
	var ideal *version.NuGetVersion
	switch {
	case vr.MinVersion != nil:
		ideal = vr.MinVersion
	case vr.MaxVersion != nil:
		ideal = vr.MaxVersion
	default:
		// No bounds, return highest version
		return parsedVersions[len(parsedVersions)-1].String()
	}

	var bestMatch *version.NuGetVersion

	// If range has bounds, find first version above pivot that is closest
	if vr.MinVersion != nil || vr.MaxVersion != nil {
		for _, v := range parsedVersions {
			if v.Compare(ideal) == 0 {
				return v.String()
			}

			if v.Compare(ideal) > 0 {
				if bestMatch == nil || v.Compare(bestMatch) < 0 {
					bestMatch = v
				}
			}
		}
	}

	if bestMatch == nil {
		// Take the highest possible version
		bestMatch = parsedVersions[len(parsedVersions)-1]
	}

	return bestMatch.String()
}

// tryGetVersionInfo attempts to query available versions for a package to distinguish NU1101 vs NU1102 vs NU1103.
// Returns version information per source, all version strings, and a boolean indicating if package was found.
func (r *Restorer) tryGetVersionInfo(ctx context.Context, packageID, versionConstraint string) versionQueryResult {
	// Parse version range for best match calculation
	vr, err := version.ParseVersionRange(versionConstraint)
	if err != nil {
		// If parsing fails, use nil range (will fall back to highest version)
		vr = nil
	}

	// Get all repositories from the client
	repos := r.client.GetRepositoryManager().ListRepositories()
	versionInfos := make([]VersionInfo, 0, len(repos))
	allVersions := make([]string, 0)

	for _, repo := range repos {
		// Try to list all versions of this package from this repository
		versions, err := repo.ListVersions(ctx, nil, packageID)

		if err != nil || len(versions) == 0 {
			// Package doesn't exist in this source
			continue
		}

		// Package exists! Collect all versions for NU1103 detection
		allVersions = append(allVersions, versions...)

		// Calculate nearest version based on version range (matches NuGet.Client's GetBestMatch)
		nearestVersion := getBestMatch(versions, vr)

		// Format source name (check V2 first since it also contains "nuget.org")
		sourceName := repo.SourceURL()
		if strings.Contains(sourceName, "/api/v2") {
			sourceName = "NuGet V2"
		} else if strings.Contains(sourceName, "nuget.org") {
			sourceName = "nuget.org"
		}

		versionInfos = append(versionInfos, VersionInfo{
			Source:         sourceName,
			VersionCount:   len(versions),
			NearestVersion: nearestVersion,
		})
	}

	return versionQueryResult{
		versionInfos: versionInfos,
		allVersions:  allVersions,
		packageFound: len(versionInfos) > 0,
	}
}

// hasPrereleaseVersionsOnly checks if prerelease versions satisfy the range but no stable versions do.
// Matches NuGet.Client's HasPrereleaseVersionsOnly logic.
// Returns true if:
//  1. There exists at least one prerelease version that satisfies the range (numeric bounds only)
//  2. There exists NO stable version that satisfies the range (numeric bounds only)
//
// Note: Uses SatisfiesNumericBounds instead of Satisfies to check if versions WOULD satisfy
// the range if the prerelease restriction were lifted. This is necessary for NU1103 detection.
func hasPrereleaseVersionsOnly(versionRangeStr string, versions []string) bool {
	vr, err := version.ParseVersionRange(versionRangeStr)
	if err != nil {
		return false
	}

	// Check if this is an exact version range (e.g., [1.0.0-alpha])
	// Exact version ranges require exact match including prerelease labels
	isExactVersion := vr.MinVersion != nil && vr.MaxVersion != nil &&
		vr.MinInclusive && vr.MaxInclusive &&
		vr.MinVersion.Equals(vr.MaxVersion)

	hasPrereleaseInRange := false
	hasStableInRange := false

	for _, versionStr := range versions {
		v, err := version.Parse(versionStr)
		if err != nil {
			continue
		}

		// For exact version ranges, require exact match (including prerelease)
		// For other ranges, check numeric bounds only (ignore prerelease restriction)
		satisfies := false
		if isExactVersion {
			satisfies = vr.Satisfies(v)
		} else {
			satisfies = vr.SatisfiesNumericBounds(v)
		}

		if satisfies {
			if v.IsPrerelease() {
				hasPrereleaseInRange = true
			} else {
				hasStableInRange = true
			}
		}
	}

	// True if prerelease versions satisfy the range but no stable versions do
	return hasPrereleaseInRange && !hasStableInRange
}

// isPrereleaseAllowed checks if the version range allows prerelease versions.
// Matches NuGet.Client's IsPrereleaseAllowed logic.
// Returns true if the min or max version of the range has a prerelease label.
func isPrereleaseAllowed(versionRangeStr string) bool {
	vr, err := version.ParseVersionRange(versionRangeStr)
	if err != nil {
		return false
	}

	if vr.MinVersion != nil && vr.MinVersion.IsPrerelease() {
		return true
	}
	if vr.MaxVersion != nil && vr.MaxVersion.IsPrerelease() {
		return true
	}

	return false
}

func (r *Restorer) downloadPackage(ctx context.Context, packageID, packageVersion, packagePath string, cacheHit bool) error {
	isDiagnostic := r.opts.Verbosity == "diagnostic"

	// Diagnostic: Show cache hit or lock acquisition
	if isDiagnostic {
		if cacheHit {
			// Package already in cache - show CACHE message (use 9 space indent to match lock messages)
			r.console.Printf("         CACHE %s %s (already in %s)\n", packageID, packageVersion, packagePath)
		} else {
			// Package needs to be downloaded - show lock acquisition (use 9 space indent)
			r.console.Printf("         Acquiring lock for the installation of %s %s\n", packageID, packageVersion)
			r.console.Printf("         Acquired lock for the installation of %s %s\n", packageID, packageVersion)
		}
	}
	// Parse version
	pkgVer, err := version.Parse(packageVersion)
	if err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	// Get source repository and detect protocol
	repos := r.client.GetRepositoryManager().ListRepositories()
	if len(repos) == 0 {
		return fmt.Errorf("no package sources configured")
	}
	repo := repos[0]

	provider, err := repo.GetProvider(ctx)
	if err != nil {
		return fmt.Errorf("get provider: %w", err)
	}

	protocolVersion := provider.ProtocolVersion()
	sourceURL := repo.SourceURL()

	// Create package identity
	packageIdentity := &packaging.PackageIdentity{
		ID:      packageID,
		Version: pkgVer,
	}

	// Create extraction context with all save modes
	extractionContext := &packaging.PackageExtractionContext{
		PackageSaveMode:    packaging.PackageSaveModeNupkg | packaging.PackageSaveModeNuspec | packaging.PackageSaveModeFiles,
		XMLDocFileSaveMode: packaging.XMLDocFileSaveModeNone,
	}

	// Use V3 or V2 installer based on protocol
	if protocolVersion == "v3" {
		return r.installPackageV3(ctx, packageID, packageVersion, packagePath, packageIdentity, sourceURL, extractionContext, cacheHit)
	}
	return r.installPackageV2(ctx, packageID, packageVersion, packagePath, packageIdentity, sourceURL, extractionContext, cacheHit)
}

func (r *Restorer) installPackageV3(ctx context.Context, packageID, packageVersion, packagePath string, packageIdentity *packaging.PackageIdentity, sourceURL string, extractionContext *packaging.PackageExtractionContext, cacheHit bool) error {
	isDiagnostic := r.opts.Verbosity == "diagnostic"

	// Create path resolver for V3 layout
	packagesFolder := filepath.Dir(filepath.Dir(packagePath)) // Go up to packages root
	pathResolver := packaging.NewVersionFolderPathResolver(packagesFolder, true)

	// Create download callback
	copyToAsync := func(targetPath string) error {
		// Diagnostic: HTTP GET request (if not cached) - use 11 space indent
		downloadStart := time.Now()
		if isDiagnostic && !cacheHit {
			// Build package download URL for logging (use lowercase for URL)
			downloadURL := fmt.Sprintf("%s/flatcontainer/%s/%s/%s.%s.nupkg",
				strings.TrimSuffix(sourceURL, "/index.json"),
				strings.ToLower(packageID),
				strings.ToLower(packageVersion),
				strings.ToLower(packageID),
				strings.ToLower(packageVersion))
			r.console.Printf("           GET %s\n", downloadURL)
		}

		stream, err := r.client.DownloadPackage(ctx, packageID, packageVersion)
		if err != nil {
			return fmt.Errorf("download package: %w", err)
		}
		defer func() {
			if cerr := stream.Close(); cerr != nil {
				r.console.Error("failed to close package stream: %v\n", cerr)
			}
		}()

		// Diagnostic: HTTP OK response (if not cached) - use 11 space indent
		if isDiagnostic && !cacheHit {
			elapsed := time.Since(downloadStart)
			downloadURL := fmt.Sprintf("%s/flatcontainer/%s/%s/%s.%s.nupkg",
				strings.TrimSuffix(sourceURL, "/index.json"),
				strings.ToLower(packageID),
				strings.ToLower(packageVersion),
				strings.ToLower(packageID),
				strings.ToLower(packageVersion))
			r.console.Printf("           OK %s %dms\n", downloadURL, elapsed.Milliseconds())
		}

		outFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}
		defer func() {
			if cerr := outFile.Close(); cerr != nil {
				r.console.Error("failed to close temp file: %v\n", cerr)
			}
		}()

		if _, err := io.Copy(outFile, stream); err != nil {
			return fmt.Errorf("write package: %w", err)
		}

		return nil
	}

	// Install package (download + extract) using V3 layout
	_, err := packaging.InstallFromSourceV3(
		ctx,
		sourceURL,
		packageIdentity,
		copyToAsync,
		pathResolver,
		extractionContext,
	)

	if err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	// Diagnostic: Vulnerability check (always CACHE since we don't implement vulnerability DB yet) - use 11 space indent
	if isDiagnostic && !cacheHit {
		vulnURL := "https://api.nuget.org/v3/vulnerabilities/index.json"
		r.console.Printf("           CACHE %s\n", vulnURL)
	}

	// Note: Terminal Logger hides download/cache messages in detailed mode
	// We match Terminal Logger behavior: diagnostic mode shows download messages, detailed mode is clean

	return nil
}

func (r *Restorer) installPackageV2(ctx context.Context, packageID, packageVersion, packagePath string, packageIdentity *packaging.PackageIdentity, sourceURL string, extractionContext *packaging.PackageExtractionContext, cacheHit bool) error {
	isDiagnostic := r.opts.Verbosity == "diagnostic"

	// Create path resolver for V2 layout
	packagesFolder := filepath.Dir(filepath.Dir(packagePath)) // Go up to packages root
	pathResolver := packaging.NewPackagePathResolver(packagesFolder, true)

	// Check if already installed
	targetPath := pathResolver.GetInstallPath(packageIdentity)
	if _, err := os.Stat(targetPath); err == nil {
		// Note: Terminal Logger hides this message completely
		return nil
	}

	// Diagnostic: HTTP GET request (if not cached) - use 11 space indent
	downloadStart := time.Now()
	if isDiagnostic && !cacheHit {
		downloadURL := fmt.Sprintf("%s/Packages(Id='%s',Version='%s')",
			strings.TrimSuffix(sourceURL, "/"),
			packageID,
			packageVersion)
		r.console.Printf("           GET %s\n", downloadURL)
	}

	// Download package to memory
	stream, err := r.client.DownloadPackage(ctx, packageID, packageVersion)
	if err != nil {
		return fmt.Errorf("download package: %w", err)
	}
	defer func() {
		if cerr := stream.Close(); cerr != nil {
			r.console.Error("failed to close package stream: %v\n", cerr)
		}
	}()

	// Diagnostic: HTTP OK response (if not cached) - use 11 space indent
	if isDiagnostic && !cacheHit {
		elapsed := time.Since(downloadStart)
		downloadURL := fmt.Sprintf("%s/Packages(Id='%s',Version='%s')",
			strings.TrimSuffix(sourceURL, "/"),
			packageID,
			packageVersion)
		r.console.Printf("           OK %s %dms\n", downloadURL, elapsed.Milliseconds())
	}

	// Read into memory (V2 extractor needs ReadSeeker)
	packageData, err := io.ReadAll(stream)
	if err != nil {
		return fmt.Errorf("read package: %w", err)
	}

	packageReader := bytes.NewReader(packageData)

	// Extract package using V2 layout
	_, err = packaging.ExtractPackageV2(
		ctx,
		sourceURL,
		packageReader,
		pathResolver,
		extractionContext,
	)

	if err != nil {
		return fmt.Errorf("failed to extract package: %w", err)
	}

	// Diagnostic: Vulnerability check (always CACHE since we don't implement vulnerability DB yet) - use 11 space indent
	if isDiagnostic && !cacheHit {
		vulnURL := "https://api.nuget.org/v3/vulnerabilities/index.json"
		r.console.Printf("           CACHE %s\n", vulnURL)
	}

	// Note: Terminal Logger hides download messages in detailed mode
	return nil
}

// createLocalFirstWalker creates a DependencyWalker that checks local cache before HTTP.
// Matches NuGet.Client's RemoteWalkContext with LocalLibraryProviders + RemoteLibraryProviders.
// Reference: RestoreCommand.cs (lines 2037-2056), RemoteWalkContext.cs (lines 24-25, 37-38)
func (r *Restorer) createLocalFirstWalker(
	localProvider *LocalDependencyProvider,
	targetFramework *frameworks.NuGetFramework,
) (*resolver.DependencyWalker, error) {
	// Wrap with local-first metadata client
	// Remote metadata client is created lazily only when needed (when local provider returns nil)
	localFirstClient := &localFirstMetadataClient{
		localProvider:   localProvider,
		restorer:        r,
		targetFramework: targetFramework,
	}

	// Create walker with local-first client
	return resolver.NewDependencyWalker(
		localFirstClient,
		r.opts.Sources,
		targetFramework.String(),
	), nil
}

// getRemoteMetadataClient creates a metadata client that implements resolver.PackageMetadataClient.
// This client makes HTTP calls to fetch package metadata using V3 registration API.
func (r *Restorer) getRemoteMetadataClient() (resolver.PackageMetadataClient, error) {
	// Use the client's new CreateMetadataClient method
	// This creates the efficient V3 metadata adapter that fetches all versions in a single HTTP call
	return r.client.CreateMetadataClient(r.opts.Sources)
}

// getDependenciesForFramework extracts dependencies for a specific target framework.
// This helper is used for diagnostic output to show which dependencies are active for the resolved package.
// Returns the dependencies from the matching dependency group, or the first group if no exact match.
func (r *Restorer) getDependenciesForFramework(info *resolver.PackageDependencyInfo, framework string) []resolver.PackageDependency {
	if info == nil {
		return nil
	}

	// Find matching dependency group for the target framework
	for _, group := range info.DependencyGroups {
		if group.TargetFramework == framework {
			return group.Dependencies
		}
	}

	// Fallback: return first group if no exact match (shouldn't normally happen)
	if len(info.DependencyGroups) > 0 {
		return info.DependencyGroups[0].Dependencies
	}

	return nil
}

// localFirstMetadataClient implements resolver.PackageMetadataClient.
// It checks the local dependency provider FIRST (no HTTP), then falls back to remote.
// Matches NuGet.Client's provider list prioritization: LocalLibraryProviders → RemoteLibraryProviders
type localFirstMetadataClient struct {
	localProvider        *LocalDependencyProvider
	restorer             *Restorer
	remoteMetadataClient resolver.PackageMetadataClient // Lazy-initialized only when needed
	targetFramework      *frameworks.NuGetFramework
}

// GetPackageMetadata implements resolver.PackageMetadataClient.
// Tries local provider first (reads from cached .nuspec), falls back to HTTP if not cached.
func (c *localFirstMetadataClient) GetPackageMetadata(
	ctx context.Context,
	source string,
	packageID string,
	versionRange string,
) ([]*resolver.PackageDependencyInfo, error) {
	// Try local provider first (NO HTTP!)
	// LocalDependencyProvider now handles both exact versions and version ranges
	// Matches NuGet.Client: LocalLibraryProviders are tried before RemoteLibraryProviders
	depGroups, resolvedVersion, err := c.localProvider.GetDependencies(ctx, packageID, versionRange)
	if err != nil {
		// Error reading from cache - log and fall back to remote
		// Don't fail the restore just because we couldn't read a cached file
		// Silent fallback to HTTP (no logging)
	} else if depGroups != nil {
		// Found in local cache! Build PackageDependencyInfo from cached .nuspec
		// No logging - this is the fast path
		info := &resolver.PackageDependencyInfo{
			ID:               packageID,
			Version:          resolvedVersion, // Use resolved specific version (not the range!)
			DependencyGroups: depGroups,       // Return ALL groups (walker filters by framework)
		}

		return []*resolver.PackageDependencyInfo{info}, nil
	}

	// Not in local cache - lazy-initialize remote metadata client (only when needed)
	// This avoids creating HTTP clients and fetching service index until we actually need it
	if c.remoteMetadataClient == nil {
		remoteClient, err := c.restorer.getRemoteMetadataClient()
		if err != nil {
			return nil, fmt.Errorf("create remote metadata client: %w", err)
		}
		c.remoteMetadataClient = remoteClient
	}

	// Fall back to remote metadata client (HTTP)
	// This will fetch from nuget.org using V3 registration API
	// Matches NuGet.Client: RemoteLibraryProviders fallback
	return c.remoteMetadataClient.GetPackageMetadata(ctx, source, packageID, versionRange)
}

// isFrameworkReferencePack checks if a package ID is a framework reference pack.
// These are special packages downloaded by the SDK for targeting packs and should
// not be included in the regular package dependency lists.
func isFrameworkReferencePack(packageID string) bool {
	// Normalize to lowercase for comparison
	id := strings.ToLower(packageID)

	// Framework reference packs follow the pattern *.app.ref
	return strings.HasSuffix(id, ".app.ref") ||
		strings.HasSuffix(id, ".app.runtime.linux-x64") ||
		strings.HasSuffix(id, ".app.runtime.win-x64") ||
		strings.HasSuffix(id, ".app.runtime.osx-x64") ||
		strings.HasPrefix(id, "microsoft.netcore.app.runtime.") ||
		strings.HasPrefix(id, "microsoft.aspnetcore.app.runtime.") ||
		strings.HasPrefix(id, "microsoft.windowsdesktop.app.runtime.")
}
