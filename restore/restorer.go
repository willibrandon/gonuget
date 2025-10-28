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

	// 1. Find project file
	projectPath, err := findProjectFile(args)
	if err != nil {
		return err
	}

	console.Printf("Restoring packages for %s...\n", projectPath)

	// 2. Load project
	proj, err := project.LoadProject(projectPath)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	// 3. Get package references
	packageRefs := proj.GetPackageReferences()

	if len(packageRefs) == 0 {
		console.Printf("Nothing to restore\n")
		return nil
	}

	// 4. Create restorer
	restorer := NewRestorer(opts, console)

	// 5. Execute restore
	result, err := restorer.Restore(ctx, proj, packageRefs)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	// 6. Generate lock file (project.assets.json)
	lockFile := NewLockFileBuilder().Build(proj, result)
	objDir := filepath.Join(filepath.Dir(proj.Path), "obj")
	assetsPath := filepath.Join(objDir, "project.assets.json")
	if err := lockFile.Save(assetsPath); err != nil {
		return fmt.Errorf("failed to save project.assets.json: %w", err)
	}

	// 7. Report summary
	elapsed := time.Since(start)
	console.Printf("  Restored %s (in %d ms)\n", projectPath, elapsed.Milliseconds())

	return nil
}

func findProjectFile(args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return project.FindProjectFile(cwd)
}

// Restorer executes restore operations.
type Restorer struct {
	opts    *Options
	console Console
	client  *core.Client
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
}

// PackageInfo holds package information.
type PackageInfo struct {
	ID      string
	Version string
	Path    string

	// IsDirect indicates if this is a direct dependency
	IsDirect bool

	// Parents lists packages that depend on this (for transitive deps)
	Parents []string
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
			r.console.Printf("  Restore skipped (cache valid)\n")

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
			// Group packages by direct vs transitive
			directPackageIDs := make(map[string]bool)
			for _, pkgRef := range packageRefs {
				directPackageIDs[strings.ToLower(pkgRef.Include)] = true
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

				info := PackageInfo{
					ID:       id,
					Version:  version,
					Path:     filepath.Join(packagesFolder, strings.ToLower(id), version),
					IsDirect: directPackageIDs[strings.ToLower(id)],
					Parents:  []string{},
				}

				if info.IsDirect {
					result.DirectPackages = append(result.DirectPackages, info)
				} else {
					result.TransitivePackages = append(result.TransitivePackages, info)
				}
			}

			result.CacheHit = true
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

	// Phase 1: Walk dependency graph for each direct dependency
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
		// Convert version to version range format if needed
		// Plain versions like "13.0.1" need to become "[13.0.1]" (exact range)
		versionRange := pkgRef.Version
		if versionRange != "" && !strings.Contains(versionRange, "[") && !strings.Contains(versionRange, "(") && !strings.Contains(versionRange, ",") && !strings.Contains(versionRange, "*") {
			// Looks like a plain version, convert to exact range
			versionRange = "[" + versionRange + "]"
		}
		if versionRange == "" {
			versionRange = "0.0.0" // Empty means any version >= 0.0.0
		}

		// Walk dependency graph (matches RemoteDependencyWalker.WalkAsync line 28)
		graphNode, err := walker.Walk(
			ctx,
			pkgRef.Include,
			versionRange,
			targetFrameworkStr,
			true, // recursive=true for transitive resolution
		)
		if err != nil {
			return nil, fmt.Errorf("failed to walk dependencies for %s: %w", pkgRef.Include, err)
		}

		// Collect all packages from graph (breadth-first)
		// Matches NuGet.Client: collect both resolved and unresolved packages
		if err := r.collectPackagesFromGraph(graphNode, allResolvedPackages, &unresolvedPackages); err != nil {
			return nil, err
		}
	}

	// Check for unresolved packages and fail restore if any found
	// Matches NuGet.Client: RestoreCommand fails when graphs have Unresolved.Count > 0
	if len(unresolvedPackages) > 0 {
		return nil, r.buildUnresolvedError(unresolvedPackages)
	}

	// Phase 2: Download all resolved packages (direct + transitive)
	// Matches ProjectRestoreCommand.InstallPackagesAsync behavior
	for _, pkgInfo := range allResolvedPackages {
		normalizedID := strings.ToLower(pkgInfo.ID)
		packagePath := filepath.Join(packagesFolder, normalizedID, pkgInfo.Version)

		// Check if package already exists in cache
		if !r.opts.Force {
			if _, err := os.Stat(packagePath); err == nil {
				continue
			}
		}

		// Download package
		if err := r.downloadPackage(ctx, normalizedID, pkgInfo.Version, packagePath); err != nil {
			return nil, fmt.Errorf("failed to download package %s %s: %w", pkgInfo.ID, pkgInfo.Version, err)
		}
	}

	// Phase 3: Categorize packages as direct vs transitive
	directPackageIDs := make(map[string]bool)
	for _, pkgRef := range packageRefs {
		directPackageIDs[strings.ToLower(pkgRef.Include)] = true
	}

	for _, pkgInfo := range allResolvedPackages {
		normalizedID := strings.ToLower(pkgInfo.ID)
		packagePath := filepath.Join(packagesFolder, normalizedID, pkgInfo.Version)

		info := PackageInfo{
			ID:       pkgInfo.ID,
			Version:  pkgInfo.Version,
			Path:     packagePath,
			IsDirect: directPackageIDs[normalizedID],
			Parents:  []string{}, // TODO: Collect from graph
		}

		if info.IsDirect {
			result.DirectPackages = append(result.DirectPackages, info)
		} else {
			result.TransitivePackages = append(result.TransitivePackages, info)
		}
	}

	// Phase 4: Write cache file for no-op optimization
	// Matches RestoreCommand.CommitCacheFileAsync (RestoreResult.cs line 296)
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
			Logs:                 []LogMessage{}, // TODO: Capture warnings/errors
		}

		// Save cache file
		if err := cacheFile.Save(cachePath); err != nil {
			// Don't fail restore if cache write fails
			r.console.Warning("Failed to write cache file: %v\n", err)
		}
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

