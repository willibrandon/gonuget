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

	// Tracks operation start times for timeout
	startTimes sync.Map // key -> time.Time

	// TTL for cache entries (0 = no expiration)
	ttl time.Duration
}

// operationState holds the state of an in-flight operation
type operationState struct {
	once   sync.Once
	result *cacheResult
	done   chan struct{}
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
	// Check if expired first
	if oc.isExpired(key) {
		oc.operations.Delete(key)
		oc.startTimes.Delete(key)
	}

	// Create new operation state or get existing
	state := &operationState{
		done: make(chan struct{}),
	}

	actual, loaded := oc.operations.LoadOrStore(key, state)
	state = actual.(*operationState)

	if !loaded {
		// We created the operation - run it
		oc.startTimes.Store(key, time.Now())

		state.once.Do(func() {
			info, err := operation(ctx)
			state.result = &cacheResult{info: info, err: err}
			close(state.done)

			// Clean up after a delay
			time.AfterFunc(5*time.Second, func() {
				oc.operations.Delete(key)
				oc.startTimes.Delete(key)
			})
		})
	}

	// Wait for operation to complete
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-state.done:
		return state.result.info, state.result.err
	}
}

type cacheResult struct {
	info *PackageDependencyInfo
	err  error
}

// isExpired checks if a cache entry has exceeded TTL
func (oc *OperationCache) isExpired(key string) bool {
	if oc.ttl == 0 {
		return false
	}

	value, ok := oc.startTimes.Load(key)
	if !ok {
		return true
	}

	startTime := value.(time.Time)
	return time.Since(startTime) > oc.ttl
}

// Clear removes all cached operations
func (oc *OperationCache) Clear() {
	oc.operations.Range(func(key, value any) bool {
		oc.operations.Delete(key)
		return true
	})
	oc.startTimes.Range(func(key, value any) bool {
		oc.startTimes.Delete(key)
		return true
	})
}
