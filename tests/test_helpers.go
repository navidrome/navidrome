package tests

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type testingT interface {
	TempDir() string
}

func TempFileName(t testingT, prefix, suffix string) string {
	return filepath.Join(t.TempDir(), prefix+uuid.NewString()+suffix)
}

func TempFile(t testingT, prefix, suffix string) (fs.File, string, error) {
	name := TempFileName(t, prefix, suffix)
	f, err := os.Create(name)
	return f, name, err
}