// buildUnresolvedError creates an error message for unresolved packages.
// Matches NuGet.Client's error reporting format.
func (r *Restorer) buildUnresolvedError(unresolvedNodes []*resolver.GraphNode) error {
	if len(unresolvedNodes) == 0 {
		return nil
	}

	// Build error message
	msg := fmt.Sprintf("Restore failed. Unable to resolve %d package(s):\n", len(unresolvedNodes))
	for _, node := range unresolvedNodes {
		msg += fmt.Sprintf("  - %s %s\n", node.Item.ID, node.Item.Version)
	}

	return fmt.Errorf("%s", msg)
}

func (r *Restorer) downloadPackage(ctx context.Context, packageID, packageVersion, packagePath string) error {
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
		return r.installPackageV3(ctx, packageID, packageVersion, packagePath, packageIdentity, sourceURL, extractionContext)
	}
	return r.installPackageV2(ctx, packageID, packageVersion, packagePath, packageIdentity, sourceURL, extractionContext)
}

func (r *Restorer) installPackageV3(ctx context.Context, packageID, packageVersion, packagePath string, packageIdentity *packaging.PackageIdentity, sourceURL string, extractionContext *packaging.PackageExtractionContext) error {
	// Create path resolver for V3 layout
	packagesFolder := filepath.Dir(filepath.Dir(packagePath)) // Go up to packages root
	pathResolver := packaging.NewVersionFolderPathResolver(packagesFolder, true)

	// Create download callback
	copyToAsync := func(targetPath string) error {
		stream, err := r.client.DownloadPackage(ctx, packageID, packageVersion)
		if err != nil {
			return fmt.Errorf("download package: %w", err)
		}
		defer func() {
			if cerr := stream.Close(); cerr != nil {
				r.console.Error("failed to close package stream: %v\n", cerr)
			}
		}()

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
	installed, err := packaging.InstallFromSourceV3(
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

	if installed {
		r.console.Printf("    Downloaded and extracted to %s\n", packagePath)
	} else {
		r.console.Printf("    Package already cached at %s\n", packagePath)
	}

	return nil
}

func (r *Restorer) installPackageV2(ctx context.Context, packageID, packageVersion, packagePath string, packageIdentity *packaging.PackageIdentity, sourceURL string, extractionContext *packaging.PackageExtractionContext) error {
	// Create path resolver for V2 layout
	packagesFolder := filepath.Dir(filepath.Dir(packagePath)) // Go up to packages root
	pathResolver := packaging.NewPackagePathResolver(packagesFolder, true)

	// Check if already installed
	targetPath := pathResolver.GetInstallPath(packageIdentity)
	if _, err := os.Stat(targetPath); err == nil {
		r.console.Printf("    Package already cached at %s\n", packagePath)
		return nil
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

	r.console.Printf("    Downloaded and extracted to %s\n", packagePath)
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
