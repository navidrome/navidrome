package utils

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func TempFileName(prefix, suffix string) string {
	return filepath.Join(os.TempDir(), prefix+uuid.NewString()+suffix)
}

func BaseName(filePath string) string {
	p := path.Base(filePath)
	return strings.TrimSuffix(p, path.Ext(p))
}
