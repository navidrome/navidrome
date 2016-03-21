package domain

type Playlist struct {
	Id       string
	Name     string
	FullPath string
	Duration int
	Owner    string
	Public   bool
	Tracks   []string
}

type PlaylistRepository interface {
	BaseRepository
	Put(m *Playlist) error
	Get(id string) (*Playlist, error)
	GetAll(options QueryOptions) (Playlists, error)
	PurgeInactive(active Playlists) ([]string, error)
}

type Playlists []Playlist
