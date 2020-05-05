package model

import "time"

type Playlist struct {
	Public    bool
	Duration  float32
	CreatedAt time.Time
	UpdatedAt time.Time
	ID        string
	Name      string
	Comment   string
	Owner     string
	Tracks    MediaFiles
}

type PlaylistRepository interface {
	CountAll() (int64, error)
	Exists(id string) (bool, error)
	Put(pls *Playlist) error
	Get(id string) (*Playlist, error)
	GetAll(options ...QueryOptions) (Playlists, error)
	Delete(id string) error
}

type Playlists []Playlist
