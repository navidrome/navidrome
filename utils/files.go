package utils

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
)

func RandomSocketOrFileName(prefix, suffix string) string {
	socketPath := os.TempDir()
	// Windows needs to use a named pipe instead of a file for the socket
	// see https://mpv.io/manual/master#using-mpv-from-other-programs-or-scripts
	if runtime.GOOS == "windows" {
		socketPath = `\\.\pipe\mpvsocket`
	}
	return filepath.Join(socketPath, prefix+uuid.NewString()+suffix)
}
