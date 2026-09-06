package cli

import (
	"fmt"
	"time"

	"github.com/harishphk/axen/internal/resolvers"
	"github.com/harishphk/axen/internal/utils"
)

// AcquireProcessLock attempts to acquire the lock, retrying briefly up to 5 seconds
// so quick background operations do not cause instantaneous failures for the user.
func AcquireProcessLock() (*utils.FileLock, error) {
	lockPath := resolvers.GetProcessLockPath()
	if err := utils.EnsureDir(resolvers.GetAxenDir()); err != nil {
		return nil, err
	}
	lock := utils.NewFileLock(lockPath)

	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := lock.Lock(); err == nil {
			return lock, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("another axen process is currently running. Please wait")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TryAcquireProcessLock attempts to acquire the lock non-blockingly without waiting (for background tasks).
func TryAcquireProcessLock() (*utils.FileLock, error) {
	lockPath := resolvers.GetProcessLockPath()
	if err := utils.EnsureDir(resolvers.GetAxenDir()); err != nil {
		return nil, err
	}
	lock := utils.NewFileLock(lockPath)
	if err := lock.Lock(); err != nil {
		return nil, err
	}
	return lock, nil
}
