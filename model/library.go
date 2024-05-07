package model

import (
	"io/fs"
	"os"
)

type Library struct {
	ID   int32
	Name string
	Path string
}

func (f Library) FS() fs.FS {
	return os.DirFS(f.Path)
}

type Libraries []Library

type LibraryRepository interface {
	Get(id int32) (*Library, error)
	GetAll() (Libraries, error)
}
