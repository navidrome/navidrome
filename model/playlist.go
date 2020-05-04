package model

import "time"

type Playlist struct {
	ID        string
	Name      string
	Comment   string
	Duration  float32
	Owner     string
	Public    bool
	Tracks    MediaFiles
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PlaylistRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(pls *Playlist) error
	Get(id string) (*Playlist, error)
	GetAll(options ...QueryOptions) (Playlists, error)
	Delete(id string) error
}

type Playlists []Playlist
