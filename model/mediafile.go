package model

import (
	"cmp"
	"encoding/json"
	"mime"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/navidrome/navidrome/utils/str"
)

type MediaFile struct {
	Annotations  `structs:"-"`
	Bookmarkable `structs:"-"`

	ID                   string  `structs:"id" json:"id"`
	LibraryID            int     `structs:"library_id" json:"libraryId"`
	FolderID             string  `structs:"folder_id" json:"folderId"`
	Path                 string  `structs:"path" json:"path"`
	Title                string  `structs:"title" json:"title"`
	Album                string  `structs:"album" json:"album"`
	ArtistID             string  `structs:"artist_id" json:"artistId"`
	Artist               string  `structs:"artist" json:"artist"`
	AlbumArtistID        string  `structs:"album_artist_id" json:"albumArtistId"`
	AlbumArtist          string  `structs:"album_artist" json:"albumArtist"`
	AlbumID              string  `structs:"album_id" json:"albumId"`
	AlbumPID             string  `structs:"album_pid" json:"albumPid"`
	HasCoverArt          bool    `structs:"has_cover_art" json:"hasCoverArt"`
	TrackNumber          int     `structs:"track_number" json:"trackNumber"`
	DiscNumber           int     `structs:"disc_number" json:"discNumber"`
	DiscSubtitle         string  `structs:"disc_subtitle" json:"discSubtitle,omitempty"`
	Year                 int     `structs:"year" json:"year"`
	Date                 string  `structs:"date" json:"date,omitempty"`
	OriginalYear         int     `structs:"original_year" json:"originalYear"`
	OriginalDate         string  `structs:"original_date" json:"originalDate,omitempty"`
	ReleaseYear          int     `structs:"release_year" json:"releaseYear"`
	ReleaseDate          string  `structs:"release_date" json:"releaseDate,omitempty"`
	Size                 int64   `structs:"size" json:"size"`
	Suffix               string  `structs:"suffix" json:"suffix"`
	Duration             float32 `structs:"duration" json:"duration"`
	BitRate              int     `structs:"bit_rate" json:"bitRate"`
	SampleRate           int     `structs:"sample_rate" json:"sampleRate"`
	Channels             int     `structs:"channels" json:"channels"`
	Genre                string  `structs:"genre" json:"genre"`
	Genres               Genres  `structs:"-" json:"genres,omitempty"`
	FullText             string  `structs:"full_text" json:"-"`
	SortTitle            string  `structs:"sort_title" json:"sortTitle,omitempty"`
	SortAlbumName        string  `structs:"sort_album_name" json:"sortAlbumName,omitempty"`
	SortArtistName       string  `structs:"sort_artist_name" json:"sortArtistName,omitempty"`
	SortAlbumArtistName  string  `structs:"sort_album_artist_name" json:"sortAlbumArtistName,omitempty"`
	OrderTitle           string  `structs:"order_title" json:"orderTitle,omitempty"`
	OrderAlbumName       string  `structs:"order_album_name" json:"orderAlbumName"`
	OrderArtistName      string  `structs:"order_artist_name" json:"orderArtistName"`
	OrderAlbumArtistName string  `structs:"order_album_artist_name" json:"orderAlbumArtistName"`
	Compilation          bool    `structs:"compilation" json:"compilation"`
	Comment              string  `structs:"comment" json:"comment,omitempty"`
	Lyrics               string  `structs:"lyrics" json:"lyrics"`
	Bpm                  int     `structs:"bpm" json:"bpm,omitempty"`
	CatalogNum           string  `structs:"catalog_num" json:"catalogNum,omitempty"`
	MbzRecordingID       string  `structs:"mbz_recording_id" json:"mbzRecordingID,omitempty"`
	MbzReleaseTrackID    string  `structs:"mbz_release_track_id" json:"mbzReleaseTrackId,omitempty"`
	MbzAlbumID           string  `structs:"mbz_album_id" json:"mbzAlbumId,omitempty"`
	MbzArtistID          string  `structs:"mbz_artist_id" json:"mbzArtistId,omitempty"`
	MbzAlbumArtistID     string  `structs:"mbz_album_artist_id" json:"mbzAlbumArtistId,omitempty"`
	MbzAlbumType         string  `structs:"mbz_album_type" json:"mbzAlbumType,omitempty"`
	MbzAlbumComment      string  `structs:"mbz_album_comment" json:"mbzAlbumComment,omitempty"`
	RgAlbumGain          float64 `structs:"rg_album_gain" json:"rgAlbumGain"`
	RgAlbumPeak          float64 `structs:"rg_album_peak" json:"rgAlbumPeak"`
	RgTrackGain          float64 `structs:"rg_track_gain" json:"rgTrackGain"`
	RgTrackPeak          float64 `structs:"rg_track_peak" json:"rgTrackPeak"`

	Tags Tags   `structs:"tags" json:"tags,omitempty"` // All imported tags from the original file
	TID  string `structs:"tid"  json:"tid"`            // Unique ID based on all tags

	Missing   bool      `structs:"missing" json:"missing"`      // If the file is not found in the library's FS
	CreatedAt time.Time `structs:"created_at" json:"createdAt"` // Time this entry was created in the DB
	UpdatedAt time.Time `structs:"updated_at" json:"updatedAt"` // Time of file last update (mtime)
}

