package model

import (
	"mime"
	"time"
)

type MediaFile struct {
	HasCoverArt          bool      `json:"hasCoverArt"`
	Compilation          bool      `json:"compilation"`
	TrackNumber          int       `json:"trackNumber"`
	DiscNumber           int       `json:"discNumber"`
	Year                 int       `json:"year"`
	Size                 int       `json:"size"`
	BitRate              int       `json:"bitRate"`
	Duration             float32   `json:"duration"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
	ID                   string    `json:"id"            orm:"pk;column(id)"`
	Path                 string    `json:"path"`
	Title                string    `json:"title"`
	Album                string    `json:"album"`
	ArtistID             string    `json:"artistId"      orm:"pk;column(artist_id)"`
	Artist               string    `json:"artist"`
	AlbumArtistID        string    `json:"albumArtistId"`
	AlbumArtist          string    `json:"albumArtist"`
	AlbumID              string    `json:"albumId"       orm:"pk;column(album_id)"`
	Suffix               string    `json:"suffix"`
	Genre                string    `json:"genre"`
	FullText             string    `json:"fullText"`
	SortTitle            string    `json:"sortTitle"`
	SortAlbumName        string    `json:"sortAlbumName"`
	SortArtistName       string    `json:"sortArtistName"`
	SortAlbumArtistName  string    `json:"sortAlbumArtistName"`
	OrderAlbumName       string    `json:"orderAlbumName"`
	OrderArtistName      string    `json:"orderArtistName"`
	OrderAlbumArtistName string    `json:"orderAlbumArtistName"`

	// Annotations
	PlayCount int       `json:"playCount"   orm:"-"`
	PlayDate  time.Time `json:"playDate"    orm:"-"`
	Rating    int       `json:"rating"      orm:"-"`
	Starred   bool      `json:"starred"     orm:"-"`
	StarredAt time.Time `json:"starredAt"   orm:"-"`
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
	GetAll(options ...QueryOptions) (MediaFiles, error)
	FindByAlbum(albumId string) (MediaFiles, error)
	FindByPath(path string) (MediaFiles, error)
	GetStarred(options ...QueryOptions) (MediaFiles, error)
	GetRandom(options ...QueryOptions) (MediaFiles, error)
	Search(q string, offset int, size int) (MediaFiles, error)
	Delete(id string) error
	DeleteByPath(path string) error

	AnnotatedRepository
}
