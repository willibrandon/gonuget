package restore

import (
	"fmt"
	"time"
)

// DiagnosticTracer captures restore operations for diagnostic output.
// This interface allows pluggable diagnostic implementations for different verbosity levels.
type DiagnosticTracer interface {
	// TracePackageResolution logs package version selection.
	// Shows why a specific version was chosen from available versions.
	TracePackageResolution(packageID, constraint string, available []string, selected string, reason string)

	// TraceFrameworkCheck logs framework compatibility check.
	TraceFrameworkCheck(packageID, packageVersion string, framework string, compatible bool)

	// TraceDependencyDiscovered logs when a dependency is found.
	TraceDependencyDiscovered(parentID, dependencyID, constraint string, isDirect bool)

	// TraceDependencyGraph logs the final resolved graph.
	TraceDependencyGraph(directCount, transitiveCount int)

	// TraceProjectAnalysis logs project file details.
	// Shows SDK, target frameworks, and package references.
	TraceProjectAnalysis(projectPath string, sdk string, targetFrameworks []string, packageCount int)

	// TracePackageSources logs active package sources.
	// Shows which sources will be used for package resolution.
	TracePackageSources(sources []string)

	// TracePerformanceBreakdown logs detailed performance metrics.
	// Shows timing for each phase and per-package operations.
	TracePerformanceBreakdown(timing *PerformanceTiming)

	// TraceAssetsGeneration logs generated output files.
	// Shows file paths, sizes, and metadata about generated assets.
	TraceAssetsGeneration(assets *AssetsInfo)
}

// AssetsInfo holds information about generated restore output files.
type AssetsInfo struct {
	// ProjectAssetsFile is the path to project.assets.json
	ProjectAssetsFile string
	ProjectAssetsSize int64
	PackageCount      int
	TargetFrameworks  []string

	// CacheFile is the path to project.nuget.cache
	CacheFile     string
	CacheFileSize int64
	DgSpecHash    string
}

// ResolutionTracer implements DiagnosticTracer for dependency resolution tracing.
// Only active when verbosity is set to "diagnostic".
type ResolutionTracer struct {
	console Console
	enabled bool
}

// NewResolutionTracer creates a new resolution tracer.
// Tracing is only enabled when verbosity is "diagnostic" or "diag".
func NewResolutionTracer(console Console, verbosity string) *ResolutionTracer {
	return &ResolutionTracer{
		console: console,
		enabled: verbosity == "diagnostic" || verbosity == "diag",
	}
}

// TracePackageResolution logs package version selection (diagnostic only).
func (t *ResolutionTracer) TracePackageResolution(packageID, constraint string, available []string, selected string, reason string) {
	if !t.enabled {
		return
	}

	// This will be called from the resolution code
	// Format: show available versions and selected version
	// Note: Implementation will be added when we hook into resolution code
}

// TraceFrameworkCheck logs framework compatibility check (diagnostic only).
func (t *ResolutionTracer) TraceFrameworkCheck(packageID, packageVersion string, framework string, compatible bool) {
	if !t.enabled {
		return
	}

	// Log framework compatibility
	// Format: "Framework: compatible with net8.0" or "Framework: incompatible with net6.0"
}

// TraceDependencyDiscovered logs when a dependency is found (diagnostic only).
func (t *ResolutionTracer) TraceDependencyDiscovered(parentID, dependencyID, constraint string, isDirect bool) {
	if !t.enabled {
		return
	}

	// Log dependency discovery
	// Format: "→ Serilog.Sinks.File >= 5.0.0" (indented under parent)
}

// TraceDependencyGraph logs the final resolved graph (diagnostic only).
func (t *ResolutionTracer) TraceDependencyGraph(directCount, transitiveCount int) {
	if !t.enabled {
		return
	}

	// Summary of dependency graph
	totalCount := directCount + transitiveCount
	t.console.Printf("Dependency graph resolved:\n")
	t.console.Printf("  %d packages total (%d direct, %d transitive)\n\n",
		totalCount, directCount, transitiveCount)
}

// TraceProjectAnalysis logs project file details (diagnostic only).
func (t *ResolutionTracer) TraceProjectAnalysis(projectPath string, sdk string, targetFrameworks []string, packageCount int) {
	if !t.enabled {
		return
	}

	t.console.Printf("Project analysis:\n")
	t.console.Printf("  File: %s\n", projectPath)
	if sdk != "" {
		t.console.Printf("  SDK: %s\n", sdk)
	}
	if len(targetFrameworks) > 0 {
		if len(targetFrameworks) == 1 {
			t.console.Printf("  Target framework: %s\n", targetFrameworks[0])
		} else {
			t.console.Printf("  Target frameworks: %v\n", targetFrameworks)
		}
	}
	if packageCount > 0 {
		t.console.Printf("  Package references: %d\n", packageCount)
	} else {
		t.console.Printf("  Package references: (none)\n")
	}
	t.console.Printf("\n")
}

