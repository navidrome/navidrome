package subsonic

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"strconv"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/model"
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
	child.Size = strconv.Itoa(entry.Size)
	child.Suffix = entry.Suffix
	child.BitRate = entry.BitRate
	child.ContentType = entry.ContentType
	if !entry.Starred.IsZero() {
		child.Starred = &entry.Starred
	}
	child.Path = entry.Path
	child.PlayCount = entry.PlayCount
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
	if trc, ok := ctx.Value("transcoding").(model.Transcoding); ok {
		format = trc.TargetFormat
	}
	if plr, ok := ctx.Value("player").(model.Player); ok {
		bitRate = plr.MaxBitRate
	}
	return
}
