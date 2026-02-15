package apimodel

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

// Song is the API representation of a media file. It decouples the API response
// from the internal model.MediaFile, allowing us to expose calculated fields,
// hide internal details, and evolve the API independently of the database schema.
type Song struct {
	ID          string `json:"id"`
	LibraryID   int    `json:"libraryId"`
	LibraryName string `json:"libraryName"`

	Title        string `json:"title"`
	Album        string `json:"album"`
	AlbumID      string `json:"albumId"`
	Artist       string `json:"artist"`
	ArtistID     string `json:"artistId"`
	AlbumArtist   string `json:"albumArtist"`
	AlbumArtistID string `json:"albumArtistId"`
	Compilation   bool   `json:"compilation"`
	TrackNumber  int    `json:"trackNumber"`
	DiscNumber   int    `json:"discNumber"`
	DiscSubtitle string `json:"discSubtitle,omitempty"`

	// Dates
	Year         int    `json:"year"`
	Date         string `json:"date,omitempty"`
	OriginalYear int    `json:"originalYear"`
	OriginalDate string `json:"originalDate,omitempty"`
	ReleaseYear  int    `json:"releaseYear"`
	ReleaseDate  string `json:"releaseDate,omitempty"`

	// Audio properties
	Duration   float32 `json:"duration"`
	Size       int64   `json:"size"`
	Suffix     string  `json:"suffix"`
	BitRate    int     `json:"bitRate"`
	SampleRate int     `json:"sampleRate"`
	BitDepth   int     `json:"bitDepth"`
	Channels   int     `json:"channels"`

	// Metadata
	Genre          string            `json:"genre"`
	Genres         model.Genres      `json:"genres,omitempty"`
	Comment        string            `json:"comment,omitempty"`
	BPM            int               `json:"bpm,omitempty"`
	ExplicitStatus string            `json:"explicitStatus"`
	CatalogNum     string            `json:"catalogNum,omitempty"`
	Tags           model.Tags        `json:"tags,omitempty"`
	Participants   model.Participants `json:"participants"`

	// Sort fields
	SortTitle           string `json:"sortTitle,omitempty"`
	SortAlbumName       string `json:"sortAlbumName,omitempty"`
	SortArtistName      string `json:"sortArtistName,omitempty"`
	SortAlbumArtistName string `json:"sortAlbumArtistName,omitempty"`

	// MusicBrainz IDs
	MbzRecordingID    string `json:"mbzRecordingID,omitempty"`
	MbzReleaseTrackID string `json:"mbzReleaseTrackId,omitempty"`
	MbzAlbumID        string `json:"mbzAlbumId,omitempty"`
	MbzReleaseGroupID string `json:"mbzReleaseGroupId,omitempty"`
	MbzArtistID       string `json:"mbzArtistId,omitempty"`
	MbzAlbumArtistID  string `json:"mbzAlbumArtistId,omitempty"`
	MbzAlbumType      string `json:"mbzAlbumType,omitempty"`
	MbzAlbumComment   string `json:"mbzAlbumComment,omitempty"`

	// ReplayGain
	RGAlbumGain *float64 `json:"rgAlbumGain"`
	RGAlbumPeak *float64 `json:"rgAlbumPeak"`
	RGTrackGain *float64 `json:"rgTrackGain"`
	RGTrackPeak *float64 `json:"rgTrackPeak"`

	// Lyrics
	Lyrics string `json:"lyrics"`

	HasCoverArt bool `json:"hasCoverArt"`

	// User annotations
	PlayCount int64      `json:"playCount,omitempty"`
	PlayDate  *time.Time `json:"playDate,omitempty"`
	Rating    int        `json:"rating,omitempty"`
	Starred   bool       `json:"starred,omitempty"`
	StarredAt *time.Time `json:"starredAt,omitempty"`

	// Bookmark
	BookmarkPosition int64 `json:"bookmarkPosition"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromMediaFile converts a model.MediaFile into the API Song representation.
func FromMediaFile(mf model.MediaFile) Song {
	return Song{
		ID:          mf.ID,
		LibraryID:   mf.LibraryID,
		LibraryName: mf.LibraryName,

		Title:        mf.Title,
		Album:        mf.Album,
		AlbumID:      mf.AlbumID,
		Artist:       mf.Artist,
		ArtistID:     mf.ArtistID,
		AlbumArtist:   mf.AlbumArtist,
		AlbumArtistID: mf.AlbumArtistID,
		Compilation:   mf.Compilation,
		TrackNumber:  mf.TrackNumber,
		DiscNumber:   mf.DiscNumber,
		DiscSubtitle: mf.DiscSubtitle,

		Year:         mf.Year,
		Date:         mf.Date,
		OriginalYear: mf.OriginalYear,
		OriginalDate: mf.OriginalDate,
		ReleaseYear:  mf.ReleaseYear,
		ReleaseDate:  mf.ReleaseDate,

		Duration:   mf.Duration,
		Size:       mf.Size,
		Suffix:     mf.Suffix,
		BitRate:    mf.BitRate,
		SampleRate: mf.SampleRate,
		BitDepth:   mf.BitDepth,
		Channels:   mf.Channels,

		Genre:          mf.Genre,
		Genres:         mf.Genres,
		Comment:        mf.Comment,
		BPM:            mf.BPM,
		ExplicitStatus: mf.ExplicitStatus,
		CatalogNum:     mf.CatalogNum,
		Tags:           mf.Tags,
		Participants:   mf.Participants,

		SortTitle:           mf.SortTitle,
		SortAlbumName:       mf.SortAlbumName,
		SortArtistName:      mf.SortArtistName,
		SortAlbumArtistName: mf.SortAlbumArtistName,

		MbzRecordingID:    mf.MbzRecordingID,
		MbzReleaseTrackID: mf.MbzReleaseTrackID,
		MbzAlbumID:        mf.MbzAlbumID,
		MbzReleaseGroupID: mf.MbzReleaseGroupID,
		MbzArtistID:       mf.MbzArtistID,
		MbzAlbumArtistID:  mf.MbzAlbumArtistID,
		MbzAlbumType:      mf.MbzAlbumType,
		MbzAlbumComment:   mf.MbzAlbumComment,

		RGAlbumGain: mf.RGAlbumGain,
		RGAlbumPeak: mf.RGAlbumPeak,
		RGTrackGain: mf.RGTrackGain,
		RGTrackPeak: mf.RGTrackPeak,

		Lyrics: mf.Lyrics,

		HasCoverArt: mf.HasCoverArt,

		PlayCount: mf.PlayCount,
		PlayDate:  mf.PlayDate,
		Rating:    mf.Rating,
		Starred:   mf.Starred,
		StarredAt: mf.StarredAt,

		BookmarkPosition: mf.BookmarkPosition,

		CreatedAt: mf.CreatedAt,
		UpdatedAt: mf.UpdatedAt,
	}
}

// FromMediaFiles converts a slice of model.MediaFile into a slice of API Songs.
func FromMediaFiles(mfs model.MediaFiles) []Song {
	songs := make([]Song, len(mfs))
	for i, mf := range mfs {
		songs[i] = FromMediaFile(mf)
	}
	return songs
}
