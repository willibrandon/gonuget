package resolver

import (
	"context"
	"sync"
	"time"
)

// OperationCache caches in-flight operations to avoid duplicate work.
// Matches NuGet.Client's ConcurrentDictionary<LibraryRange, Task<GraphItem>>.
type OperationCache struct {
	// Maps cache key to operation state
	operations sync.Map // key -> *operationState

	// TTL for cache entries (0 = no expiration)
	ttl time.Duration
}

// operationState holds the state of an in-flight operation
type operationState struct {
	once      sync.Once
	result    *cacheResult
	done      chan struct{}
	startTime time.Time
}

// NewOperationCache creates a new operation cache
func NewOperationCache(ttl time.Duration) *OperationCache {
	return &OperationCache{
		ttl: ttl,
	}
}

// GetOrStart gets cached operation or starts a new one.
// This is the Go equivalent of ConcurrentDictionary.GetOrAdd with Task<T>.
func (oc *OperationCache) GetOrStart(
	ctx context.Context,
	key string,
	operation func(context.Context) (*PackageDependencyInfo, error),
) (*PackageDependencyInfo, error) {
	for {
		// Create new operation state with current time
		state := &operationState{
			done:      make(chan struct{}),
			startTime: time.Now(),
		}

		// Try to store our state, or get existing
		actual, loaded := oc.operations.LoadOrStore(key, state)
		state = actual.(*operationState)

		if !loaded {
			// We created the operation - run it
			state.once.Do(func() {
				info, err := operation(ctx)
				state.result = &cacheResult{info: info, err: err}
				close(state.done)

				// Clean up after a delay
				time.AfterFunc(5*time.Second, func() {
					oc.operations.Delete(key)
				})
			})
		} else if oc.ttl > 0 && time.Since(state.startTime) > oc.ttl {
			// Got existing operation - expired, delete and retry
			oc.operations.Delete(key)
			continue
		}

		// Wait for operation to complete
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-state.done:
			return state.result.info, state.result.err
		}
	}
}

type cacheResult struct {
	info *PackageDependencyInfo
	err  error
}

// Clear removes all cached operations
func (oc *OperationCache) Clear() {
	oc.operations.Range(func(key, value any) bool {
		oc.operations.Delete(key)
		return true
	})
}
