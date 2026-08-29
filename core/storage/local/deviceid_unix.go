//go:build !windows

package local

import (
	"io/fs"
	"syscall"
)

// deviceID identifies the filesystem a file lives on, used to key birth time support per mount.
func deviceID(fi fs.FileInfo) (uint64, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(st.Dev), true //nolint:gosec // Dev is an opaque identifier, only used as a map key
}
