package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/model"
)

type Entry struct {
	IsDir       bool
	PlayCount   int32
	Year        int
	Track       int
	Duration    int
	Size        int
	BitRate     int
	DiscNumber  int
	UserRating  int
	SongCount   int
	MinutesAgo  int
	PlayerId    int
	AlbumCount  int
	Starred     time.Time
	Created     time.Time
	Id          string
	Title       string
	Parent      string
	Album       string
	Artist      string
	Genre       string
	CoverArt    string
	Suffix      string
	ContentType string
	Path        string
	AlbumId     string
	ArtistId    string
	Type        string
	PlayerName  string
	UserName    string

	AbsolutePath string
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

func userName(ctx context.Context) string {
	user := ctx.Value("user")
	if user == nil {
		return "UNKNOWN"
	}
	usr := user.(model.User)
	return usr.UserName
}
