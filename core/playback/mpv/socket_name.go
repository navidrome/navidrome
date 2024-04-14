//go:build !windows

package mpv

import "github.com/navidrome/navidrome/utils"

func socketName(prefix, suffix string) string {
	return utils.TempFileName(prefix, suffix)
}
