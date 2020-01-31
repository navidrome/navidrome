package model

import (
	"mime"
	"time"
)

type MediaFile struct {
	ID          string    `json:"id"            orm:"pk;column(id)"`
	Path        string    `json:"path"`
	Title       string    `json:"title"`
	Album       string    `json:"album"`
	Artist      string    `json:"artist"`
	ArtistID    string    `json:"artistId"      orm:"pk;column(artist_id)"`
	AlbumArtist string    `json:"albumArtist"`
	AlbumID     string    `json:"albumId"       orm:"pk;column(album_id)"`
	HasCoverArt bool      `json:"hasCoverArt"`
	TrackNumber int       `json:"trackNumber"`
	DiscNumber  int       `json:"discNumber"`
	Year        int       `json:"year"`
	Size        int       `json:"size"`
	Suffix      string    `json:"suffix"`
	Duration    int       `json:"duration"`
	BitRate     int       `json:"bitRate"`
	Genre       string    `json:"genre"`
	Compilation bool      `json:"compilation"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`

	// Annotations
	PlayCount int       `json:"-" orm:"-"`
	PlayDate  time.Time `json:"-" orm:"-"`
	Rating    int       `json:"-" orm:"-"`
	Starred   bool      `json:"-" orm:"-"`
	StarredAt time.Time `json:"-" orm:"-"`
}

func (mf *MediaFile) ContentType() string {
	return mime.TypeByExtension("." + mf.Suffix)
}

type MediaFiles []MediaFile

type MediaFileRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(m *MediaFile) error
	Get(id string) (*MediaFile, error)
	FindByAlbum(albumId string) (MediaFiles, error)
	FindByPath(path string) (MediaFiles, error)
	GetStarred(options ...QueryOptions) (MediaFiles, error)
	GetRandom(options ...QueryOptions) (MediaFiles, error)
	Search(q string, offset int, size int) (MediaFiles, error)
	Delete(id string) error
	DeleteByPath(path string) error
}
