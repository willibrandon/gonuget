package resolver

import (
	"context"
	"fmt"
	"sync"
)

// ConcurrencyTracker optionally tracks concurrent operations in ParallelResolver.
type ConcurrencyTracker interface {
	// Enter is called when a worker starts (after acquiring semaphore).
	Enter()
	// Exit is called when a worker finishes (before releasing semaphore).
	Exit()
}

// ParallelResolver provides advanced parallel resolution strategies.
// Matches NuGet.Client's parallel restoration capabilities.
type ParallelResolver struct {
	resolver   *Resolver
	maxWorkers int
	semaphore  chan struct{}      // Limits concurrent operations
	tracker    ConcurrencyTracker // Optional concurrency tracker
}

// NewParallelResolver creates a new parallel resolver.
func NewParallelResolver(resolver *Resolver, maxWorkers int) *ParallelResolver {
	if maxWorkers <= 0 {
		maxWorkers = 10 // Default
	}

	return &ParallelResolver{
		resolver:   resolver,
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
	}
}

// WithTracker sets an optional concurrency tracker.
func (pr *ParallelResolver) WithTracker(tracker ConcurrencyTracker) *ParallelResolver {
	pr.tracker = tracker
	return pr
}

// ResolveMultiplePackages resolves multiple packages in parallel.
func (pr *ParallelResolver) ResolveMultiplePackages(
	ctx context.Context,
	packages []PackageDependency,
) ([]*ResolutionResult, error) {
	results := make([]*ResolutionResult, len(packages))
	errors := make([]error, len(packages))

	var wg sync.WaitGroup

	for i, pkg := range packages {
		wg.Add(1)

		go func(index int, p PackageDependency) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case pr.semaphore <- struct{}{}:
				defer func() { <-pr.semaphore }()
			case <-ctx.Done():
				errors[index] = ctx.Err()
				return
			}

			// Track concurrency if tracker is set
			if pr.tracker != nil {
				pr.tracker.Enter()
				defer pr.tracker.Exit()
			}

			// Resolve package
			result, err := pr.resolver.Resolve(ctx, p.ID, p.VersionRange)
			if err != nil {
				errors[index] = err
				return
			}

			results[index] = result
		}(i, pkg)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("resolve package %d: %w", i, err)
		}
	}

	return results, nil
}

// ResolveProjectParallel resolves project dependencies with parallel optimization.
func (pr *ParallelResolver) ResolveProjectParallel(
	ctx context.Context,
	roots []PackageDependency,
) (*ResolutionResult, error) {
	// Use regular project resolution (already parallelized at fetch level)
	// The parallel fetch in DependencyWalker handles parallelism
	return pr.resolver.ResolveProject(ctx, roots)
}

// BatchResolve resolves packages in batches for better resource control.
func (pr *ParallelResolver) BatchResolve(
	ctx context.Context,
	packages []PackageDependency,
	batchSize int,
) ([]*ResolutionResult, error) {
	if batchSize <= 0 {
		batchSize = pr.maxWorkers
	}

	results := make([]*ResolutionResult, 0, len(packages))

	// Process in batches
	for i := 0; i < len(packages); i += batchSize {
		end := min(i+batchSize, len(packages))

		batch := packages[i:end]
		batchResults, err := pr.ResolveMultiplePackages(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", i/batchSize, err)
		}

		results = append(results, batchResults...)
	}

	return results, nil
}
