//go:build windows

package packaging

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx = kernel32.NewProc("LockFileEx")
	// Note: No explicit unlock needed - lock is released when file is closed
	// Reference: NuGet.Client ConcurrencyUtilities.cs uses FileShare.None
	// and relies on FileStream.Dispose() for lock release
)

const (
	// LockFileEx flags
	LOCKFILE_EXCLUSIVE_LOCK   = 0x00000002
	LOCKFILE_FAIL_IMMEDIATELY = 0x00000001

	// Error codes
	ERROR_LOCK_VIOLATION = 33
)

// tryAcquireLock attempts to acquire the file lock using LockFileEx (Windows).
// Returns non-nil error if lock is held by another process.
// Reference: NuGet.Client uses FileShare.None which translates to exclusive lock
func tryAcquireLock(lockFilePath string) (*FileLock, error) {
	// Open or create the lock file with exclusive access
	// This matches NuGet.Client's FileShare.None behavior
	lockFile, err := os.OpenFile(lockFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	// Get the file handle
	handle := syscall.Handle(lockFile.Fd())

	// Prepare OVERLAPPED structure (required for LockFileEx, but we use zeros for synchronous)
	var overlapped syscall.Overlapped

	// Try to acquire exclusive lock (non-blocking)
	// LOCKFILE_EXCLUSIVE_LOCK = exclusive lock
	// LOCKFILE_FAIL_IMMEDIATELY = non-blocking (fail immediately if can't acquire)
	flags := uint32(LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY)

	// Lock the entire file (0xFFFFFFFF bytes from offset 0)
	r1, _, err := procLockFileEx.Call(
		uintptr(handle),
		uintptr(flags),
		uintptr(0),          // reserved, must be 0
		uintptr(0xFFFFFFFF), // number of bytes to lock (low)
		uintptr(0xFFFFFFFF), // number of bytes to lock (high)
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if r1 == 0 {
		_ = lockFile.Close()
		// Check if the error is ERROR_LOCK_VIOLATION (lock held by another process)
		if errno, ok := err.(syscall.Errno); ok && errno == ERROR_LOCK_VIOLATION {
			return nil, fmt.Errorf("lock held by another process")
		}
		return nil, fmt.Errorf("lock file error: %w", err)
	}

	return &FileLock{
		lockFilePath: lockFilePath,
		lockFile:     lockFile,
	}, nil
}

// releaseLock releases the file lock (Windows).
// On Windows, we close the file AND remove it (similar to DeleteOnClose).
// Reference: NuGet.Client uses FileOptions.DeleteOnClose on Windows.
func releaseLock(lock *FileLock) {
	_ = lock.lockFile.Close()
	_ = os.Remove(lock.lockFilePath)
}
