package domain

type Album struct {
	Id           string
	Name         string
	ArtistId     string `parent:"artist"`
	CoverArtPath string // TODO http://stackoverflow.com/questions/13795842/linking-itunes-itc2-files-and-ituneslibrary-xml
	Year         int
	Compilation  bool
	Rating       int
}

type AlbumRepository interface {
	BaseRepository
	Put(m *Album) error
	Get(id string) (*Album, error)
}
