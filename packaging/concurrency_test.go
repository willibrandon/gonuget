package packaging

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquireFileLock(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "acquire lock successfully",
			timeout: 5 * time.Second,
			wantErr: false,
		},
		{
			name:    "acquire lock with short timeout",
			timeout: 100 * time.Millisecond,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			targetPath := filepath.Join(tempDir, "test.nupkg")

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			unlock, err := acquireFileLock(ctx, targetPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("acquireFileLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				defer unlock()

				// Verify lock file exists
				lockPath := targetPath + LockFileExtension
				if _, err := os.Stat(lockPath); os.IsNotExist(err) {
					t.Errorf("Lock file not created at %s", lockPath)
				}
			}
		})
	}
}

func TestAcquireFileLock_Concurrent(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "concurrent.nupkg")

	const numGoroutines = 10
	var wg sync.WaitGroup
	var successCount atomic.Int32
	var lockHolders atomic.Int32

	// Launch multiple goroutines trying to acquire the same lock
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			unlock, err := acquireFileLock(ctx, targetPath)
			if err != nil {
				t.Logf("Goroutine %d failed to acquire lock: %v", id, err)
				return
			}

			successCount.Add(1)

			// Verify we're the only lock holder
			current := lockHolders.Add(1)
			if current > 1 {
				t.Errorf("Multiple lock holders detected: %d", current)
			}

			// Hold lock briefly
			time.Sleep(10 * time.Millisecond)

			// Release lock
			lockHolders.Add(-1)
			unlock()
		}(i)
	}

	wg.Wait()

	// At least some goroutines should have succeeded
	if successCount.Load() == 0 {
		t.Errorf("No goroutines successfully acquired lock")
	}

	t.Logf("Success count: %d/%d", successCount.Load(), numGoroutines)
}

func TestAcquireFileLock_Timeout(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "timeout.nupkg")

	// First goroutine acquires lock
	ctx1 := context.Background()
	unlock1, err := acquireFileLock(ctx1, targetPath)
	if err != nil {
		t.Fatalf("First acquireFileLock() error = %v", err)
	}
	defer unlock1()

	// Second goroutine tries with very short timeout
	ctx2, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = acquireFileLock(ctx2, targetPath)
	if err == nil {
		t.Errorf("acquireFileLock() should timeout when lock is held")
	}
}

func TestAcquireFileLock_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "cancel.nupkg")

	// Hold lock in background
	ctx1 := context.Background()
	unlock1, err := acquireFileLock(ctx1, targetPath)
	if err != nil {
		t.Fatalf("First acquireFileLock() error = %v", err)
	}
	defer unlock1()

	// Try to acquire with cancelled context
	ctx2, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = acquireFileLock(ctx2, targetPath)
	if err == nil {
		t.Errorf("acquireFileLock() should fail on cancelled context")
	}
}

func TestAcquireFileLock_CreateLockDirectory(t *testing.T) {
	tempDir := t.TempDir()
	// Path with non-existent parent directories
	targetPath := filepath.Join(tempDir, "subdir", "nested", "test.nupkg")

	ctx := context.Background()
	unlock, err := acquireFileLock(ctx, targetPath)
	if err != nil {
		t.Fatalf("acquireFileLock() error = %v", err)
	}
	defer unlock()

	// Verify lock directory was created
	lockDir := filepath.Dir(targetPath + LockFileExtension)
	if _, err := os.Stat(lockDir); os.IsNotExist(err) {
		t.Errorf("Lock directory not created at %s", lockDir)
	}
}

