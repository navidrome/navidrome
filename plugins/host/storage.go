package host

import "context"

// StorageService provides access to a plugin-specific directory with read/write permissions
//
//nd:hostservice name=Storage permission=storage
type StorageService interface {
	// GetStoragePath retrieves the persistent storage path, if allowed
	//
	//nd:hostfunc
	GetStoragePath(ctx context.Context) string
}
