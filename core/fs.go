package core

import (
	"io/fs"
	"os"
)

func NewFS() fs.FS {
	return os.DirFS(".")
}
