//go:build !windows

package mpv

import (
	"os"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils"
)

func socketName(prefix, suffix string) string {
	return utils.TempFileName(prefix, suffix)
}

func removeSocket(socketName string) {
	log.Debug("Removing socketfile", "socketfile", socketName)
	err := os.Remove(socketName)
	if err != nil {
		log.Error("Error cleaning up socketfile", "socketfile", socketName, err)
	}
}
