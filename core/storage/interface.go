package storage

import (
	"io/fs"

	"github.com/navidrome/navidrome/model/tag"
)

type Storage interface {
	FS() (MusicFS, error)
}

// MusicFS is an interface that extends the fs.FS interface with the ability to read tags from files
type MusicFS interface {
	fs.FS
	ReadTags(path ...string) (map[string]tag.Properties, error)
}

// WatcherFS is an interface that extends the fs.FS interface with the ability to start and stop a fs watcher.
type WatcherFS interface {
	fs.FS

	// StartWatcher starts a watcher on the whole FS and returns a channel to send detected changes
	StartWatcher() (chan<- string, error)

	// StopWatcher stops the watcher
	StopWatcher()
}
