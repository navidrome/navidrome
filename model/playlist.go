package model

type Playlist struct {
	ID       string
	Name     string
	Comment  string
	FullPath string
	Duration int
	Owner    string
	Public   bool
	Tracks   MediaFiles
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
