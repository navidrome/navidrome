package utils

import (
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

func TempFileName(prefix, suffix string) string {
	return filepath.Join(os.TempDir(), prefix+uuid.NewString()+suffix)
}
