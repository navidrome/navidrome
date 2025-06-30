package model

import (
	"time"

	"github.com/navidrome/navidrome/utils/slice"
)

type Library struct {
	ID                 int       `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	Path               string    `json:"path" db:"path"`
	RemotePath         string    `json:"remotePath" db:"remote_path"`
	LastScanAt         time.Time `json:"lastScanAt" db:"last_scan_at"`
	LastScanStartedAt  time.Time `json:"lastScanStartedAt" db:"last_scan_started_at"`
	FullScanInProgress bool      `json:"fullScanInProgress" db:"full_scan_in_progress"`
	UpdatedAt          time.Time `json:"updatedAt" db:"updated_at"`
	CreatedAt          time.Time `json:"createdAt" db:"created_at"`
	TotalSongs         int       `json:"totalSongs" db:"total_songs"`
	TotalAlbums        int       `json:"totalAlbums" db:"total_albums"`
	TotalArtists       int       `json:"totalArtists" db:"total_artists"`
	TotalFolders       int       `json:"totalFolders" db:"total_folders"`
	TotalFiles         int       `json:"totalFiles" db:"total_files"`
	TotalMissingFiles  int       `json:"totalMissingFiles" db:"total_missing_files"`
	TotalSize          int64     `json:"totalSize" db:"total_size"`
	TotalDuration      float64   `json:"totalDuration" db:"total_duration"`
	DefaultNewUsers    bool      `json:"defaultNewUsers" db:"default_new_users"`
}

const (
	DefaultLibraryID   = 1
	DefaultLibraryName = "Music Library"
)

type Libraries []Library

func (l Libraries) IDs() []int {
	return slice.Map(l, func(lib Library) int { return lib.ID })
}

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
