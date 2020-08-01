package subsonic

import (
	"context"
	"fmt"
	"mime"
	"net/http"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

func NewResponse() *responses.Subsonic {
	return &responses.Subsonic{Status: "ok", Version: Version, Type: consts.AppName, ServerVersion: consts.Version()}
}

func RequiredParamString(r *http.Request, param string, msg string) (string, error) {
	p := utils.ParamString(r, param)
	if p == "" {
		return "", NewError(responses.ErrorMissingParameter, msg)
	}
	return p, nil
}

func RequiredParamStrings(r *http.Request, param string, msg string) ([]string, error) {
	ps := utils.ParamStrings(r, param)
	if len(ps) == 0 {
		return nil, NewError(responses.ErrorMissingParameter, msg)
	}
	return ps, nil
}

func RequiredParamInt(r *http.Request, param string, msg string) (int, error) {
	p := utils.ParamString(r, param)
	if p == "" {
		return 0, NewError(responses.ErrorMissingParameter, msg)
	}
	return utils.ParamInt(r, param, 0), nil
}

type SubsonicError struct {
	code     int
	messages []interface{}
}

func NewError(code int, message ...interface{}) error {
	return SubsonicError{
		code:     code,
		messages: message,
	}
}

func (e SubsonicError) Error() string {
	var msg string
	if len(e.messages) == 0 {
		msg = responses.ErrorMsg(e.code)
	} else {
		msg = fmt.Sprintf(e.messages[0].(string), e.messages[1:]...)
	}
	return msg
}

func ToAlbums(ctx context.Context, entries engine.Entries) []responses.Child {
	children := make([]responses.Child, len(entries))
	for i, entry := range entries {
		children[i] = ToAlbum(ctx, entry)
	}
	return children
}

func ToAlbum(ctx context.Context, entry engine.Entry) responses.Child {
	album := ToChild(ctx, entry)
	album.Name = album.Title
	album.Title = ""
	album.Parent = ""
	album.Album = ""
	album.AlbumId = ""
	return album
}

func ToArtists(entries engine.Entries) []responses.Artist {
	artists := make([]responses.Artist, len(entries))
	for i, entry := range entries {
		artists[i] = responses.Artist{
			Id:         entry.Id,
			Name:       entry.Title,
			AlbumCount: entry.AlbumCount,
		}
		if !entry.Starred.IsZero() {
			artists[i].Starred = &entry.Starred
		}
	}
	return artists
}

func ToChildren(ctx context.Context, entries engine.Entries) []responses.Child {
	children := make([]responses.Child, len(entries))
	for i, entry := range entries {
		children[i] = ToChild(ctx, entry)
	}
	return children
}

func ToChild(ctx context.Context, entry engine.Entry) responses.Child {
	child := responses.Child{}
	child.Id = entry.Id
	child.Title = entry.Title
	child.IsDir = entry.IsDir
	child.Parent = entry.Parent
	child.Album = entry.Album
	child.Year = entry.Year
	child.Artist = entry.Artist
	child.Genre = entry.Genre
	child.CoverArt = entry.CoverArt
	child.Track = entry.Track
	child.Duration = entry.Duration
	child.Size = entry.Size
	child.Suffix = entry.Suffix
	child.BitRate = entry.BitRate
	child.ContentType = entry.ContentType
	if !entry.Starred.IsZero() {
		child.Starred = &entry.Starred
	}
	child.Path = entry.Path
	child.PlayCount = int64(entry.PlayCount)
	child.DiscNumber = entry.DiscNumber
	if !entry.Created.IsZero() {
		child.Created = &entry.Created
	}
	child.AlbumId = entry.AlbumId
	child.ArtistId = entry.ArtistId
	child.Type = entry.Type
	child.IsVideo = false
	child.UserRating = entry.UserRating
	child.SongCount = entry.SongCount
	format, _ := getTranscoding(ctx)
	if entry.Suffix != "" && format != "" && entry.Suffix != format {
		child.TranscodedSuffix = format
		child.TranscodedContentType = mime.TypeByExtension("." + format)
	}
	child.BookmarkPosition = entry.BookmarkPosition
	return child
}

func ToGenres(genres model.Genres) *responses.Genres {
	response := make([]responses.Genre, len(genres))
	for i, g := range genres {
		response[i] = responses.Genre(g)
	}
	return &responses.Genres{Genre: response}
}

func getTranscoding(ctx context.Context) (format string, bitRate int) {
	if trc, ok := request.TranscodingFrom(ctx); ok {
		format = trc.TargetFormat
	}
	if plr, ok := request.PlayerFrom(ctx); ok {
		bitRate = plr.MaxBitRate
	}
	return
}

// This seems to be duplicated, but it is an initial step into merging `engine` and the `subsonic` packages,
// In the future there won't be any conversion to/from `engine. Entry` anymore
func ChildFromMediaFile(ctx context.Context, mf model.MediaFile) responses.Child {
	child := responses.Child{}
	child.Id = mf.ID
	child.Title = mf.Title
	child.IsDir = false
	child.Parent = mf.AlbumID
	child.Album = mf.Album
	child.Year = mf.Year
	child.Artist = mf.Artist
	child.Genre = mf.Genre
	child.Track = mf.TrackNumber
	child.Duration = int(mf.Duration)
	child.Size = mf.Size
	child.Suffix = mf.Suffix
	child.BitRate = mf.BitRate
	if mf.HasCoverArt {
		child.CoverArt = mf.ID
	} else {
		child.CoverArt = "al-" + mf.AlbumID
	}
	child.ContentType = mf.ContentType()
	child.Path = mf.Path
	child.DiscNumber = mf.DiscNumber
	child.Created = &mf.CreatedAt
	child.AlbumId = mf.AlbumID
	child.ArtistId = mf.ArtistID
	child.Type = "music"
	child.PlayCount = mf.PlayCount
	if mf.Starred {
		child.Starred = &mf.StarredAt
	}
	child.UserRating = mf.Rating

	format, _ := getTranscoding(ctx)
	if mf.Suffix != "" && format != "" && mf.Suffix != format {
		child.TranscodedSuffix = format
		child.TranscodedContentType = mime.TypeByExtension("." + format)
	}
	child.BookmarkPosition = mf.BookmarkPosition
	return child
}

func ChildrenFromMediaFiles(ctx context.Context, mfs model.MediaFiles) []responses.Child {
	children := make([]responses.Child, len(mfs))
	for i, mf := range mfs {
		children[i] = ChildFromMediaFile(ctx, mf)
	}
	return children
}
