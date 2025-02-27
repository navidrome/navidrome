//go:build windows

package mpv

import (
	"path/filepath"

	"github.com/navidrome/navidrome/model/id"
)

func socketName(prefix, suffix string) string {
	// Windows needs to use a named pipe for the socket
	// see https://mpv.io/manual/master#using-mpv-from-other-programs-or-scripts
	return filepath.Join(`\\.\pipe\mpvsocket`, prefix+id.NewRandom()+suffix)
}

func removeSocket(string) {
	// Windows automatically handles cleaning up named pipe
}
