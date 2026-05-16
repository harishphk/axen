package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileLock(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "axen-lock-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	lockPath := filepath.Join(tempDir, "axen.lock")

	lock1 := NewFileLock(lockPath)
	lock2 := NewFileLock(lockPath)

	// Acquire lock 1
	if err := lock1.Lock(); err != nil {
		t.Fatalf("Expected lock1 to lock successfully, got error: %v", err)
	}
	defer lock1.Unlock()

	// Try acquiring lock 2 (should fail)
	if err := lock2.Lock(); err == nil {
		lock2.Unlock()
		t.Fatal("Expected lock2 acquisition to fail while lock1 is held, but it succeeded")
	}

	// Release lock 1
	lock1.Unlock()

	// Now lock 2 should succeed
	if err := lock2.Lock(); err != nil {
		t.Fatalf("Expected lock2 to lock successfully after lock1 released, got error: %v", err)
	}
	lock2.Unlock()
}
