package model

import "time"

type Album struct {
	Annotations

	ID                   string    `json:"id"            orm:"column(id)"`
	Name                 string    `json:"name"`
	CoverArtPath         string    `json:"coverArtPath"`
	CoverArtId           string    `json:"coverArtId"`
	ArtistID             string    `json:"artistId"      orm:"column(artist_id)"`
	Artist               string    `json:"artist"`
	AlbumArtistID        string    `json:"albumArtistId" orm:"column(album_artist_id)"`
	AlbumArtist          string    `json:"albumArtist"`
	AllArtistIDs         string    `json:"allArtistIds"  orm:"column(all_artist_ids)"`
	MaxYear              int       `json:"maxYear"`
	MinYear              int       `json:"minYear"`
	Compilation          bool      `json:"compilation"`
	Comment              string    `json:"comment,omitempty"`
	SongCount            int       `json:"songCount"`
	Duration             float32   `json:"duration"`
	Size                 int64     `json:"size"`
	Genre                string    `json:"genre"`
	FullText             string    `json:"fullText"`
	SortAlbumName        string    `json:"sortAlbumName,omitempty"`
	SortArtistName       string    `json:"sortArtistName,omitempty"`
	SortAlbumArtistName  string    `json:"sortAlbumArtistName,omitempty"`
	OrderAlbumName       string    `json:"orderAlbumName"`
	OrderAlbumArtistName string    `json:"orderAlbumArtistName"`
	CatalogNum           string    `json:"catalogNum,omitempty"`
	MbzAlbumID           string    `json:"mbzAlbumId,omitempty"         orm:"column(mbz_album_id)"`
	MbzAlbumArtistID     string    `json:"mbzAlbumArtistId,omitempty"   orm:"column(mbz_album_artist_id)"`
	MbzAlbumType         string    `json:"mbzAlbumType,omitempty"`
	MbzAlbumComment      string    `json:"mbzAlbumComment,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type Albums []Album

type AlbumRepository interface {
	CountAll(...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Get(id string) (*Album, error)
	FindByArtist(albumArtistId string) (Albums, error)
	GetAll(...QueryOptions) (Albums, error)
	GetRandom(...QueryOptions) (Albums, error)
	GetStarred(options ...QueryOptions) (Albums, error)
	Search(q string, offset int, size int) (Albums, error)
	Refresh(ids ...string) error
	AnnotatedRepository
}

func (a Album) GetAnnotations() Annotations {
	return a.Annotations
}