func TestTryAcquireLock(t *testing.T) {
	tests := []struct {
		name       string
		setupLock  bool
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "acquire lock on new file",
			setupLock: false,
			wantErr:   false,
		},
		{
			name:       "fail to acquire held lock",
			setupLock:  true,
			wantErr:    true,
			wantErrMsg: "lock held by another process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			lockPath := filepath.Join(tempDir, "test.lock")

			// Setup: acquire lock first if needed
			if tt.setupLock {
				existingLock, err := tryAcquireLock(lockPath)
				if err != nil {
					t.Fatalf("Failed to setup lock: %v", err)
				}
				defer func() {
					if existingLock != nil {
						_ = existingLock.lockFile.Close()
						_ = os.Remove(existingLock.lockFilePath)
					}
				}()
			}

			lock, err := tryAcquireLock(lockPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("tryAcquireLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.wantErrMsg != "" {
				if !contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("tryAcquireLock() error = %v, want error containing %q", err, tt.wantErrMsg)
				}
			}

			if err == nil {
				defer func() {
					_ = lock.lockFile.Close()
					_ = os.Remove(lock.lockFilePath)
				}()

				// Verify lock file exists
				if _, err := os.Stat(lockPath); os.IsNotExist(err) {
					t.Errorf("Lock file not created at %s", lockPath)
				}

				// Verify lock file is open
				if lock.lockFile == nil {
					t.Errorf("Lock file handle is nil")
				}

				// Verify lock path is correct
				if lock.lockFilePath != lockPath {
					t.Errorf("Lock file path = %s, want %s", lock.lockFilePath, lockPath)
				}
			}
		})
	}
}

func TestTryAcquireLock_CleanupTempFile(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "cleanup.lock")

	// Acquire lock first to hold it
	existingLock, err := tryAcquireLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to acquire initial lock: %v", err)
	}
	defer func() {
		_ = existingLock.lockFile.Close()
		_ = os.Remove(existingLock.lockFilePath)
	}()

	// Try to acquire again (should fail)
	_, err = tryAcquireLock(lockPath)
	if err == nil {
		t.Errorf("tryAcquireLock() should fail when lock is held")
	}

	// Verify temp files are cleaned up
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		// Check for orphaned temp lock files (pattern: *.lock.*)
		if contains(name, ".lock.") && name != "cleanup.lock" {
			t.Errorf("Temp lock file not cleaned up: %s", name)
		}
	}
}

func TestGenerateRandomHex(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 8", 8},
		{"length 16", 16},
		{"length 32", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hex1 := generateRandomHex(tt.length)
			hex2 := generateRandomHex(tt.length)

			// Check length
			if len(hex1) != tt.length {
				t.Errorf("generateRandomHex(%d) length = %d, want %d", tt.length, len(hex1), tt.length)
			}

			// Check uniqueness
			if hex1 == hex2 {
				t.Errorf("generateRandomHex() generated duplicate: %s", hex1)
			}

			// Check hex characters
			for _, c := range hex1 {
				if !isHexChar(c) {
					t.Errorf("generateRandomHex() contains non-hex character: %c", c)
				}
			}
		})
	}
}

func TestWithFileLock(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "withlock.nupkg")

	var executed bool
	fn := func() error {
		executed = true
		// Verify lock file exists while function runs
		lockPath := targetPath + LockFileExtension
		if _, err := os.Stat(lockPath); os.IsNotExist(err) {
			t.Errorf("Lock file not found during function execution")
		}
		return nil
	}

	ctx := context.Background()
	err := WithFileLock(ctx, targetPath, fn)
	if err != nil {
		t.Errorf("WithFileLock() error = %v", err)
	}

	if !executed {
		t.Errorf("WithFileLock() did not execute function")
	}

	// Verify lock is released after function completes
	// Note: On Unix, lock files are intentionally NOT removed (like NuGet.Client)
	// On Windows, they are removed. We just verify the lock is released by
	// successfully acquiring it again.
	unlock2, err := acquireFileLock(ctx, targetPath)
	if err != nil {
		t.Errorf("Failed to acquire lock after release: %v", err)
	}
	if unlock2 != nil {
		unlock2()
	}
}

