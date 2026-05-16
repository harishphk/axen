//go:build windows

package utils

import (
	"os"

	"golang.org/x/sys/windows"
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

	var overlapped windows.Overlapped
	h := windows.Handle(l.file.Fd())
	err = windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &overlapped)
	if err != nil {
		_ = l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

func (l *FileLock) Unlock() {
	if l.file != nil {
		var overlapped windows.Overlapped
		h := windows.Handle(l.file.Fd())
		_ = windows.UnlockFileEx(h, 0, 1, 0, &overlapped)
		_ = l.file.Close()
		l.file = nil
	}
}
