//go:build !windows

package local

import (
	"io/fs"
	"syscall"
)

// deviceID identifies the filesystem a file lives on, used to key birth time support per mount.
// It is returned opaquely because its width varies by platform, and it is only used as a map key.
func deviceID(fi fs.FileInfo) (any, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, false
	}
	return st.Dev, true
}
