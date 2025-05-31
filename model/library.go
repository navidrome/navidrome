package model

import (
	"time"
)

type Library struct {
	ID                 int
	Name               string
	Path               string
	RemotePath         string
	LastScanAt         time.Time
	LastScanStartedAt  time.Time
	FullScanInProgress bool
	UpdatedAt          time.Time
	CreatedAt          time.Time
}

type Libraries []Library

type LibraryRepository interface {
	Get(id int) (*Library, error)
	// GetPath returns the path of the library with the given ID.
	// Its implementation must be optimized to avoid unnecessary queries.
	GetPath(id int) (string, error)
	GetAll(...QueryOptions) (Libraries, error)
	Put(*Library) error
	StoreMusicFolder() error
	AddArtist(id int, artistID string) error

	// TODO These methods should be moved to a core service
	ScanBegin(id int, fullScan bool) error
	ScanEnd(id int) error
	ScanInProgress() (bool, error)
}
