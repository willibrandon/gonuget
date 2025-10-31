package restore

import "time"

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

// versionQueryResult holds the results of querying for versions from all sources.
type versionQueryResult struct {
	versionInfos []VersionInfo
	allVersions  []string
	packageFound bool
}
