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
	"github.com/willibrandon/gonuget/core"
	"github.com/willibrandon/gonuget/core/resolver"
	"github.com/willibrandon/gonuget/frameworks"
	"github.com/willibrandon/gonuget/version"
)

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

	// Get target frameworks
	targetFrameworkStrings := proj.GetTargetFrameworks()
	if len(targetFrameworkStrings) == 0 {
		return nil, fmt.Errorf("project has no target frameworks")
	}

	// Initialize FrameworkResults for multi-TFM support
	result.FrameworkResults = make(map[string]*FrameworkResult)

	// Track all resolved packages across ALL frameworks (union)
	// This will be used for backward compat (DirectPackages, TransitivePackages)
	allResolvedPackagesUnion := make(map[string]*resolver.PackageDependencyInfo)

	// Track direct package IDs (normalized to lowercase)
	// Matches cache hit path behavior: categorize by package ID, not ID+version
	// This ensures packages are categorized as direct if they're explicitly referenced in .csproj,
	// regardless of which version gets resolved (e.g., direct ref to 13.0.1 that resolves to 13.0.3)
	directPackageIDs := make(map[string]bool)
	for _, pkgRef := range packageRefs {
		normalizedID := strings.ToLower(pkgRef.Include)
		directPackageIDs[normalizedID] = true
	}

	// Loop through ALL target frameworks and restore each
	// Matches NuGet.Client RestoreCommand.GenerateRestoreGraphsAsync (creates one graph per framework)
	isDiagnostic = r.opts.Verbosity == "diagnostic" || r.opts.Verbosity == "diag"
	for _, targetFrameworkStr := range targetFrameworkStrings {
		// Parse target framework
		targetFramework, err := frameworks.ParseFramework(targetFrameworkStr)
		if err != nil {
			return nil, fmt.Errorf("parse target framework %s: %w", targetFrameworkStr, err)
		}

		// Restore this framework (dependency resolution only, no downloads yet)
		frameworkResult, err := r.restoreFramework(
			ctx,
			packageRefs,
			targetFrameworkStr,
			targetFramework,
			packagesFolder,
			directPackageIDs,
			isDiagnostic,
			result.PerformanceTiming,
		)
		if err != nil {
			return result, err
		}

		// Store framework-specific result
		result.FrameworkResults[targetFrameworkStr] = frameworkResult

		// Merge into union (for backward compatibility)
		for key, pkg := range frameworkResult.allResolvedPackages {
			allResolvedPackagesUnion[key] = pkg
		}
	}

	// Build backward-compatible DirectPackages and TransitivePackages from union
	// This maintains compatibility with existing code that expects flat lists
	allResolvedPackages := allResolvedPackagesUnion

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
	// Check if package ID (not ID+version) is in directPackageIDs
	// This matches NuGet.Client behavior and cache hit path
	for _, pkgInfo := range allResolvedPackages {
		normalizedID := strings.ToLower(pkgInfo.ID)
		packagePath := filepath.Join(packagesFolder, normalizedID, pkgInfo.Version)

		// Check if this package ID was directly referenced in project file
		isDirect := directPackageIDs[normalizedID]

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

// restoreFramework handles dependency resolution for a single target framework.
// Matches NuGet.Client's WalkDependenciesAsync pattern.
// Returns FrameworkResult with resolved packages for this framework.
func (r *Restorer) restoreFramework(
	ctx context.Context,
	packageRefs []project.PackageReference,
	targetFrameworkStr string,
	targetFramework *frameworks.NuGetFramework,
	packagesFolder string,
	directPackageIDs map[string]bool,
	isDiagnostic bool,
	perfTiming *PerformanceTiming,
) (*FrameworkResult, error) {
	// Diagnostic: Start resolution trace for this framework
	if isDiagnostic {
		r.console.Printf("\nResolving dependencies for %s:\n", targetFrameworkStr)
	}

	// Start timing dependency resolution for this framework
	resolutionStart := time.Now()

	// Create local dependency provider for cached packages (NO HTTP!)
	localProvider := NewLocalDependencyProvider(packagesFolder)

	// Build list of package dependencies for multi-root resolution
	packageDependencies := make([]resolver.PackageDependency, 0, len(packageRefs))

	// First pass: Validate all package versions exist (early failure optimization)
	for _, pkgRef := range packageRefs {
		versionRange := pkgRef.Version
		if versionRange == "" {
			versionRange = "0.0.0" // Empty means any version >= 0.0.0
		}

		// Diagnostic: Trace package resolution start
		if isDiagnostic {
			r.console.Printf("  %s %s (direct reference)\n", pkgRef.Include, versionRange)
			r.console.Printf("    Constraint: %s\n", versionRange)
		}

		// OPTIMIZATION: Early version availability check
		versionInfos, allVersions, allSourceNames, canSatisfy := r.checkVersionAvailability(ctx, pkgRef.Include, versionRange)

		// Diagnostic: Show available versions (limit to last 10)
		if isDiagnostic && len(allVersions) > 0 {
			displayVersions := allVersions
			if len(displayVersions) > 10 {
				displayVersions = displayVersions[len(displayVersions)-10:]
			}
			r.console.Printf("    Available versions: %s\n", strings.Join(displayVersions, ", "))
		}

		if !canSatisfy {
			// Version not found - return error for this framework
			var nugetErr *NuGetError
			switch {
			case len(versionInfos) == 0:
				nugetErr = NewPackageNotFoundError("", pkgRef.Include, versionRange, allSourceNames)
			case !isPrereleaseAllowed(versionRange) && hasPrereleaseVersionsOnly(versionRange, allVersions):
				parsedRange, _ := version.ParseVersionRange(versionRange)
				versionInfosNU1103 := r.updateNearestVersionForNU1103(versionInfos, allVersions, parsedRange)
				nugetErr = NewPackageDownloadFailedError("", pkgRef.Include, versionRange, versionInfosNU1103)
			default:
				nugetErr = NewPackageVersionNotFoundError("", pkgRef.Include, versionRange, versionInfos)
			}

			// Add error log for this framework
			r.addErrorLog(nugetErr, targetFrameworkStr)

			return nil, fmt.Errorf("package version not found for framework %s: %s %s", targetFrameworkStr, pkgRef.Include, versionRange)
		}

		// Add to package dependencies list
		packageDependencies = append(packageDependencies, resolver.PackageDependency{
			ID:           pkgRef.Include,
			VersionRange: versionRange,
		})
	}

	// Phase 2: Resolve all dependencies together using multi-root resolution
	// Create metadata client for resolver
	metadataClient, err := r.createLocalFirstMetadataClient(localProvider, targetFramework)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata client for %s: %w", targetFrameworkStr, err)
	}

	// Create resolver with conflict detection and resolution
	res := resolver.NewResolver(metadataClient, r.opts.Sources, targetFrameworkStr)
	transitiveResolver := resolver.NewTransitiveResolver(res)

	// Resolve all dependencies together (creates synthetic project root internally)
	resolutionResult, err := transitiveResolver.ResolveMultipleRoots(ctx, packageDependencies)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies for %s: %w", targetFrameworkStr, err)
	}

	// Extract resolved packages from resolution result
	allResolvedPackages := make(map[string]*resolver.PackageDependencyInfo)
	for _, pkg := range resolutionResult.Packages {
		key := pkg.Key()
		allResolvedPackages[key] = pkg
	}

	// Handle unresolved packages
	for _, pkg := range resolutionResult.Packages {
		if pkg.IsUnresolved {
			return nil, fmt.Errorf("unresolved package for %s: %s", targetFrameworkStr, pkg.ID)
		}
	}

	// Record resolution timing
	if isDiagnostic && perfTiming != nil {
		perfTiming.DependencyResolution += time.Since(resolutionStart)
	}

	// Diagnostic: Show dependency graph summary
	if isDiagnostic {
		directCount := len(packageRefs)
		transitiveCount := max(0, len(allResolvedPackages)-directCount)
		r.console.Printf("    Resolved %d direct + %d transitive = %d total packages\n", directCount, transitiveCount, len(allResolvedPackages))
	}

	// Check for downgrades
	if len(resolutionResult.Downgrades) > 0 {
		return nil, fmt.Errorf("package downgrades detected for framework %s", targetFrameworkStr)
	}

	// Build FrameworkResult
	frameworkResult := &FrameworkResult{
		Framework:           targetFrameworkStr,
		DirectPackages:      make([]PackageInfo, 0),
		TransitivePackages:  make([]PackageInfo, 0),
		allResolvedPackages: allResolvedPackages,
	}

	// Categorize packages as direct vs transitive
	for _, pkgInfo := range allResolvedPackages {
		normalizedID := strings.ToLower(pkgInfo.ID)
		packagePath := filepath.Join(packagesFolder, normalizedID, pkgInfo.Version)

		pkg := PackageInfo{
			ID:       pkgInfo.ID,
			Version:  pkgInfo.Version,
			Path:     packagePath,
			IsDirect: directPackageIDs[normalizedID],
		}

		if directPackageIDs[normalizedID] {
			frameworkResult.DirectPackages = append(frameworkResult.DirectPackages, pkg)
		} else {
			frameworkResult.TransitivePackages = append(frameworkResult.TransitivePackages, pkg)
		}
	}

	return frameworkResult, nil
}
