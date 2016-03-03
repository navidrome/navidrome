package domain

type Album struct {
	Id           string
	Name         string
	ArtistId     string `parent:"artist"`
	CoverArtPath string // TODO http://stackoverflow.com/questions/13795842/linking-itunes-itc2-files-and-ituneslibrary-xml
	Artist       string
	AlbumArtist  string
	Year         int
	Compilation  bool
	Rating       int
	Genre        string
}

type AlbumRepository interface {
	BaseRepository
	Put(m *Album) error
	Get(id string) (*Album, error)
	FindByArtist(artistId string) ([]Album, error)
}
