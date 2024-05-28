package model

import (
	"time"
)

type Library struct {
	ID                int
	Name              string
	Path              string
	RemotePath        string
	LastScanAt        time.Time
	LastScanStartedAt time.Time
	UpdatedAt         time.Time
	CreatedAt         time.Time
}

type Libraries []Library

type LibraryRepository interface {
	Get(id int) (*Library, error)
	Put(*Library) error
	StoreMusicFolder() error
	AddArtist(id int, artistID string) error
	UpdateLastScanStartedAt(id int, t time.Time) error
	UpdateLastScanCompletedAt(id int, t time.Time) error
	GetAll(...QueryOptions) (Libraries, error)
}
