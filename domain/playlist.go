package domain

type Playlist struct {
	Id     string
	Name   string
	Tracks []string
}

type PlaylistRepository interface {
	BaseRepository
	Put(m *Playlist) error
	Get(id string) (*Playlist, error)
	GetAll(options QueryOptions) (*Playlists, error)
	PurgeInactive(active *Playlists) error
}

type Playlists []Playlist
