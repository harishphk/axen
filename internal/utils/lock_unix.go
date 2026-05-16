//go:build !windows

package utils

import (
	"os"
	"syscall"
)

type FileLock struct {
	path string
	file *os.File
}

func NewFileLock(path string) *FileLock {
	return &FileLock{path: path}
}

func (l *FileLock) Lock() error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	l.file = f

	// Acquire exclusive, non-blocking lock
	// #nosec G115
	err = syscall.Flock(int(l.file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

func (l *FileLock) Unlock() {
	if l.file != nil {
		// #nosec G115
		_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
		_ = l.file.Close()
		l.file = nil
	}
}
