package model

import (
	"io/fs"
	"os"
)

type MediaFolder struct {
	ID   string
	Name string
	Path string
}

func (f MediaFolder) FS() fs.FS {
	return os.DirFS(f.Path)
}

type MediaFolders []MediaFolder

type MediaFolderRepository interface {
	Get(id string) (*MediaFolder, error)
	GetAll() (MediaFolders, error)
}
