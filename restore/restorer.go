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
	targetFrameworks := proj.GetTargetFrameworks()
	if len(targetFrameworks) == 0 {
		return nil, fmt.Errorf("project has no target frameworks")
	}
	targetFramework := targetFrameworks[0] // Use first TFM

	// Track all resolved packages (direct + transitive)
	allResolvedPackages := make(map[string]*resolver.PackageDependencyInfo)
	unresolvedPackages := make([]*resolver.GraphNode, 0)

	// Phase 1: Walk dependency graph for each direct dependency
	// Use client's efficient metadata adapter (V3 registration API in single HTTP call)
	walker, err := r.client.CreateDependencyWalker(r.opts.Sources, targetFramework)
	if err != nil {
		return nil, fmt.Errorf("failed to create dependency walker: %w", err)
	}

	for _, pkgRef := range packageRefs {
		r.console.Printf("  Resolving %s %s...\n", pkgRef.Include, pkgRef.Version)

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
			targetFramework,
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
				r.console.Printf("    Package %s %s already cached\n", pkgInfo.ID, pkgInfo.Version)
				continue
			}
		}

		r.console.Printf("  Downloading %s %s...\n", pkgInfo.ID, pkgInfo.Version)

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
