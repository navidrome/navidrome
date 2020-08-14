package engine

import (
	"fmt"
	"time"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/model"
)

type Entry struct {
	Id               string
	Title            string
	IsDir            bool
	Parent           string
	Album            string
	Year             int
	Artist           string
	Genre            string
	CoverArt         string
	Starred          time.Time
	Track            int
	Duration         int
	Size             int64
	Suffix           string
	BitRate          int
	ContentType      string
	Path             string
	PlayCount        int32
	DiscNumber       int
	Created          time.Time
	AlbumId          string
	ArtistId         string
	Type             string
	UserRating       int
	SongCount        int
	UserName         string
	MinutesAgo       int
	PlayerId         int
	PlayerName       string
	AlbumCount       int
	BookmarkPosition int64
	AbsolutePath     string
}

type Entries []Entry

func FromArtist(ar *model.Artist) Entry {
	e := Entry{}
	e.Id = ar.ID
	e.Title = ar.Name
	e.AlbumCount = ar.AlbumCount
	e.IsDir = true
	if ar.Starred {
		e.Starred = ar.StarredAt
	}
	return e
}

func FromAlbum(al *model.Album) Entry {
	e := Entry{}
	e.Id = al.ID
	e.Title = al.Name
	e.IsDir = true
	e.Parent = al.AlbumArtistID
	e.Album = al.Name
	e.Year = al.MaxYear
	e.Artist = al.AlbumArtist
	e.Genre = al.Genre
	e.CoverArt = al.CoverArtId
	e.Created = al.CreatedAt
	e.AlbumId = al.ID
	e.ArtistId = al.AlbumArtistID
	e.Duration = int(al.Duration)
	e.SongCount = al.SongCount
	if al.Starred {
		e.Starred = al.StarredAt
	}
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
	e.Duration = int(mf.Duration)
	e.Size = mf.Size
	e.Suffix = mf.Suffix
	e.BitRate = mf.BitRate
	if mf.HasCoverArt {
		e.CoverArt = mf.ID
	} else {
		e.CoverArt = "al-" + mf.AlbumID
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
	e.Type = "music"
	e.PlayCount = int32(mf.PlayCount)
	if mf.Starred {
		e.Starred = mf.StarredAt
	}
	e.UserRating = mf.Rating
	e.BookmarkPosition = mf.BookmarkPosition
	return e
}

func realArtistName(mf *model.MediaFile) string {
	switch {
	case mf.Compilation:
		return consts.VariousArtists
	case mf.AlbumArtist != "":
		return mf.AlbumArtist
	}

	return mf.Artist
}

func FromAlbums(albums model.Albums) Entries {
	entries := make(Entries, len(albums))
	for i := range albums {
		al := albums[i]
		entries[i] = FromAlbum(&al)
	}
	return entries
}

func FromMediaFiles(mfs model.MediaFiles) Entries {
	entries := make(Entries, len(mfs))
	for i := range mfs {
		mf := mfs[i]
		entries[i] = FromMediaFile(&mf)
	}
	return entries
}

func FromArtists(ars model.Artists) Entries {
	entries := make(Entries, len(ars))
	for i := range ars {
		ar := ars[i]
		entries[i] = FromArtist(&ar)
	}
	return entries
}
