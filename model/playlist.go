package model

type Playlist struct {
	ID       string
	Name     string
	Comment  string
	FullPath string
	Duration int
	Owner    string
	Public   bool
	Tracks   []string
}

type PlaylistRepository interface {
	CountAll() (int64, error)
	Exists(id string) (bool, error)
	Put(m *Playlist) error
	Get(id string) (*Playlist, error)
	GetAll(options ...QueryOptions) (Playlists, error)
}

type Playlists []Playlist
