package model

import (
	"cmp"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"iter"
	"mime"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gohugoio/hashstructure"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/slice"
)

type MediaFile struct {
	Annotations  `structs:"-" hash:"ignore"`
	Bookmarkable `structs:"-" hash:"ignore"`

	ID          string `structs:"id"  json:"id" hash:"ignore"`
	PID         string `structs:"pid" json:"-" hash:"ignore"`
	LibraryID   int    `structs:"library_id" json:"libraryId" hash:"ignore"`
	LibraryPath string `structs:"-" json:"libraryPath" hash:"ignore"`
	LibraryName string `structs:"-" json:"libraryName" hash:"ignore"`
	FolderID    string `structs:"folder_id" json:"folderId" hash:"ignore"`
	Path        string `structs:"path" json:"path" hash:"ignore"`
	Title       string `structs:"title" json:"title"`
	Album       string `structs:"album" json:"album"`
	ArtistID    string `structs:"artist_id" json:"artistId"` // Deprecated: Use Participants instead
	// Artist is the display name used for the artist.
	Artist        string `structs:"artist" json:"artist"`
	AlbumArtistID string `structs:"album_artist_id" json:"albumArtistId"` // Deprecated: Use Participants instead
	// AlbumArtist is the display name used for the album artist.
	AlbumArtist          string   `structs:"album_artist" json:"albumArtist"`
	AlbumID              string   `structs:"album_id" json:"albumId"`
	HasCoverArt          bool     `structs:"has_cover_art" json:"hasCoverArt"`
	TrackNumber          int      `structs:"track_number" json:"trackNumber"`
	DiscNumber           int      `structs:"disc_number" json:"discNumber"`
	DiscSubtitle         string   `structs:"disc_subtitle" json:"discSubtitle,omitempty"`
	Year                 int      `structs:"year" json:"year"`
	Date                 string   `structs:"date" json:"date,omitempty"`
	OriginalYear         int      `structs:"original_year" json:"originalYear"`
	OriginalDate         string   `structs:"original_date" json:"originalDate,omitempty"`
	ReleaseYear          int      `structs:"release_year" json:"releaseYear"`
	ReleaseDate          string   `structs:"release_date" json:"releaseDate,omitempty"`
	Size                 int64    `structs:"size" json:"size"`
	Suffix               string   `structs:"suffix" json:"suffix"`
	Duration             float32  `structs:"duration" json:"duration"`
	BitRate              int      `structs:"bit_rate" json:"bitRate"`
	SampleRate           int      `structs:"sample_rate" json:"sampleRate"`
	BitDepth             int      `structs:"bit_depth" json:"bitDepth"`
	Channels             int      `structs:"channels" json:"channels"`
	Genre                string   `structs:"genre" json:"genre"`
	Genres               Genres   `structs:"-" json:"genres,omitempty"`
	SortTitle            string   `structs:"sort_title" json:"sortTitle,omitempty"`
	SortAlbumName        string   `structs:"sort_album_name" json:"sortAlbumName,omitempty"`
	SortArtistName       string   `structs:"sort_artist_name" json:"sortArtistName,omitempty"`            // Deprecated: Use Participants instead
	SortAlbumArtistName  string   `structs:"sort_album_artist_name" json:"sortAlbumArtistName,omitempty"` // Deprecated: Use Participants instead
	OrderTitle           string   `structs:"order_title" json:"orderTitle,omitempty"`
	OrderAlbumName       string   `structs:"order_album_name" json:"orderAlbumName"`
	OrderArtistName      string   `structs:"order_artist_name" json:"orderArtistName"`            // Deprecated: Use Participants instead
	OrderAlbumArtistName string   `structs:"order_album_artist_name" json:"orderAlbumArtistName"` // Deprecated: Use Participants instead
	Compilation          bool     `structs:"compilation" json:"compilation"`
	Comment              string   `structs:"comment" json:"comment,omitempty"`
	Lyrics               string   `structs:"lyrics" json:"lyrics"`
	BPM                  int      `structs:"bpm" json:"bpm,omitempty"`
	ExplicitStatus       string   `structs:"explicit_status" json:"explicitStatus"`
	CatalogNum           string   `structs:"catalog_num" json:"catalogNum,omitempty"`
	MbzRecordingID       string   `structs:"mbz_recording_id" json:"mbzRecordingID,omitempty"`
	MbzReleaseTrackID    string   `structs:"mbz_release_track_id" json:"mbzReleaseTrackId,omitempty"`
	MbzAlbumID           string   `structs:"mbz_album_id" json:"mbzAlbumId,omitempty"`
	MbzReleaseGroupID    string   `structs:"mbz_release_group_id" json:"mbzReleaseGroupId,omitempty"`
	MbzArtistID          string   `structs:"mbz_artist_id" json:"mbzArtistId,omitempty"`            // Deprecated: Use Participants instead
	MbzAlbumArtistID     string   `structs:"mbz_album_artist_id" json:"mbzAlbumArtistId,omitempty"` // Deprecated: Use Participants instead
	MbzAlbumType         string   `structs:"mbz_album_type" json:"mbzAlbumType,omitempty"`
	MbzAlbumComment      string   `structs:"mbz_album_comment" json:"mbzAlbumComment,omitempty"`
	RGAlbumGain          *float64 `structs:"rg_album_gain" json:"rgAlbumGain"`
	RGAlbumPeak          *float64 `structs:"rg_album_peak" json:"rgAlbumPeak"`
	RGTrackGain          *float64 `structs:"rg_track_gain" json:"rgTrackGain"`
	RGTrackPeak          *float64 `structs:"rg_track_peak" json:"rgTrackPeak"`

	Tags         Tags         `structs:"tags" json:"tags,omitempty" hash:"ignore"`       // All imported tags from the original file
	Participants Participants `structs:"participants" json:"participants" hash:"ignore"` // All artists that participated in this track

	Missing   bool      `structs:"missing" json:"missing" hash:"ignore"`      // If the file is not found in the library's FS
	BirthTime time.Time `structs:"birth_time" json:"birthTime" hash:"ignore"` // Time of file creation (ctime)
	CreatedAt time.Time `structs:"created_at" json:"createdAt" hash:"ignore"` // Time this entry was created in the DB
	UpdatedAt time.Time `structs:"updated_at" json:"updatedAt" hash:"ignore"` // Time of file last update (mtime)
}

