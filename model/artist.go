package model

import "time"

type Artist struct {
	Annotations

	ID                    string    `json:"id"               orm:"column(id)"`
	Name                  string    `json:"name"`
	AlbumCount            int       `json:"albumCount"`
	SongCount             int       `json:"songCount"`
	FullText              string    `json:"fullText"`
	SortArtistName        string    `json:"sortArtistName,omitempty"`
	OrderArtistName       string    `json:"orderArtistName"`
	Size                  int64     `json:"size"`
	MbzArtistID           string    `json:"mbzArtistId,omitempty"      orm:"column(mbz_artist_id)"`
	Biography             string    `json:"biography,omitempty"`
	SmallImageUrl         string    `json:"smallImageUrl,omitempty"`
	MediumImageUrl        string    `json:"mediumImageUrl,omitempty"`
	LargeImageUrl         string    `json:"largeImageUrl,omitempty"`
	ExternalUrl           string    `json:"externalUrl,omitempty"      orm:"column(external_url)"`
	SimilarArtists        Artists   `json:"-"   orm:"-"`
	ExternalInfoUpdatedAt time.Time `json:"externalInfoUpdatedAt"`
}

func (a Artist) ArtistImageUrl() string {
	if a.MediumImageUrl != "" {
		return a.MediumImageUrl
	}
	if a.LargeImageUrl != "" {
		return a.LargeImageUrl
	}
	return a.SmallImageUrl
}

type Artists []Artist

type ArtistIndex struct {
	ID      string
	Artists Artists
}
type ArtistIndexes []ArtistIndex

type ArtistRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(m *Artist) error
	Get(id string) (*Artist, error)
	GetAll(options ...QueryOptions) (Artists, error)
	GetStarred(options ...QueryOptions) (Artists, error)
	Search(q string, offset int, size int) (Artists, error)
	Refresh(ids ...string) error
	GetIndex() (ArtistIndexes, error)
	AnnotatedRepository
}

func (a Artist) GetAnnotations() Annotations {
	return a.Annotations
}
