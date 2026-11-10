package model

import (
	"cmp"
	"context"
	"strings"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
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
	PIDAlbum           string    `json:"pidAlbum" db:"pid_album"`
	PIDTrack           string    `json:"pidTrack" db:"pid_track"`
	ScannedPIDAlbum    string    `json:"-" db:"scanned_pid_album"`
	ScannedPIDTrack    string    `json:"-" db:"scanned_pid_track"`
}

// PIDConfig holds the persistent ID specs used to compute track and album IDs.
type PIDConfig struct {
	Track string
	Album string
}

// EffectivePID returns the PID specs in effect for this library: its own overrides, falling back to
// the global config.
func (l Library) EffectivePID() PIDConfig {
	return PIDConfig{
		Track: cmp.Or(l.PIDTrack, conf.Server.PID.Track),
		Album: cmp.Or(l.PIDAlbum, conf.Server.PID.Album),
	}
}

// PIDChanged reports whether the effective PID specs differ from the ones used by the last finished
// scan of this library. A library that was never scanned counts as changed.
func (l Library) PIDChanged() bool {
	pid := l.EffectivePID()
	return !strings.EqualFold(l.ScannedPIDAlbum, pid.Album) || !strings.EqualFold(l.ScannedPIDTrack, pid.Track)
}

// NeedsPIDRescan reports whether the library has content imported with an old PID config, so it must be
// rescanned in full. A library that never finished a scan has nothing to regroup.
func (l Library) NeedsPIDRescan() bool {
	return !l.LastScanAt.IsZero() && l.PIDChanged()
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
	rest.Repository[Library]
	Get(ctx context.Context, id int) (*Library, error)
	// GetPath returns the path of the library with the given ID.
	// Its implementation must be optimized to avoid unnecessary queries.
	GetPath(ctx context.Context, id int) (string, error)
	GetAll(ctx context.Context, options ...QueryOptions) (Libraries, error)
	CountAll(ctx context.Context, options ...QueryOptions) (int64, error)
	Put(ctx context.Context, l *Library, colsToUpdate ...string) error
	Delete(ctx context.Context, id int) error
	StoreMusicFolder(ctx context.Context) error
	AddArtist(ctx context.Context, id int, artistID string) error

	// User-library association methods
	GetUsersWithLibraryAccess(ctx context.Context, libraryID int) (Users, error)

	// TODO These methods should be moved to a core service
	ScanBegin(ctx context.Context, id int, fullScan bool) error
	ScanEnd(ctx context.Context, id int) error
	// SetScannedPID records the PID specs used by the last finished scan of the library
	SetScannedPID(ctx context.Context, id int, pid PIDConfig) error
	ScanInProgress(ctx context.Context) (bool, error)
	RefreshStats(ctx context.Context, id int) error
}
