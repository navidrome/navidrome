package model

import (
	"time"
)

type Playlist struct {
	ID        string     `structs:"id" json:"id"          orm:"column(id)"`
	Name      string     `structs:"name" json:"name"`
	Comment   string     `structs:"comment" json:"comment"`
	Duration  float32    `structs:"duration" json:"duration"`
	Size      int64      `structs:"size" json:"size"`
	SongCount int        `structs:"song_count" json:"songCount"`
	Owner     string     `structs:"owner" json:"owner"`
	Public    bool       `structs:"public" json:"public"`
	Tracks    MediaFiles `structs:"-" json:"tracks,omitempty"`
	Path      string     `structs:"path" json:"path"`
	Sync      bool       `structs:"sync" json:"sync"`
	CreatedAt time.Time  `structs:"created_at" json:"createdAt"`
	UpdatedAt time.Time  `structs:"updated_at" json:"updatedAt"`

	// SmartPlaylist attributes
	Rules       *SmartPlaylist `structs:"-" json:"rules"`
	EvaluatedAt time.Time      `structs:"evaluated_at" json:"evaluatedAt"`
}

type Playlists []Playlist

type PlaylistRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(pls *Playlist) error
	Get(id string) (*Playlist, error)
	GetAll(options ...QueryOptions) (Playlists, error)
	FindByPath(path string) (*Playlist, error)
	FindByID(id string) (*Playlist, error)
	Delete(id string) error
	Tracks(playlistId string) PlaylistTrackRepository
}

type PlaylistTrack struct {
	ID          string `json:"id"          orm:"column(id)"`
	MediaFileID string `json:"mediaFileId" orm:"column(media_file_id)"`
	PlaylistID  string `json:"playlistId" orm:"column(playlist_id)"`
	MediaFile
}

type PlaylistTracks []PlaylistTrack

type PlaylistTrackRepository interface {
	ResourceRepository
	Add(mediaFileIds []string) (int, error)
	AddAlbums(albumIds []string) (int, error)
	AddArtists(artistIds []string) (int, error)
	AddDiscs(discs []DiscID) (int, error)
	Update(mediaFileIds []string) error
	Delete(id string) error
	Reorder(pos int, newPos int) error
}
