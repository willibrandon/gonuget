package packaging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultLockTimeout is the maximum time to wait for file lock acquisition
	DefaultLockTimeout = 2 * time.Minute

	// LockRetryDelay is the retry delay for lock acquisition
	LockRetryDelay = 100 * time.Millisecond

	// LockFileExtension is the lock file extension
	LockFileExtension = ".lock"
)

// FileLock represents an exclusive file lock for package extraction.
// This prevents concurrent extractions from corrupting the same package.
//
// Reference: NuGet.Client ConcurrencyUtilities.cs ExecuteWithFileLocked
type FileLock struct {
	lockFilePath string
	lockFile     *os.File
}

// acquireFileLock acquires an exclusive file lock for the target file.
// Returns an unlock function that MUST be called when done (use defer).
//
// Lock mechanism:
// 1. Create lock file with unique temp name: {target}.lock.{random}
// 2. Attempt to rename to {target}.lock (atomic operation)
// 3. If rename succeeds, we have the lock
// 4. If rename fails, wait and retry
//
// Reference: NuGet.Client ConcurrencyUtilities.cs ExecuteWithFileLockedAsync
func acquireFileLock(ctx context.Context, targetPath string) (unlock func(), err error) {
	// Generate lock file path
	lockFilePath := targetPath + LockFileExtension

	// Create directory for lock file
	lockDir := filepath.Dir(lockFilePath)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	// Retry loop for lock acquisition
	startTime := time.Now()
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("lock acquisition cancelled: %w", ctx.Err())
		default:
		}

		// Check timeout
		if time.Since(startTime) > DefaultLockTimeout {
			return nil, fmt.Errorf("timeout acquiring lock for %s", targetPath)
		}

		// Attempt to acquire lock
		lock, err := tryAcquireLock(lockFilePath)
		if err == nil {
			// Lock acquired successfully
			// Note: releaseLock is platform-specific (Unix vs Windows)
			unlock := func() {
				releaseLock(lock)
			}
			return unlock, nil
		}

		// Lock held by another process, wait and retry
		time.Sleep(LockRetryDelay)
	}
}

// tryAcquireLock attempts to acquire the file lock.
// Platform-specific implementation in concurrency_unix.go and concurrency_windows.go
// Returns non-nil error if lock is held by another process.
// Reference: NuGet.Client ConcurrencyUtilities.AcquireFileStream with FileShare.None

// generateRandomHex generates a random hex string of specified length.
func generateRandomHex(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	return hex.EncodeToString(bytes)
}

// WithFileLock executes a function while holding an exclusive file lock.
// This is a convenience wrapper for acquireFileLock.
func WithFileLock(ctx context.Context, targetPath string, fn func() error) error {
	unlock, err := acquireFileLock(ctx, targetPath)
	if err != nil {
		return fmt.Errorf("acquire file lock: %w", err)
	}
	defer unlock()

	return fn()
}
