package storage

import (
	"context"
	"io/fs"

	"github.com/navidrome/navidrome/model/metadata"
)

type Storage interface {
	FS() (MusicFS, error)
}

// MusicFS is an interface that extends the fs.FS interface with the ability to read tags from files
type MusicFS interface {
	fs.FS
	ReadTags(path ...string) (map[string]metadata.Info, error)
}

// SymlinkResolverFS is an optional interface for MusicFS implementations backed by a real
// filesystem. ResolveSymlink resolves the whole symlink chain of the named entry at the OS
// level and returns the final target's path — including targets outside the FS root, which
// fs.ReadLink-based resolution cannot follow.
type SymlinkResolverFS interface {
	ResolveSymlink(name string) (string, error)
}

// Watcher is a storage with the ability watch the FS and notify changes
type Watcher interface {
	// Start starts a watcher on the whole FS and returns a channel to send detected changes.
	// The watcher must be stopped when the context is done.
	Start(context.Context) (<-chan string, error)
}
