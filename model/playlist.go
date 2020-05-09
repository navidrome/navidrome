package model

import (
	"time"

	"github.com/deluan/rest"
)

type Playlist struct {
	ID        string     `json:"id"          orm:"column(id)"`
	Name      string     `json:"name"`
	Comment   string     `json:"comment"`
	Duration  float32    `json:"duration"`
	SongCount int        `json:"songCount"`
	Owner     string     `json:"owner"`
	Public    bool       `json:"public"`
	Tracks    MediaFiles `json:"tracks"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type PlaylistRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(pls *Playlist) error
	Get(id string) (*Playlist, error)
	GetAll(options ...QueryOptions) (Playlists, error)
	Delete(id string) error
	Tracks(playlistId string) PlaylistTracksRepository
}

type PlaylistTracks struct {
	ID          string `json:"id"          orm:"column(id)"`
	MediaFileID string `json:"mediaFileId" orm:"column(media_file_id)"`
	MediaFile
}

type PlaylistTracksRepository interface {
	rest.Repository
	//rest.Persistable
}

type Playlists []Playlist
