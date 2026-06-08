package cli

import (
	"github.com/harishphk/axen/internal/resolvers"
	"github.com/harishphk/axen/internal/utils"
	"fmt"
)

func AcquireProcessLock() (*utils.FileLock, error) {
	lockPath := resolvers.GetProcessLockPath()
	if err := utils.EnsureDir(resolvers.GetAxenDir()); err != nil {
		return nil, err
	}
	lock := utils.NewFileLock(lockPath)
	if err := lock.Lock(); err != nil {
		return nil, fmt.Errorf("another axen process is currently running. Please wait")
	}
	return lock, nil
}
