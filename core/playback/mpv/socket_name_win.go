//go:build windows

package mpv

import (
	"path/filepath"

	"github.com/google/uuid"
)

func SocketName() string {
	// Windows needs to use a named pipe for the socket
	// see https://mpv.io/manual/master#using-mpv-from-other-programs-or-scripts
	return filepath.Join(`\\.\pipe\mpvsocket`, prefix+uuid.NewString()+suffix)
}
