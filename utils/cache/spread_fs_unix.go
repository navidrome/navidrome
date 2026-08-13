//go:build !windows

package cache

import (
	"os"

	"github.com/djherbis/stream"
)

// createDataFile unlinks instead of truncating, so a re-created entry gets a fresh
// inode and readers still holding the old one see its full contents.
func createDataFile(name string) (stream.File, error) {
	if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
}
