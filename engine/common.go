package engine

import (
	"fmt"
	"time"

	"github.com/deluan/navidrome/model"
)

type Entry struct {
	Id          string
	Title       string
	IsDir       bool
	Parent      string
	Album       string
	Year        int
	Artist      string
	Genre       string
	CoverArt    string
	Starred     time.Time
	Track       int
	Duration    int
	Size        int
	Suffix      string
	BitRate     int
	ContentType string
	Path        string
	PlayCount   int32
	DiscNumber  int
	Created     time.Time
	AlbumId     string
	ArtistId    string
	Type        string
	UserRating  int
	SongCount   int

	UserName   string
	MinutesAgo int
	PlayerId   int
	PlayerName string
	AlbumCount int

	AbsolutePath string
}

type Entries []Entry

func FromArtist(ar *model.Artist) Entry {
	e := Entry{}
	e.Id = ar.ID
	e.Title = ar.Name
	e.AlbumCount = ar.AlbumCount
	e.IsDir = true
	e.Starred = ar.StarredAt
	return e
}

func FromAlbum(al *model.Album) Entry {
	e := Entry{}
	e.Id = al.ID
	e.Title = al.Name
	e.IsDir = true
	e.Parent = al.ArtistID
	e.Album = al.Name
	e.Year = al.Year
	e.Artist = al.AlbumArtist
	e.Genre = al.Genre
	e.CoverArt = al.CoverArtId
	e.Created = al.CreatedAt
	e.AlbumId = al.ID
	e.ArtistId = al.ArtistID
	e.Duration = al.Duration
	e.SongCount = al.SongCount
	e.Starred = al.StarredAt
	e.PlayCount = int32(al.PlayCount)
	e.UserRating = al.Rating
	return e
}

func FromMediaFile(mf *model.MediaFile) Entry {
	e := Entry{}
	e.Id = mf.ID
	e.Title = mf.Title
	e.IsDir = false
	e.Parent = mf.AlbumID
	e.Album = mf.Album
	e.Year = mf.Year
	e.Artist = mf.Artist
	e.Genre = mf.Genre
	e.Track = mf.TrackNumber
	e.Duration = mf.Duration
	e.Size = mf.Size
	e.Suffix = mf.Suffix
	e.BitRate = mf.BitRate
	if mf.HasCoverArt {
		e.CoverArt = mf.ID
	}
	e.ContentType = mf.ContentType()
	e.AbsolutePath = mf.Path
	// Creates a "pseudo" Path, to avoid sending absolute paths to the client
	if mf.Path != "" {
		e.Path = fmt.Sprintf("%s/%s/%s.%s", realArtistName(mf), mf.Album, mf.Title, mf.Suffix)
	}
	e.DiscNumber = mf.DiscNumber
	e.Created = mf.CreatedAt
	e.AlbumId = mf.AlbumID
	e.ArtistId = mf.ArtistID
	e.Type = "music" // TODO Hardcoded for now
	e.PlayCount = int32(mf.PlayCount)
	e.Starred = mf.StarredAt
	e.UserRating = mf.Rating
	return e
}

func realArtistName(mf *model.MediaFile) string {
	switch {
	case mf.Compilation:
		return "Various Artists"
	case mf.AlbumArtist != "":
		return mf.AlbumArtist
	}

	return mf.Artist
}

func FromAlbums(albums model.Albums) Entries {
	entries := make(Entries, len(albums))
	for i, al := range albums {
		entries[i] = FromAlbum(&al)
	}
	return entries
}

func FromMediaFiles(mfs model.MediaFiles) Entries {
	entries := make(Entries, len(mfs))
	for i, mf := range mfs {
		entries[i] = FromMediaFile(&mf)
	}
	return entries
}

func FromArtists(ars model.Artists) Entries {
	entries := make(Entries, len(ars))
	for i, ar := range ars {
		entries[i] = FromArtist(&ar)
	}
	return entries
}