func (mf MediaFile) ContentType() string {
	return mime.TypeByExtension("." + mf.Suffix)
}

func (mf MediaFile) CoverArtID() ArtworkID {
	// If it has a cover art, return it (if feature is disabled, skip)
	if mf.HasCoverArt && conf.Server.EnableMediaFileCoverArt {
		return artworkIDFromMediaFile(mf)
	}
	// if it does not have a coverArt, fallback to the album cover
	return mf.AlbumCoverArtID()
}

func (mf MediaFile) AlbumCoverArtID() ArtworkID {
	return artworkIDFromAlbum(Album{ID: mf.AlbumID})
}

func (mf MediaFile) StructuredLyrics() (LyricList, error) {
	lyrics := LyricList{}
	err := json.Unmarshal([]byte(mf.Lyrics), &lyrics)
	if err != nil {
		return nil, err
	}
	return lyrics, nil
}

type MediaFiles []MediaFile

// Dirs returns a deduped list of all directories from the MediaFiles' paths
func (mfs MediaFiles) Dirs() []string {
	var dirs []string
	for _, mf := range mfs {
		dir, _ := filepath.Split(mf.Path)
		dirs = append(dirs, filepath.Clean(dir))
	}
	slices.Sort(dirs)
	return slices.Compact(dirs)
}

