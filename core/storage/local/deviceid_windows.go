//go:build windows

package local

import "io/fs"

// deviceID has no Windows equivalent, and none is needed: birth time comes straight from FileInfo.
func deviceID(fs.FileInfo) (any, bool) { return nil, false }
