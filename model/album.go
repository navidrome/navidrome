package model

import (
	"cmp"
	"slices"
	"time"

	"github.com/navidrome/navidrome/utils/slice"
)

type Album struct {
	Annotations `structs:"-"`

	ID                    string     `structs:"id" json:"id"`
	LibraryID             int        `structs:"library_id" json:"libraryId"`
	Name                  string     `structs:"name" json:"name"`
	EmbedArtPath          string     `structs:"embed_art_path" json:"embedArtPath"`
	ArtistID              string     `structs:"artist_id" json:"artistId"`
	Artist                string     `structs:"artist" json:"artist"`
	AlbumArtistID         string     `structs:"album_artist_id" json:"albumArtistId"`
	AlbumArtist           string     `structs:"album_artist" json:"albumArtist"`
	AllArtistIDs          string     `structs:"all_artist_ids" json:"allArtistIds"`
	MaxYear               int        `structs:"max_year" json:"maxYear"`
	MinYear               int        `structs:"min_year" json:"minYear"`
	Date                  string     `structs:"date" json:"date,omitempty"`
	MaxOriginalYear       int        `structs:"max_original_year" json:"maxOriginalYear"`
	MinOriginalYear       int        `structs:"min_original_year" json:"minOriginalYear"`
	OriginalDate          string     `structs:"original_date" json:"originalDate,omitempty"`
	ReleaseDate           string     `structs:"release_date" json:"releaseDate,omitempty"`
	Releases              int        `structs:"releases" json:"releases"`
	Compilation           bool       `structs:"compilation" json:"compilation"`
	Comment               string     `structs:"comment" json:"comment,omitempty"`
	SongCount             int        `structs:"song_count" json:"songCount"`
	Duration              float32    `structs:"duration" json:"duration"`
	Size                  int64      `structs:"size" json:"size"`
	Genre                 string     `structs:"genre" json:"genre"`
	Genres                Genres     `structs:"-" json:"genres"`
	Discs                 Discs      `structs:"discs" json:"discs,omitempty"`
	FullText              string     `structs:"full_text" json:"-"`
	SortAlbumName         string     `structs:"sort_album_name" json:"sortAlbumName,omitempty"`
	SortArtistName        string     `structs:"sort_artist_name" json:"sortArtistName,omitempty"`
	SortAlbumArtistName   string     `structs:"sort_album_artist_name" json:"sortAlbumArtistName,omitempty"`
	OrderAlbumName        string     `structs:"order_album_name" json:"orderAlbumName"`
	OrderAlbumArtistName  string     `structs:"order_album_artist_name" json:"orderAlbumArtistName"`
	CatalogNum            string     `structs:"catalog_num" json:"catalogNum,omitempty"`
	MbzAlbumID            string     `structs:"mbz_album_id" json:"mbzAlbumId,omitempty"`
	MbzAlbumArtistID      string     `structs:"mbz_album_artist_id" json:"mbzAlbumArtistId,omitempty"`
	MbzAlbumType          string     `structs:"mbz_album_type" json:"mbzAlbumType,omitempty"`
	MbzAlbumComment       string     `structs:"mbz_album_comment" json:"mbzAlbumComment,omitempty"`
	ImageFiles            string     `structs:"image_files" json:"imageFiles,omitempty"`
	Paths                 string     `structs:"paths" json:"paths,omitempty"`
	Description           string     `structs:"description" json:"description,omitempty"`
	SmallImageUrl         string     `structs:"small_image_url" json:"smallImageUrl,omitempty"`
	MediumImageUrl        string     `structs:"medium_image_url" json:"mediumImageUrl,omitempty"`
	LargeImageUrl         string     `structs:"large_image_url" json:"largeImageUrl,omitempty"`
	ExternalUrl           string     `structs:"external_url" json:"externalUrl,omitempty"`
	ExternalInfoUpdatedAt *time.Time `structs:"external_info_updated_at" json:"externalInfoUpdatedAt"`

	Tags Tags `structs:"tags" json:"-"` // All tags for this album

	ScannedAt time.Time `structs:"scanned_at" json:"scannedAt"` // Last time this album was scanned
	CreatedAt time.Time `structs:"created_at" json:"createdAt"` // Oldest CreatedAt for all songs in this album
	UpdatedAt time.Time `structs:"updated_at" json:"updatedAt"` // Newest UpdatedAt for all songs in this album
}

func (a Album) CoverArtID() ArtworkID {
	return artworkIDFromAlbum(a)
}

type Discs map[int]string

// Add adds a disc to the Discs map. If the map is nil, it is initialized.
func (d *Discs) Add(discNumber int, discSubtitle string) {
	if *d == nil {
		*d = Discs{}
	}
	(*d)[discNumber] = discSubtitle
}

type DiscID struct {
	AlbumID     string `json:"albumId"`
	ReleaseDate string `json:"releaseDate"`
	DiscNumber  int    `json:"discNumber"`
}

type Albums []Album

// ToAlbumArtist creates an Artist object based on the attributes of this Albums collection.
// It assumes all albums have the same AlbumArtist, or else results are unpredictable.
func (als Albums) ToAlbumArtist() Artist {
	a := Artist{AlbumCount: len(als)}
	var mbzArtistIds []string
	for _, al := range als {
		a.ID = al.AlbumArtistID
		a.Name = al.AlbumArtist
		a.SortArtistName = al.SortAlbumArtistName
		a.OrderArtistName = al.OrderAlbumArtistName

		a.SongCount += al.SongCount
		a.Size += al.Size
		a.Genres = append(a.Genres, al.Genres...)
		mbzArtistIds = append(mbzArtistIds, al.MbzAlbumArtistID)
	}
	slices.SortFunc(a.Genres, func(a, b Genre) int { return cmp.Compare(a.ID, b.ID) })
	a.Genres = slices.Compact(a.Genres)
	a.MbzArtistID = slice.MostFrequent(mbzArtistIds)

	return a
}

type AlbumRepository interface {
	CountAll(...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(*Album) error
	Get(id string) (*Album, error)
	GetAll(...QueryOptions) (Albums, error)
	Touch(ids ...string) error
	GetAlbumsTouched(libID int) ([]string, error)
	Search(q string, offset int, size int) (Albums, error)
	AnnotatedRepository
}
