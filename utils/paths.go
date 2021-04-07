package utils

import (
	"os"

	"github.com/navidrome/navidrome/log"
)

func IsDirReadable(path string) (bool, error) {
	dir, err := os.Open(path)
	if err != nil {
		return false, err
	}
	if err := dir.Close(); err != nil {
		log.Error("Error closing directory", "path", path, err)
	}
	return true, nil
}
