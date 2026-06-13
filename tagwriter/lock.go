package tagwriter

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sys/unix"
)

var (
	lockRegistry = struct {
		mu    sync.RWMutex
		files map[string]*fileLock
	}{files: make(map[string]*fileLock)}
)

type fileLock struct {
	file *os.File
	ref  int
}

func LockFile(filePath string) (*fileLock, error) {
	absPath, err := abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	lockRegistry.mu.Lock()
	defer lockRegistry.mu.Unlock()

	if existing, ok := lockRegistry.files[absPath]; ok {
		existing.ref++
		return existing, nil
	}

	f, err := os.OpenFile(absPath, os.O_RDWR, 0)
	if err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied opening file: %w", err)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	lockRegistry.files[absPath] = &fileLock{file: f, ref: 1}
	return lockRegistry.files[absPath], nil
}

func UnlockFile(lock *fileLock) error {
	if lock == nil || lock.file == nil {
		return nil
	}

	lockRegistry.mu.Lock()
	defer lockRegistry.mu.Unlock()

	absPath, err := abs(lock.file.Name())
	if err != nil {
		return err
	}

	if existing, ok := lockRegistry.files[absPath]; ok {
		existing.ref--
		if existing.ref > 0 {
			return nil
		}
		delete(lockRegistry.files, absPath)
	}

	if err := unix.Flock(int(lock.file.Fd()), unix.LOCK_UN); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return lock.file.Close()
}

func abs(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if path[0] == '/' {
		return path, nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	return absPath, nil
}

func ClearLocks() {
	lockRegistry.mu.Lock()
	defer lockRegistry.mu.Unlock()
	for _, fl := range lockRegistry.files {
		unix.Flock(int(fl.file.Fd()), unix.LOCK_UN)
		fl.file.Close()
	}
	lockRegistry.files = make(map[string]*fileLock)
}