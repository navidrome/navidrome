package model

import (
	"iter"
	"math"
	"sync"
	"time"

	"github.com/gohugoio/hashstructure"
)

type Album struct {
	Annotations `structs:"-" hash:"ignore"`

	ID            string `structs:"id" json:"id"`
	LibraryID     int    `structs:"library_id" json:"libraryId"`
	LibraryPath   string `structs:"-" json:"libraryPath" hash:"ignore"`
	LibraryName   string `structs:"-" json:"libraryName" hash:"ignore"`
	Name          string `structs:"name" json:"name"`
	EmbedArtPath  string `structs:"embed_art_path" json:"-"`
	AlbumArtistID string `structs:"album_artist_id" json:"albumArtistId"` // Deprecated, use Participants
	// AlbumArtist is the display name used for the album artist.
	AlbumArtist          string   `structs:"album_artist" json:"albumArtist"`
	MaxYear              int      `structs:"max_year" json:"maxYear"`
	MinYear              int      `structs:"min_year" json:"minYear"`
	Date                 string   `structs:"date" json:"date,omitempty"`
	MaxOriginalYear      int      `structs:"max_original_year" json:"maxOriginalYear"`
	MinOriginalYear      int      `structs:"min_original_year" json:"minOriginalYear"`
	OriginalDate         string   `structs:"original_date" json:"originalDate,omitempty"`
	ReleaseDate          string   `structs:"release_date" json:"releaseDate,omitempty"`
	Compilation          bool     `structs:"compilation" json:"compilation"`
	Comment              string   `structs:"comment" json:"comment,omitempty"`
	SongCount            int      `structs:"song_count" json:"songCount"`
	Duration             float32  `structs:"duration" json:"duration"`
	Size                 int64    `structs:"size" json:"size"`
	Discs                Discs    `structs:"discs" json:"discs,omitempty"`
	SortAlbumName        string   `structs:"sort_album_name" json:"sortAlbumName,omitempty"`
	SortAlbumArtistName  string   `structs:"sort_album_artist_name" json:"sortAlbumArtistName,omitempty"`
	OrderAlbumName       string   `structs:"order_album_name" json:"orderAlbumName"`
	OrderAlbumArtistName string   `structs:"order_album_artist_name" json:"orderAlbumArtistName"`
	CatalogNum           string   `structs:"catalog_num" json:"catalogNum,omitempty"`
	MbzAlbumID           string   `structs:"mbz_album_id" json:"mbzAlbumId,omitempty"`
	MbzAlbumArtistID     string   `structs:"mbz_album_artist_id" json:"mbzAlbumArtistId,omitempty"`
	MbzAlbumType         string   `structs:"mbz_album_type" json:"mbzAlbumType,omitempty"`
	MbzAlbumComment      string   `structs:"mbz_album_comment" json:"mbzAlbumComment,omitempty"`
	MbzReleaseGroupID    string   `structs:"mbz_release_group_id" json:"mbzReleaseGroupId,omitempty"`
	FolderIDs            []string `structs:"folder_ids" json:"-" hash:"set"` // All folders that contain media_files for this album
	ExplicitStatus       string   `structs:"explicit_status" json:"explicitStatus"`

	// External metadata fields
	Description           string     `structs:"description" json:"description,omitempty" hash:"ignore"`
	SmallImageUrl         string     `structs:"small_image_url" json:"smallImageUrl,omitempty" hash:"ignore"`
	MediumImageUrl        string     `structs:"medium_image_url" json:"mediumImageUrl,omitempty" hash:"ignore"`
	LargeImageUrl         string     `structs:"large_image_url" json:"largeImageUrl,omitempty" hash:"ignore"`
	ExternalUrl           string     `structs:"external_url" json:"externalUrl,omitempty" hash:"ignore"`
	ExternalInfoUpdatedAt *time.Time `structs:"external_info_updated_at" json:"externalInfoUpdatedAt" hash:"ignore"`

	Genre        string       `structs:"genre" json:"genre" hash:"ignore"`               // Easy access to the most common genre
	Genres       Genres       `structs:"-" json:"genres" hash:"ignore"`                  // Easy access to all genres for this album
	Tags         Tags         `structs:"tags" json:"tags,omitempty" hash:"ignore"`       // All imported tags for this album
	Participants Participants `structs:"participants" json:"participants" hash:"ignore"` // All artists that participated in this album

	Missing    bool      `structs:"missing" json:"missing"`                      // If all file of the album ar missing
	ImportedAt time.Time `structs:"imported_at" json:"importedAt" hash:"ignore"` // When this album was imported/updated
	CreatedAt  time.Time `structs:"created_at" json:"createdAt"`                 // Oldest CreatedAt for all songs in this album
	UpdatedAt  time.Time `structs:"updated_at" json:"updatedAt"`                 // Newest UpdatedAt for all songs in this album
}

func (a Album) CoverArtID() ArtworkID {
	return artworkIDFromAlbum(a)
}

// Equals compares two Album structs, ignoring calculated fields
func (a Album) Equals(other Album) bool {
	// Normalize float32 values to avoid false negatives
	a.Duration = float32(math.Floor(float64(a.Duration)))
	other.Duration = float32(math.Floor(float64(other.Duration)))

	opts := &hashstructure.HashOptions{
		IgnoreZeroValue: true,
		ZeroNil:         true,
	}
	hash1, _ := hashstructure.Hash(a, opts)
	hash2, _ := hashstructure.Hash(other, opts)

	return hash1 == hash2
}

// AlbumLevelTags contains all Tags marked as `album: true` in the mappings.yml file. They are not
// "first-class citizens" in the Album struct, but are still stored in the album table, in the `tags` column.
var AlbumLevelTags = sync.OnceValue(func() map[TagName]struct{} {
	tags := make(map[TagName]struct{})
	m := TagMappings()
	for t, conf := range m {
		if conf.Album {
			tags[t] = struct{}{}
		}
	}
	return tags
})

func (a *Album) SetTags(tags TagList) {
	a.Tags = tags.GroupByFrequency()
	for k := range a.Tags {
		if _, ok := AlbumLevelTags()[k]; !ok {
			delete(a.Tags, k)
		}
	}
}

type Discs map[int]string

func (d Discs) Add(discNumber int, discSubtitle string) {
	d[discNumber] = discSubtitle
}

type DiscID struct {
	AlbumID     string `json:"albumId"`
	ReleaseDate string `json:"releaseDate"`
	DiscNumber  int    `json:"discNumber"`
}

type Albums []Album

type AlbumCursor iter.Seq2[Album, error]

type AlbumRepository interface {
	CountAll(...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(*Album) error
	UpdateExternalInfo(*Album) error
	Get(id string) (*Album, error)
	GetAll(...QueryOptions) (Albums, error)

	// The following methods are used exclusively by the scanner:
	Touch(ids ...string) error
	TouchByMissingFolder() (int64, error)
	GetTouchedAlbums(libID int) (AlbumCursor, error)
	RefreshPlayCounts() (int64, error)
	CopyAttributes(fromID, toID string, columns ...string) error

	AnnotatedRepository
	SearchableRepository[Albums]
}
