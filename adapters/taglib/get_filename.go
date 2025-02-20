//go:build !windows

package taglib

import "C"

func getFilename(s string) *C.char {
	return C.CString(s)
}
