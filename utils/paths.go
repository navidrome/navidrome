package utils

import (
	"io/fs"

	"github.com/navidrome/navidrome/log"
)

func IsDirReadable(fsys fs.FS, path string) (bool, error) {
	dir, err := fsys.Open(path)
	if err != nil {
		return false, err
	}
	if err := dir.Close(); err != nil {
		log.Error("Error closing directory", "path", path, err)
	}
	return true, nil
}
