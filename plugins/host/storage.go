package host

import (
	"context"
	"io"
)

// StorageService provides access to a plugin-specific directory with read/write permissions
//
//nd:hostservice name=Storage permission=storage
type StorageService interface {
	// GetStoragePath retrieves the persistent storage path, if allowed
	//
	//nd:hostfunc
	GetStoragePath(ctx context.Context) string

	// Open opens a file in the plugin storage for reading.
	//
	// The path is relative to the storage folder and must stay inside it; symlinks leading
	// outside it are refused.
	//nd:hostfunc
	Open(ctx context.Context, path string) (io.ReadCloser, error)

	// Create creates or truncates a file in the plugin storage for writing, creating missing
	// parent folders. The path follows the same rules as Open.
	//nd:hostfunc
	Create(ctx context.Context, path string) (io.WriteCloser, error)
}
