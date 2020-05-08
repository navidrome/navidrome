package model

import "time"

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
}

type Playlists []Playlist
