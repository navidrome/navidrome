package model

import (
	"time"
)

type Playlist struct {
	ID        string     `json:"id"          orm:"column(id)"`
	Name      string     `json:"name"`
	Comment   string     `json:"comment"`
	Duration  float32    `json:"duration"`
	Size      int64      `json:"size"`
	SongCount int        `json:"songCount"`
	Owner     string     `json:"owner"`
	Public    bool       `json:"public"`
	Tracks    MediaFiles `json:"tracks,omitempty"`
	Path      string     `json:"path"`
	Sync      bool       `json:"sync"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type Playlists []Playlist

type PlaylistRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(pls *Playlist) error
	Get(id string) (*Playlist, error)
	GetAll(options ...QueryOptions) (Playlists, error)
	FindByPath(path string) (*Playlist, error)
	Delete(id string) error
	Tracks(playlistId string) PlaylistTrackRepository
}

type PlaylistTrack struct {
	ID          string `json:"id"          orm:"column(id)"`
	MediaFileID string `json:"mediaFileId" orm:"column(media_file_id)"`
	PlaylistID  string `json:"playlistId" orm:"column(playlist_id)"`
	MediaFile
}

type AdminPlaylistTrack struct {
	ID          string `json:"id"          orm:"column(id)"`
	MediaFileID string `json:"mediaFileId" orm:"column(media_file_id)"`
	PlaylistID  string `json:"playlistId" orm:"column(playlist_id)"`
	AdminMediaFile
}

type PlaylistTracks []PlaylistTrack

type AdminPlaylistTracks []AdminPlaylistTrack

type PlaylistTrackRepository interface {
	ResourceRepository
	Add(mediaFileIds []string) error
	Update(mediaFileIds []string) error
	Delete(id string) error
	Reorder(pos int, newPos int) error
}