// ToAlbum creates an Album object based on the attributes of this MediaFiles collection.
// It assumes all mediafiles have the same Album, or else results are unpredictable.
func (mfs MediaFiles) ToAlbum() Album {
	a := Album{SongCount: len(mfs)}
	var fullText []string
	var albumArtistIds []string
	var songArtistIds []string
	var mbzAlbumIds []string
	var comments []string
	var years []int
	var dates []string
	var originalYears []int
	var originalDates []string
	var releaseDates []string
	for _, m := range mfs {
		// We assume these attributes are all the same for all songs on an album
		a.ID = m.AlbumID
		a.PID = m.AlbumPID
		a.LibraryID = m.LibraryID
		a.Name = m.Album
		a.Artist = m.Artist
		a.ArtistID = m.ArtistID
		a.AlbumArtist = m.AlbumArtist
		a.AlbumArtistID = m.AlbumArtistID
		a.SortAlbumName = m.SortAlbumName
		a.SortArtistName = m.SortArtistName
		a.SortAlbumArtistName = m.SortAlbumArtistName
		a.OrderAlbumName = m.OrderAlbumName
		a.OrderAlbumArtistName = m.OrderAlbumArtistName
		a.MbzAlbumArtistID = m.MbzAlbumArtistID
		a.MbzAlbumType = m.MbzAlbumType
		a.MbzAlbumComment = m.MbzAlbumComment
		a.CatalogNum = m.CatalogNum
		a.Compilation = m.Compilation

		// Calculated attributes based on aggregations
		a.Duration += m.Duration
		a.Size += m.Size
		years = append(years, m.Year)
		dates = append(dates, m.Date)
		originalYears = append(originalYears, m.OriginalYear)
		originalDates = append(originalDates, m.OriginalDate)
		releaseDates = append(releaseDates, m.ReleaseDate)
		a.UpdatedAt = newer(a.UpdatedAt, m.UpdatedAt)
		a.CreatedAt = older(a.CreatedAt, m.CreatedAt)
		a.Genres = append(a.Genres, m.Genres...)
		comments = append(comments, m.Comment)
		albumArtistIds = append(albumArtistIds, m.AlbumArtistID)
		songArtistIds = append(songArtistIds, m.ArtistID)
		mbzAlbumIds = append(mbzAlbumIds, m.MbzAlbumID)
		fullText = append(fullText,
			m.Album, m.AlbumArtist, m.Artist,
			m.SortAlbumName, m.SortAlbumArtistName, m.SortArtistName,
			m.DiscSubtitle)
		if m.HasCoverArt && a.EmbedArtPath == "" {
			a.EmbedArtPath = m.Path
		}
		if m.DiscNumber > 0 {
			a.Discs.Add(m.DiscNumber, m.DiscSubtitle)
		}
	}

	a.Paths = strings.Join(mfs.Dirs(), consts.Zwsp)
	a.Date, _ = allOrNothing(dates)
	a.OriginalDate, _ = allOrNothing(originalDates)
	a.ReleaseDate, a.Releases = allOrNothing(releaseDates)
	a.MinYear, a.MaxYear = minMax(years)
	a.MinOriginalYear, a.MaxOriginalYear = minMax(originalYears)
	a.Comment, _ = allOrNothing(comments)
	a.Genre = slice.MostFrequent(a.Genres).Name
	slices.SortFunc(a.Genres, func(a, b Genre) int { return cmp.Compare(a.ID, b.ID) })
	a.Genres = slices.Compact(a.Genres)
	a.FullText = " " + str.SanitizeStrings(fullText...)
	a = fixAlbumArtist(a, albumArtistIds)
	songArtistIds = append(songArtistIds, a.AlbumArtistID, a.ArtistID)
	slices.Sort(songArtistIds)
	a.AllArtistIDs = strings.Join(slices.Compact(songArtistIds), " ")
	a.MbzAlbumID = slice.MostFrequent(mbzAlbumIds)

	return a
}

func allOrNothing(items []string) (string, int) {
	sort.Strings(items)
	items = slices.Compact(items)
	if len(items) != 1 {
		return "", len(slices.Compact(items))
	}
	return items[0], 1
}

func minMax(items []int) (int, int) {
	var mn, mx = items[0], items[0]
	for _, value := range items {
		mx = max(mx, value)
		if mn == 0 {
			mn = value
		} else if value > 0 {
			mn = min(mn, value)
		}
	}
	return mn, mx
}

func newer(t1, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}

func older(t1, t2 time.Time) time.Time {
	if t1.IsZero() {
		return t2
	}
	if t1.After(t2) {
		return t2
	}
	return t1
}

func fixAlbumArtist(a Album, albumArtistIds []string) Album {
	if !a.Compilation {
		if a.AlbumArtistID == "" {
			a.AlbumArtistID = a.ArtistID
			a.AlbumArtist = a.Artist
		}
		return a
	}

	albumArtistIds = slices.Compact(albumArtistIds)
	if len(albumArtistIds) > 1 {
		a.AlbumArtist = consts.VariousArtists
		a.AlbumArtistID = consts.VariousArtistsID
	}
	return a
}

type MediaFileRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(m *MediaFile) error
	Get(id string) (*MediaFile, error)
	GetAll(options ...QueryOptions) (MediaFiles, error)
	Search(q string, offset int, size int) (MediaFiles, error)
	Delete(id string) error
	MarkMissing(MediaFiles, bool) error

	// Queries by path to support the scanner, no Annotations or Bookmarks required in the response
	FindAllByPath(path string) (MediaFiles, error)
	FindByPath(path string) (*MediaFile, error)
	FindPathsRecursively(basePath string) ([]string, error)
	DeleteByPath(path string) (int64, error)

	AnnotatedRepository
	BookmarkableRepository
}
