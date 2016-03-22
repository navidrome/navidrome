package engine

import (
	"errors"
	"time"

	"github.com/deluan/gosonic/domain"
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
	Size        string
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

	UserName   string
	MinutesAgo int
	PlayerId   int
	PlayerName string
}

type Entries []Entry

var (
	ErrDataNotFound = errors.New("Data Not Found")
)

func FromAlbum(al *domain.Album) Entry {
	c := Entry{}
	c.Id = al.Id
	c.Title = al.Name
	c.IsDir = true
	c.Parent = al.ArtistId
	c.Album = al.Name
	c.Year = al.Year
	c.Artist = al.AlbumArtist
	c.Genre = al.Genre
	c.CoverArt = al.CoverArtId
	if al.Starred {
		c.Starred = al.UpdatedAt
	}
	c.PlayCount = int32(al.PlayCount)
	c.Created = al.CreatedAt
	c.AlbumId = al.Id
	c.ArtistId = al.ArtistId
	c.UserRating = al.Rating
	c.Duration = al.Duration
	return c
}

func FromMediaFile(mf *domain.MediaFile) Entry {
	c := Entry{}
	c.Id = mf.Id
	c.Title = mf.Title
	c.IsDir = false
	c.Parent = mf.AlbumId
	c.Album = mf.Album
	c.Year = mf.Year
	c.Artist = mf.Artist
	c.Genre = mf.Genre
	c.Track = mf.TrackNumber
	c.Duration = mf.Duration
	c.Size = mf.Size
	c.Suffix = mf.Suffix
	c.BitRate = mf.BitRate
	if mf.Starred {
		c.Starred = mf.UpdatedAt
	}
	if mf.HasCoverArt {
		c.CoverArt = mf.Id
	}
	c.ContentType = mf.ContentType()
	c.Path = mf.Path
	c.PlayCount = int32(mf.PlayCount)
	c.DiscNumber = mf.DiscNumber
	c.Created = mf.CreatedAt
	c.AlbumId = mf.AlbumId
	c.ArtistId = mf.ArtistId
	c.Type = "music" // TODO Hardcoded for now
	c.UserRating = mf.Rating
	return c
}

func FromAlbums(albums domain.Albums) Entries {
	entries := make(Entries, len(albums))
	for i, al := range albums {
		entries[i] = FromAlbum(&al)
	}
	return entries
}
