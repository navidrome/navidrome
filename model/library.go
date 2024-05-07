package model

import (
	"io/fs"
	"os"
	"time"
)

type Library struct {
	ID         int
	Name       string
	Path       string
	RemotePath string
	LastScanAt time.Time
	UpdatedAt  time.Time
	CreatedAt  time.Time
}

func (f Library) FS() fs.FS {
	return os.DirFS(f.Path)
}

type Libraries []Library

type LibraryRepository interface {
	Get(id int) (*Library, error)
	Put(*Library) error
	StoreMusicFolder() error
	AddArtist(id int, artistID string) error
	UpdateLastScan(id int, t time.Time) error
	GetAll(...QueryOptions) (Libraries, error)
}
