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

func FromMediaFiles(mfs model.MediaFiles) Entries {
	entries := make(Entries, len(mfs))
	for i := range mfs {
		mf := mfs[i]
		entries[i] = FromMediaFile(&mf)
	}
	return entries
}
