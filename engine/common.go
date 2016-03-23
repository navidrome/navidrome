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
	ErrDataNotFound = errors.New("data not found")
)

func FromAlbum(al *domain.Album) Entry {
	e := Entry{}
	e.Id = al.Id
	e.Title = al.Name
	e.IsDir = true
	e.Parent = al.ArtistId
	e.Album = al.Name
	e.Year = al.Year
	e.Artist = al.AlbumArtist
	e.Genre = al.Genre
	e.CoverArt = al.CoverArtId
	e.Starred = al.StarredAt
	e.PlayCount = int32(al.PlayCount)
	e.Created = al.CreatedAt
	e.AlbumId = al.Id
	e.ArtistId = al.ArtistId
	e.UserRating = al.Rating
	e.Duration = al.Duration
	return e
}

func FromMediaFile(mf *domain.MediaFile) Entry {
	e := Entry{}
	e.Id = mf.Id
	e.Title = mf.Title
	e.IsDir = false
	e.Parent = mf.AlbumId
	e.Album = mf.Album
	e.Year = mf.Year
	e.Artist = mf.Artist
	e.Genre = mf.Genre
	e.Track = mf.TrackNumber
	e.Duration = mf.Duration
	e.Size = mf.Size
	e.Suffix = mf.Suffix
	e.BitRate = mf.BitRate
	e.Starred = mf.StarredAt
	if mf.HasCoverArt {
		e.CoverArt = mf.Id
	}
	e.ContentType = mf.ContentType()
	e.Path = mf.Path
	e.PlayCount = int32(mf.PlayCount)
	e.DiscNumber = mf.DiscNumber
	e.Created = mf.CreatedAt
	e.AlbumId = mf.AlbumId
	e.ArtistId = mf.ArtistId
	e.Type = "music" // TODO Hardcoded for now
	e.UserRating = mf.Rating
	return e
}

func FromAlbums(albums domain.Albums) Entries {
	entries := make(Entries, len(albums))
	for i, al := range albums {
		entries[i] = FromAlbum(&al)
	}
	return entries
}
