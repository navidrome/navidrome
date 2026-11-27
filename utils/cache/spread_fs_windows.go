//go:build windows

package cache

import (
	"os"

	"github.com/djherbis/stream"
)

// createDataFile truncates in place: Windows refuses to unlink a file another handle
// has open, and failing the create would break every miss on a path with a live reader.
func createDataFile(name string) (stream.File, error) {
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
}
