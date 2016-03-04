package domain

import (
	"time"
	"mime"
)

type MediaFile struct {
	Id          string
	Path        string
	Title       string
	Album       string
	Artist      string
	AlbumArtist string
	AlbumId     string `parent:"album"`
	HasCoverArt bool
	TrackNumber int
	DiscNumber  int
	Year        int
	Size        string
	Suffix      string
	Duration    int
	BitRate     int
	Genre       string
	Compilation bool
	Starred     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (mf *MediaFile) ContentType() string {
	return mime.TypeByExtension("." + mf.Suffix)
}

type MediaFiles []MediaFile

type MediaFileRepository interface {
	BaseRepository
	Put(m *MediaFile) error
	Get(id string) (*MediaFile, error)
	FindByAlbum(albumId string) (MediaFiles, error)
}
