//go:build unix

package packaging

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// tryAcquireLock attempts to acquire the file lock using flock (Unix).
// Returns non-nil error if lock is held by another process.
func tryAcquireLock(lockFilePath string) (*FileLock, error) {
	// Open or create the lock file
	lockFile, err := os.OpenFile(lockFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	// syscall.LOCK_EX = exclusive lock
	// syscall.LOCK_NB = non-blocking (return error if can't acquire immediately)
	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = lockFile.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, fmt.Errorf("lock held by another process")
		}
		return nil, fmt.Errorf("flock error: %w", err)
	}

	return &FileLock{
		lockFilePath: lockFilePath,
		lockFile:     lockFile,
	}, nil
}

// releaseLock releases the file lock (Unix).
// On Unix, we only close the file - DO NOT remove it.
// Reference: NuGet.Client ConcurrencyUtilities.cs - DeleteOnClose causes
// concurrency issues on Mac OS X and Linux, so lock files are NOT deleted.
func releaseLock(lock *FileLock) {
	_ = lock.lockFile.Close()
	// DO NOT remove lock file on Unix - this causes race conditions
}
