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

// Watcher is a storage with the ability to start and stop a fs watcher.
type Watcher interface {
	// Start starts a watcher on the whole FS and returns a channel to send detected changes.
	// The watcher should be stopped when the context is done.
	Start(context.Context) (<-chan string, error)
}
