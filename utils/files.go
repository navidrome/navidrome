package utils

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/model/id"
)

func TempFileName(prefix, suffix string) string {
	return filepath.Join(os.TempDir(), prefix+id.NewRandom()+suffix)
}

func BaseName(filePath string) string {
	p := path.Base(filePath)
	return strings.TrimSuffix(p, path.Ext(p))
}

// FileExists checks if a file or directory exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}