// TracePackageSources logs active package sources (diagnostic only).
func (t *ResolutionTracer) TracePackageSources(sources []string) {
	if !t.enabled {
		return
	}

	t.console.Printf("Package sources:\n")
	for i, source := range sources {
		t.console.Printf("  %d. %s\n", i+1, source)
	}
	t.console.Printf("\n")
}

// TracePerformanceBreakdown logs detailed performance metrics (diagnostic only).
func (t *ResolutionTracer) TracePerformanceBreakdown(timing *PerformanceTiming) {
	if !t.enabled || timing == nil {
		return
	}

	// Check if there's any timing data to display
	total := timing.DependencyResolution + timing.PackageDownloads + timing.AssetsGeneration
	if total == 0 {
		return // No work was done (cache hit), don't show empty breakdown
	}

	t.console.Printf("\nPerformance breakdown:\n")

	// Dependency resolution
	if timing.DependencyResolution > 0 {
		t.console.Printf("  Dependency resolution: %s\n", formatDuration(timing.DependencyResolution))
		if len(timing.ResolutionTimings) > 0 {
			for pkg, dur := range timing.ResolutionTimings {
				t.console.Printf("    - %s: %s\n", pkg, formatDuration(dur))
			}
		}
	}

	// Package downloads
	if timing.PackageDownloads > 0 {
		t.console.Printf("  Package downloads: %s\n", formatDuration(timing.PackageDownloads))
		if len(timing.DownloadTimings) > 0 {
			for pkg, dur := range timing.DownloadTimings {
				cacheStatus := ""
				if cacheHit, exists := timing.CacheHits[pkg]; exists && cacheHit {
					cacheStatus = " (cache)"
				}
				t.console.Printf("    - %s: %s%s\n", pkg, formatDuration(dur), cacheStatus)
			}
		}
	}

	// Assets generation
	if timing.AssetsGeneration > 0 {
		t.console.Printf("  Assets generation: %s\n", formatDuration(timing.AssetsGeneration))
	}

	// Total (already calculated above, just print it)
	t.console.Printf("  Total: %s\n\n", formatDuration(total))
}

// TraceAssetsGeneration logs generated output files (diagnostic only).
func (t *ResolutionTracer) TraceAssetsGeneration(assets *AssetsInfo) {
	if !t.enabled || assets == nil {
		return
	}

	t.console.Printf("\nWriting restore outputs:\n")

	// project.assets.json
	if assets.ProjectAssetsFile != "" {
		size := formatFileSize(assets.ProjectAssetsSize)
		t.console.Printf("  ✓ %s (%s)\n", assets.ProjectAssetsFile, size)
		if assets.PackageCount > 0 {
			t.console.Printf("    - %d packages\n", assets.PackageCount)
		}
		if len(assets.TargetFrameworks) > 0 {
			if len(assets.TargetFrameworks) == 1 {
				t.console.Printf("    - 1 target framework (%s)\n", assets.TargetFrameworks[0])
			} else {
				t.console.Printf("    - %d target frameworks\n", len(assets.TargetFrameworks))
			}
		}
	}

	// project.nuget.cache
	if assets.CacheFile != "" {
		size := formatFileSize(assets.CacheFileSize)
		t.console.Printf("  ✓ %s (%s)\n", assets.CacheFile, size)
		if assets.DgSpecHash != "" {
			t.console.Printf("    - dgspec hash: %s\n", assets.DgSpecHash)
		}
	}
}

// formatFileSize formats a file size for display.
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)

	switch {
	case bytes < KB:
		return fmt.Sprintf("%d bytes", bytes)
	case bytes < MB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	}
}

// formatDuration formats a duration for display.
// Shows milliseconds for < 1000ms, seconds for >= 1000ms.
func formatDuration(d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// noOpTracer is a no-op implementation for non-diagnostic verbosity levels.
type noOpTracer struct{}

func (t *noOpTracer) TracePackageResolution(packageID, constraint string, available []string, selected string, reason string) {
}
func (t *noOpTracer) TraceFrameworkCheck(packageID, packageVersion string, framework string, compatible bool) {
}
func (t *noOpTracer) TraceDependencyDiscovered(parentID, dependencyID, constraint string, isDirect bool) {
}
func (t *noOpTracer) TraceDependencyGraph(directCount, transitiveCount int) {
}
func (t *noOpTracer) TraceProjectAnalysis(projectPath string, sdk string, targetFrameworks []string, packageCount int) {
}
func (t *noOpTracer) TracePackageSources(sources []string) {
}
func (t *noOpTracer) TracePerformanceBreakdown(timing *PerformanceTiming) {
}
func (t *noOpTracer) TraceAssetsGeneration(assets *AssetsInfo) {
}

// Ensure interfaces are implemented at compile time
var _ DiagnosticTracer = (*ResolutionTracer)(nil)
var _ DiagnosticTracer = (*noOpTracer)(nil)