func TestWithFileLock_FunctionError(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "error.nupkg")

	expectedErr := os.ErrInvalid
	fn := func() error {
		return expectedErr
	}

	ctx := context.Background()
	err := WithFileLock(ctx, targetPath, fn)
	if err != expectedErr {
		t.Errorf("WithFileLock() error = %v, want %v", err, expectedErr)
	}

	// Verify lock is released even after error
	// Note: On Unix, lock files are intentionally NOT removed (like NuGet.Client)
	// We just verify the lock is released by successfully acquiring it again.
	ctx2 := context.Background()
	unlock2, err2 := acquireFileLock(ctx2, targetPath)
	if err2 != nil {
		t.Errorf("Failed to acquire lock after error release: %v", err2)
	}
	if unlock2 != nil {
		unlock2()
	}
}

func TestWithFileLock_LockAcquisitionError(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "lockfail.nupkg")

	// Hold lock in background
	ctx1 := context.Background()
	unlock1, err := acquireFileLock(ctx1, targetPath)
	if err != nil {
		t.Fatalf("acquireFileLock() error = %v", err)
	}
	defer unlock1()

	// Try WithFileLock with short timeout
	ctx2, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var executed bool
	fn := func() error {
		executed = true
		return nil
	}

	err = WithFileLock(ctx2, targetPath, fn)
	if err == nil {
		t.Errorf("WithFileLock() should fail when lock cannot be acquired")
	}

	if executed {
		t.Errorf("WithFileLock() executed function despite lock failure")
	}
}

func TestLockFileExtension(t *testing.T) {
	if LockFileExtension != ".lock" {
		t.Errorf("LockFileExtension = %q, want %q", LockFileExtension, ".lock")
	}
}

func TestDefaultLockTimeout(t *testing.T) {
	if DefaultLockTimeout != 2*time.Minute {
		t.Errorf("DefaultLockTimeout = %v, want %v", DefaultLockTimeout, 2*time.Minute)
	}
}

func TestLockRetryDelay(t *testing.T) {
	if LockRetryDelay != 100*time.Millisecond {
		t.Errorf("LockRetryDelay = %v, want %v", LockRetryDelay, 100*time.Millisecond)
	}
}

func TestAcquireFileLock_RealTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "realtimeout.nupkg")

	// Hold lock
	ctx1 := context.Background()
	unlock1, err := acquireFileLock(ctx1, targetPath)
	if err != nil {
		t.Fatalf("acquireFileLock() error = %v", err)
	}
	defer unlock1()

	// Try to acquire with timeout exceeding DefaultLockTimeout
	// Note: This test may take >2 minutes if timeout is actually enforced
	// We use a shorter custom timeout
	start := time.Now()
	customTimeout := 500 * time.Millisecond

	ctx2, cancel := context.WithTimeout(context.Background(), customTimeout)
	defer cancel()

	// Temporarily reduce DefaultLockTimeout behavior would need code change,
	// so we rely on context timeout instead
	_, err = acquireFileLock(ctx2, targetPath)
	elapsed := time.Since(start)

	if err == nil {
		t.Errorf("acquireFileLock() should timeout")
	}

	// Verify it timed out in reasonable time (not full 2 minutes)
	if elapsed > customTimeout+time.Second {
		t.Errorf("Timeout took too long: %v", elapsed)
	}
}

func BenchmarkAcquireFileLock(b *testing.B) {
	tempDir := b.TempDir()
	ctx := context.Background()

	b.ResetTimer()
	i := 0
	for b.Loop() {
		targetPath := filepath.Join(tempDir, "bench"+string(rune(i))+".nupkg")
		unlock, err := acquireFileLock(ctx, targetPath)
		if err != nil {
			b.Fatalf("acquireFileLock() error = %v", err)
		}
		unlock()
		i++
	}
}

func BenchmarkTryAcquireLock(b *testing.B) {
	tempDir := b.TempDir()

	b.ResetTimer()
	i := 0
	for b.Loop() {
		lockPath := filepath.Join(tempDir, "bench"+string(rune(i))+".lock")
		lock, err := tryAcquireLock(lockPath)
		if err != nil {
			b.Fatalf("tryAcquireLock() error = %v", err)
		}
		_ = lock.lockFile.Close()
		_ = os.Remove(lock.lockFilePath)
		i++
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func isHexChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