func (mf MediaFile) FullTitle() string {
	if conf.Server.Subsonic.AppendSubtitle && mf.Tags[TagSubtitle] != nil {
		return fmt.Sprintf("%s (%s)", mf.Title, mf.Tags[TagSubtitle][0])
	}
	return mf.Title
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

// String is mainly used for debugging
func (mf MediaFile) String() string {
	return mf.Path
}

// Hash returns a hash of the MediaFile based on its tags and audio properties
func (mf MediaFile) Hash() string {
	opts := &hashstructure.HashOptions{
		IgnoreZeroValue: true,
		ZeroNil:         true,
	}
	hash, _ := hashstructure.Hash(mf, opts)
	sum := md5.New()
	sum.Write([]byte(fmt.Sprintf("%d", hash)))
	sum.Write(mf.Tags.Hash())
	sum.Write(mf.Participants.Hash())
	return fmt.Sprintf("%x", sum.Sum(nil))
}

// Equals compares two MediaFiles by their hash. It does not consider the ID, PID, Path and other identifier fields.
// Check the structure for the fields that are marked with `hash:"ignore"`.
func (mf MediaFile) Equals(other MediaFile) bool {
	return mf.Hash() == other.Hash()
}

// IsEquivalent compares two MediaFiles by path only. Used for matching missing tracks.
func (mf MediaFile) IsEquivalent(other MediaFile) bool {
	return utils.BaseName(mf.Path) == utils.BaseName(other.Path)
}

func (mf MediaFile) AbsolutePath() string {
	return filepath.Join(mf.LibraryPath, mf.Path)
}

type MediaFiles []MediaFile

// ToAlbum creates an Album object based on the attributes of this MediaFiles collection.
// It assumes all mediafiles have the same Album (same ID), or else results are unpredictable.
func (mfs MediaFiles) ToAlbum() Album {
	if len(mfs) == 0 {
		return Album{}
	}
	a := Album{SongCount: len(mfs), Tags: make(Tags), Participants: make(Participants), Discs: Discs{1: ""}}

	// Sorting the mediafiles ensure the results will be consistent
	slices.SortFunc(mfs, func(a, b MediaFile) int { return cmp.Compare(a.Path, b.Path) })

	mbzAlbumIds := make([]string, 0, len(mfs))
	mbzReleaseGroupIds := make([]string, 0, len(mfs))
	comments := make([]string, 0, len(mfs))
	years := make([]int, 0, len(mfs))
	dates := make([]string, 0, len(mfs))
	originalYears := make([]int, 0, len(mfs))
	originalDates := make([]string, 0, len(mfs))
	releaseDates := make([]string, 0, len(mfs))
	tags := make(TagList, 0, len(mfs[0].Tags)*len(mfs))

	a.Missing = true
	embedArtPath := ""
	embedArtDisc := 0
	for _, m := range mfs {
		// We assume these attributes are all the same for all songs in an album
		a.ID = m.AlbumID
		a.LibraryID = m.LibraryID
		a.Name = m.Album
		a.AlbumArtist = m.AlbumArtist
		a.AlbumArtistID = m.AlbumArtistID
		a.SortAlbumName = m.SortAlbumName
		a.SortAlbumArtistName = m.SortAlbumArtistName
		a.OrderAlbumName = m.OrderAlbumName
		a.OrderAlbumArtistName = m.OrderAlbumArtistName
		a.MbzAlbumArtistID = m.MbzAlbumArtistID
		a.MbzAlbumType = m.MbzAlbumType
		a.MbzAlbumComment = m.MbzAlbumComment
		a.CatalogNum = m.CatalogNum
		a.Compilation = a.Compilation || m.Compilation

		// Calculated attributes based on aggregations
		a.Duration += m.Duration
		a.Size += m.Size
		years = append(years, m.Year)
		dates = append(dates, m.Date)
		originalYears = append(originalYears, m.OriginalYear)
		originalDates = append(originalDates, m.OriginalDate)
		releaseDates = append(releaseDates, m.ReleaseDate)
		comments = append(comments, m.Comment)
		mbzAlbumIds = append(mbzAlbumIds, m.MbzAlbumID)
		mbzReleaseGroupIds = append(mbzReleaseGroupIds, m.MbzReleaseGroupID)
		if m.DiscNumber > 0 {
			a.Discs.Add(m.DiscNumber, m.DiscSubtitle)
		}
		tags = append(tags, m.Tags.FlattenAll()...)
		a.Participants.Merge(m.Participants)

		// Find the MediaFile with cover art and the lowest disc number to use for album cover
		embedArtPath, embedArtDisc = firstArtPath(embedArtPath, embedArtDisc, m)

		if m.ExplicitStatus == "c" && a.ExplicitStatus != "e" {
			a.ExplicitStatus = "c"
		} else if m.ExplicitStatus == "e" {
			a.ExplicitStatus = "e"
		}

		a.UpdatedAt = newer(a.UpdatedAt, m.UpdatedAt)
		a.CreatedAt = older(a.CreatedAt, m.BirthTime)
		a.Missing = a.Missing && m.Missing
	}

	a.EmbedArtPath = embedArtPath
	a.SetTags(tags)
	a.FolderIDs = slice.Unique(slice.Map(mfs, func(m MediaFile) string { return m.FolderID }))
	a.Date, _ = allOrNothing(dates)
	a.OriginalDate, _ = allOrNothing(originalDates)
	a.ReleaseDate, _ = allOrNothing(releaseDates)
	a.MinYear, a.MaxYear = minMax(years)
	a.MinOriginalYear, a.MaxOriginalYear = minMax(originalYears)
	a.Comment, _ = allOrNothing(comments)
	a.MbzAlbumID = slice.MostFrequent(mbzAlbumIds)
	a.MbzReleaseGroupID = slice.MostFrequent(mbzReleaseGroupIds)
	fixAlbumArtist(&a)

	return a
}

func allOrNothing(items []string) (string, int) {
	if len(items) == 0 {
		return "", 0
	}
	items = slice.Unique(items)
	if len(items) != 1 {
		return "", len(items)
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

// fixAlbumArtist sets the AlbumArtist to "Various Artists" if the album has more than one artist
// or if it is a compilation
func fixAlbumArtist(a *Album) {
	if !a.Compilation {
		if a.AlbumArtistID == "" {
			artist := a.Participants.First(RoleArtist)
			a.AlbumArtistID = artist.ID
			a.AlbumArtist = artist.Name
		}
		return
	}
	albumArtistIds := slice.Map(a.Participants[RoleAlbumArtist], func(p Participant) string { return p.ID })
	if len(slice.Unique(albumArtistIds)) > 1 {
		a.AlbumArtist = consts.VariousArtists
		a.AlbumArtistID = consts.VariousArtistsID
	}
}

// firstArtPath determines which media file path should be used for album artwork
// based on disc number (preferring lower disc numbers) and path (for consistency)
func firstArtPath(currentPath string, currentDisc int, m MediaFile) (string, int) {
	if !m.HasCoverArt {
		return currentPath, currentDisc
	}

	// If current has no disc number (currentDisc == 0) or new file has lower disc number
	if currentDisc == 0 || (m.DiscNumber < currentDisc && m.DiscNumber > 0) {
		return m.Path, m.DiscNumber
	}

	// If disc numbers are equal, use path for ordering
	if m.DiscNumber == currentDisc {
		if m.Path < currentPath || currentPath == "" {
			return m.Path, m.DiscNumber
		}
	}

	return currentPath, currentDisc
}

// ToM3U8 exports the playlist to the Extended M3U8 format, as specified in
// https://docs.fileformat.com/audio/m3u/#extended-m3u
func (mfs MediaFiles) ToM3U8(title string, absolutePaths bool) string {
	buf := strings.Builder{}
	buf.WriteString("#EXTM3U\n")
	buf.WriteString(fmt.Sprintf("#PLAYLIST:%s\n", title))
	for _, t := range mfs {
		buf.WriteString(fmt.Sprintf("#EXTINF:%.f,%s - %s\n", t.Duration, t.Artist, t.Title))
		if absolutePaths {
			buf.WriteString(t.AbsolutePath() + "\n")
		} else {
			buf.WriteString(t.Path + "\n")
		}
	}
	return buf.String()
}

type MediaFileCursor iter.Seq2[MediaFile, error]

type MediaFileRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(m *MediaFile) error
	Get(id string) (*MediaFile, error)
	GetWithParticipants(id string) (*MediaFile, error)
	GetAll(options ...QueryOptions) (MediaFiles, error)
	GetCursor(options ...QueryOptions) (MediaFileCursor, error)
	Delete(id string) error
	DeleteMissing(ids []string) error
	DeleteAllMissing() (int64, error)
	FindByPaths(paths []string) (MediaFiles, error)

	// The following methods are used exclusively by the scanner:
	MarkMissing(bool, ...*MediaFile) error
	MarkMissingByFolder(missing bool, folderIDs ...string) error
	GetMissingAndMatching(libId int) (MediaFileCursor, error)
	FindRecentFilesByMBZTrackID(missing MediaFile, since time.Time) (MediaFiles, error)
	FindRecentFilesByProperties(missing MediaFile, since time.Time) (MediaFiles, error)

	AnnotatedRepository
	BookmarkableRepository
	SearchableRepository[MediaFiles]
}
