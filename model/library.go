package model

import (
	"time"
)

type Library struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	Path               string    `json:"path"`
	RemotePath         string    `json:"remotePath"`
	LastScanAt         time.Time `json:"lastScanAt"`
	LastScanStartedAt  time.Time `json:"lastScanStartedAt"`
	FullScanInProgress bool      `json:"fullScanInProgress"`
	UpdatedAt          time.Time `json:"updatedAt"`
	CreatedAt          time.Time `json:"createdAt"`
	TotalSongs         int       `json:"totalSongs"`
	TotalAlbums        int       `json:"totalAlbums"`
	TotalArtists       int       `json:"totalArtists"`
	TotalFolders       int       `json:"totalFolders"`
	TotalFiles         int       `json:"totalFiles"`
	TotalMissingFiles  int       `json:"totalMissingFiles"`
	TotalSize          int64     `json:"totalSize"`
}

type Libraries []Library

type LibraryRepository interface {
	Get(id int) (*Library, error)
	// GetPath returns the path of the library with the given ID.
	// Its implementation must be optimized to avoid unnecessary queries.
	GetPath(id int) (string, error)
	GetAll(...QueryOptions) (Libraries, error)
	CountAll(...QueryOptions) (int64, error)
	Put(*Library) error
	Delete(id int) error
	StoreMusicFolder() error
	AddArtist(id int, artistID string) error

	// User-library association methods
	GetUsersWithLibraryAccess(libraryID int) (Users, error)

	// TODO These methods should be moved to a core service
	ScanBegin(id int, fullScan bool) error
	ScanEnd(id int) error
	ScanInProgress() (bool, error)
	RefreshStats(id int) error
}
