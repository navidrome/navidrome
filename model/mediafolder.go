package model

import (
	"io/fs"
	"os"
)

type MediaFolder struct {
	ID   int32
	Name string
	Path string
}

func (f MediaFolder) FS() fs.FS {
	return os.DirFS(f.Path)
}

type MediaFolders []MediaFolder

type MediaFolderRepository interface {
	Get(id int32) (*MediaFolder, error)
	GetAll() (MediaFolders, error)
}
