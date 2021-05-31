package model

import (
	"mime"
	"time"
)

type MediaFile struct {
	Annotations
	Bookmarkable

	ID                   string    `json:"id"            orm:"pk;column(id)"`
	Path                 string    `json:"path"`
	Title                string    `json:"title"`
	Album                string    `json:"album"`
	ArtistID             string    `json:"artistId"      orm:"pk;column(artist_id)"`
	Artist               string    `json:"artist"`
	AlbumArtistID        string    `json:"albumArtistId" orm:"pk;column(album_artist_id)"`
	AlbumArtist          string    `json:"albumArtist"`
	AlbumID              string    `json:"albumId"       orm:"pk;column(album_id)"`
	HasCoverArt          bool      `json:"hasCoverArt"`
	TrackNumber          int       `json:"trackNumber"`
	DiscNumber           int       `json:"discNumber"`
	DiscSubtitle         string    `json:"discSubtitle,omitempty"`
	Year                 int       `json:"year"`
	Size                 int64     `json:"size"`
	Suffix               string    `json:"suffix"`
	Duration             float32   `json:"duration"`
	BitRate              int       `json:"bitRate"`
	Genre                string    `json:"genre"`
	FullText             string    `json:"fullText"`
	SortTitle            string    `json:"sortTitle,omitempty"`
	SortAlbumName        string    `json:"sortAlbumName,omitempty"`
	SortArtistName       string    `json:"sortArtistName,omitempty"`
	SortAlbumArtistName  string    `json:"sortAlbumArtistName,omitempty"`
	OrderAlbumName       string    `json:"orderAlbumName"`
	OrderArtistName      string    `json:"orderArtistName"`
	OrderAlbumArtistName string    `json:"orderAlbumArtistName"`
	Compilation          bool      `json:"compilation"`
	Comment              string    `json:"comment,omitempty"`
	Lyrics               string    `json:"lyrics,omitempty"`
	Bpm                  int       `json:"bpm,omitempty"`
	CatalogNum           string    `json:"catalogNum,omitempty"`
	MbzTrackID           string    `json:"mbzTrackId,omitempty"         orm:"column(mbz_track_id)"`
	MbzAlbumID           string    `json:"mbzAlbumId,omitempty"         orm:"column(mbz_album_id)"`
	MbzArtistID          string    `json:"mbzArtistId,omitempty"        orm:"column(mbz_artist_id)"`
	MbzAlbumArtistID     string    `json:"mbzAlbumArtistId,omitempty"   orm:"column(mbz_album_artist_id)"`
	MbzAlbumType         string    `json:"mbzAlbumType,omitempty"`
	MbzAlbumComment      string    `json:"mbzAlbumComment,omitempty"`
	CreatedAt            time.Time `json:"createdAt"` // Time this entry was created in the DB
	UpdatedAt            time.Time `json:"updatedAt"` // Time of file last update (mtime)
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
	FindAllByPath(path string) (MediaFiles, error)
	FindByPath(path string) (*MediaFile, error)
	FindPathsRecursively(basePath string) ([]string, error)
	GetStarred(options ...QueryOptions) (MediaFiles, error)
	GetRandom(options ...QueryOptions) (MediaFiles, error)
	Search(q string, offset int, size int) (MediaFiles, error)
	Delete(id string) error
	DeleteByPath(path string) (int64, error)

	AnnotatedRepository
	BookmarkableRepository
}

func (mf MediaFile) GetAnnotations() Annotations {
	return mf.Annotations
}
