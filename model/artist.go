package model

import (
	"maps"
	"slices"
	"time"
)

type Artist struct {
	Annotations `structs:"-"`

	ID string `structs:"id" json:"id"`

	// Data based on tags
	Name            string `structs:"name" json:"name"`
	SortArtistName  string `structs:"sort_artist_name" json:"sortArtistName,omitempty"`
	OrderArtistName string `structs:"order_artist_name" json:"orderArtistName,omitempty"`
	MbzArtistID     string `structs:"mbz_artist_id" json:"mbzArtistId,omitempty"`

	// Data calculated from files
	Stats      map[Role]ArtistStats `structs:"-" json:"stats,omitempty"`
	Size       int64                `structs:"-" json:"size,omitempty"`
	AlbumCount int                  `structs:"-" json:"albumCount,omitempty"`
	SongCount  int                  `structs:"-" json:"songCount,omitempty"`

	// Data imported from external sources
	Biography             string     `structs:"biography" json:"biography,omitempty"`
	SmallImageUrl         string     `structs:"small_image_url" json:"smallImageUrl,omitempty"`
	MediumImageUrl        string     `structs:"medium_image_url" json:"mediumImageUrl,omitempty"`
	LargeImageUrl         string     `structs:"large_image_url" json:"largeImageUrl,omitempty"`
	ExternalUrl           string     `structs:"external_url" json:"externalUrl,omitempty"`
	SimilarArtists        Artists    `structs:"similar_artists"  json:"-"`
	ExternalInfoUpdatedAt *time.Time `structs:"external_info_updated_at" json:"externalInfoUpdatedAt,omitempty"`

	Missing bool `structs:"missing" json:"missing"`

	CreatedAt *time.Time `structs:"created_at" json:"createdAt,omitempty"`
	UpdatedAt *time.Time `structs:"updated_at" json:"updatedAt,omitempty"`
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

// Roles returns the roles this artist has participated in., based on the Stats field
func (a Artist) Roles() []Role {
	return slices.Collect(maps.Keys(a.Stats))
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
	UpdateExternalInfo(a *Artist) error
	Get(id string) (*Artist, error)
	GetAll(options ...QueryOptions) (Artists, error)
	GetIndex(includeMissing bool, libraryIds []int, roles ...Role) (ArtistIndexes, error)

	// The following methods are used exclusively by the scanner:
	RefreshPlayCounts() (int64, error)
	RefreshStats(allArtists bool) (int64, error)

	AnnotatedRepository
	SearchableRepository[Artists]
}
