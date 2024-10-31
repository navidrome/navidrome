package model

import "time"

type Artist struct {
	Annotations `structs:"-"`

	ID              string               `structs:"id" json:"id"`
	Name            string               `structs:"name" json:"name"`
	SortArtistName  string               `structs:"sort_artist_name" json:"sortArtistName,omitempty"`
	OrderArtistName string               `structs:"order_artist_name" json:"orderArtistName,omitempty"`
	MbzArtistID     string               `structs:"mbz_artist_id" json:"mbzArtistId,omitempty"`
	Stats           map[Role]ArtistStats `structs:"-" json:"stats,omitempty"`
	// Deprecated: Use Stats instead
	Size int64 `structs:"-" json:"size,omitempty"`
	// Deprecated: Use Stats instead
	AlbumCount int `structs:"-" json:"albumCount,omitempty"`
	// Deprecated: Use Stats instead
	SongCount             int        `structs:"-" json:"songCount,omitempty"`
	Genres                Genres     `structs:"-" json:"genres,omitempty"`
	Biography             string     `structs:"biography" json:"biography,omitempty"`
	SmallImageUrl         string     `structs:"small_image_url" json:"smallImageUrl,omitempty"`
	MediumImageUrl        string     `structs:"medium_image_url" json:"mediumImageUrl,omitempty"`
	LargeImageUrl         string     `structs:"large_image_url" json:"largeImageUrl,omitempty"`
	ExternalUrl           string     `structs:"external_url" json:"externalUrl,omitempty"`
	SimilarArtists        Artists    `structs:"similar_artists"  json:"-"`
	ExternalInfoUpdatedAt *time.Time `structs:"external_info_updated_at" json:"externalInfoUpdatedAt,omitempty"`

	CreatedAt time.Time `structs:"created_at" json:"createdAt,omitempty"` // Oldest CreatedAt for all songs for this artist
	UpdatedAt time.Time `structs:"updated_at" json:"updatedAt,omitempty"` // Newest UpdatedAt for all songs for this artist
}

type ArtistStats struct {
	SongCount  int   `json:"songCount"`
	AlbumCount int   `json:"albumCount"`
	Size       int64 `json:"size"`
}

func (a Artist) ArtistImageUrl() string {
	if a.LargeImageUrl != "" {
		return a.LargeImageUrl
	}
	if a.MediumImageUrl != "" {
		return a.MediumImageUrl
	}
	return a.SmallImageUrl
}

func (a Artist) CoverArtID() ArtworkID {
	return artworkIDFromArtist(a)
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
	Put(m *Artist, colsToUpdate ...string) error
	Get(id string) (*Artist, error)
	GetAll(options ...QueryOptions) (Artists, error)
	Search(q string, offset int, size int) (Artists, error)
	GetIndex() (ArtistIndexes, error)
	RefreshCounters() (int64, error)
	AnnotatedRepository
}
