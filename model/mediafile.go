package model

import (
	"mime"
	"time"
)

type MediaFile struct {
	Annotations  `structs:"-"`
	Bookmarkable `structs:"-"`

	ID                   string    `structs:"id" json:"id"            orm:"pk;column(id)"`
	Path                 string    `structs:"path" json:"path"`
	Title                string    `structs:"title" json:"title"`
	Album                string    `structs:"album" json:"album"`
	ArtistID             string    `structs:"artist_id" json:"artistId"      orm:"pk;column(artist_id)"`
	Artist               string    `structs:"artist" json:"artist"`
	AlbumArtistID        string    `structs:"album_artist_id" json:"albumArtistId" orm:"pk;column(album_artist_id)"`
	AlbumArtist          string    `structs:"album_artist" json:"albumArtist"`
	AlbumID              string    `structs:"album_id" json:"albumId"       orm:"pk;column(album_id)"`
	HasCoverArt          bool      `structs:"has_cover_art" json:"hasCoverArt"`
	TrackNumber          int       `structs:"track_number" json:"trackNumber"`
	DiscNumber           int       `structs:"disc_number" json:"discNumber"`
	DiscSubtitle         string    `structs:"disc_subtitle" json:"discSubtitle,omitempty"`
	Year                 int       `structs:"year" json:"year"`
	Size                 int64     `structs:"size" json:"size"`
	Suffix               string    `structs:"suffix" json:"suffix"`
	Duration             float32   `structs:"duration" json:"duration"`
	BitRate              int       `structs:"bit_rate" json:"bitRate"`
	Channels             int       `structs:"channels" json:"channels"`
	Genre                string    `structs:"genre" json:"genre"`
	Genres               Genres    `structs:"-" json:"genres"`
	FullText             string    `structs:"full_text" json:"fullText"`
	SortTitle            string    `structs:"sort_title" json:"sortTitle,omitempty"`
	SortAlbumName        string    `structs:"sort_album_name" json:"sortAlbumName,omitempty"`
	SortArtistName       string    `structs:"sort_artist_name" json:"sortArtistName,omitempty"`
	SortAlbumArtistName  string    `structs:"sort_album_artist_name" json:"sortAlbumArtistName,omitempty"`
	OrderTitle           string    `structs:"order_title" json:"orderTitle,omitempty"`
	OrderAlbumName       string    `structs:"order_album_name" json:"orderAlbumName"`
	OrderArtistName      string    `structs:"order_artist_name" json:"orderArtistName"`
	OrderAlbumArtistName string    `structs:"order_album_artist_name" json:"orderAlbumArtistName"`
	Compilation          bool      `structs:"compilation" json:"compilation"`
	Comment              string    `structs:"comment" json:"comment,omitempty"`
	Lyrics               string    `structs:"lyrics" json:"lyrics,omitempty"`
	Bpm                  int       `structs:"bpm" json:"bpm,omitempty"`
	CatalogNum           string    `structs:"catalog_num" json:"catalogNum,omitempty"`
	MbzTrackID           string    `structs:"mbz_track_id" json:"mbzTrackId,omitempty"         orm:"column(mbz_track_id)"`
	MbzReleaseTrackID    string    `structs:"mbz_release_track_id" json:"mbzReleaseTrackId,omitempty" orm:"column(mbz_release_track_id)"`
	MbzAlbumID           string    `structs:"mbz_album_id" json:"mbzAlbumId,omitempty"         orm:"column(mbz_album_id)"`
	MbzArtistID          string    `structs:"mbz_artist_id" json:"mbzArtistId,omitempty"        orm:"column(mbz_artist_id)"`
	MbzAlbumArtistID     string    `structs:"mbz_album_artist_id" json:"mbzAlbumArtistId,omitempty"   orm:"column(mbz_album_artist_id)"`
	MbzAlbumType         string    `structs:"mbz_album_type" json:"mbzAlbumType,omitempty"`
	MbzAlbumComment      string    `structs:"mbz_album_comment" json:"mbzAlbumComment,omitempty"`
	CreatedAt            time.Time `structs:"created_at" json:"createdAt"` // Time this entry was created in the DB
	UpdatedAt            time.Time `structs:"updated_at" json:"updatedAt"` // Time of file last update (mtime)
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
	Search(q string, offset int, size int) (MediaFiles, error)
	Delete(id string) error

	// Queries by path to support the scanner, no Annotations or Bookmarks required in the response
	FindAllByPath(path string) (MediaFiles, error)
	FindByPath(path string) (*MediaFile, error)
	FindPathsRecursively(basePath string) ([]string, error)
	DeleteByPath(path string) (int64, error)

	AnnotatedRepository
	BookmarkableRepository
}
