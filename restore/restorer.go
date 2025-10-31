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
