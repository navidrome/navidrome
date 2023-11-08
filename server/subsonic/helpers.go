package subsonic

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/public"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
)

func newResponse() *responses.Subsonic {
	return &responses.Subsonic{
		Status:        "ok",
		Version:       Version,
		Type:          consts.AppName,
		ServerVersion: consts.Version,
		OpenSubsonic:  true,
	}
}

func requiredParamString(r *http.Request, param string) (string, error) {
	p := utils.ParamString(r, param)
	if p == "" {
		return "", newError(responses.ErrorMissingParameter, "required '%s' parameter is missing", param)
	}
	return p, nil
}

func requiredParamStrings(r *http.Request, param string) ([]string, error) {
	ps := utils.ParamStrings(r, param)
	if len(ps) == 0 {
		return nil, newError(responses.ErrorMissingParameter, "required '%s' parameter is missing", param)
	}
	return ps, nil
}

func requiredParamInt(r *http.Request, param string) (int, error) {
	p := utils.ParamString(r, param)
	if p == "" {
		return 0, newError(responses.ErrorMissingParameter, "required '%s' parameter is missing", param)
	}
	return utils.ParamInt(r, param, 0), nil
}

type subError struct {
	code     int
	messages []interface{}
}

func newError(code int, message ...interface{}) error {
	return subError{
		code:     code,
		messages: message,
	}
}

func (e subError) Error() string {
	var msg string
	if len(e.messages) == 0 {
		msg = responses.ErrorMsg(e.code)
	} else {
		msg = fmt.Sprintf(e.messages[0].(string), e.messages[1:]...)
	}
	return msg
}

func getUser(ctx context.Context) model.User {
	user, ok := request.UserFrom(ctx)
	if ok {
		return user
	}
	return model.User{}
}

func toArtists(r *http.Request, artists model.Artists) []responses.Artist {
	as := make([]responses.Artist, len(artists))
	for i, artist := range artists {
		as[i] = toArtist(r, artist)
	}
	return as
}

func toArtist(r *http.Request, a model.Artist) responses.Artist {
	artist := responses.Artist{
		Id:             a.ID,
		Name:           a.Name,
		AlbumCount:     int32(a.AlbumCount),
		UserRating:     int32(a.Rating),
		CoverArt:       a.CoverArtID().String(),
		ArtistImageUrl: public.ImageURL(r, a.CoverArtID(), 600),
	}
	if a.Starred {
		artist.Starred = &a.StarredAt
	}
	return artist
}

func toArtistID3(r *http.Request, a model.Artist) responses.ArtistID3 {
	artist := responses.ArtistID3{
		Id:             a.ID,
		Name:           a.Name,
		AlbumCount:     int32(a.AlbumCount),
		CoverArt:       a.CoverArtID().String(),
		ArtistImageUrl: public.ImageURL(r, a.CoverArtID(), 600),
		UserRating:     int32(a.Rating),
		MusicBrainzId:  a.MbzArtistID,
	}
	if a.Starred {
		artist.Starred = &a.StarredAt
	}
	return artist
}

func toGenres(genres model.Genres) *responses.Genres {
	response := make([]responses.Genre, len(genres))
	for i, g := range genres {
		response[i] = responses.Genre{
			Name:       g.Name,
			SongCount:  int32(g.SongCount),
			AlbumCount: int32(g.AlbumCount),
		}
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
func childFromMediaFile(ctx context.Context, mf model.MediaFile) responses.Child {
	child := responses.Child{}
	child.Id = mf.ID
	child.Title = mf.Title
	child.IsDir = false
	child.Parent = mf.AlbumID
	child.Album = mf.Album
	child.Year = int32(mf.Year)
	child.Artist = mf.Artist
	child.Genre = mf.Genre
	child.Genres = itemGenresFromGenres(mf.Genres)
	child.Track = int32(mf.TrackNumber)
	child.Duration = int32(mf.Duration)
	child.Size = mf.Size
	child.Suffix = mf.Suffix
	child.BitRate = int32(mf.BitRate)
	child.CoverArt = mf.CoverArtID().String()
	child.ContentType = mf.ContentType()
	player, ok := request.PlayerFrom(ctx)
	if ok && player.ReportRealPath {
		child.Path = mf.Path
	} else {
		child.Path = fakePath(mf)
	}
	child.DiscNumber = int32(mf.DiscNumber)
	child.Created = &mf.CreatedAt
	child.AlbumId = mf.AlbumID
	child.ArtistId = mf.ArtistID
	child.Type = "music"
	child.PlayCount = mf.PlayCount
	if mf.PlayCount > 0 {
		child.Played = &mf.PlayDate
	}
	if mf.Starred {
		child.Starred = &mf.StarredAt
	}
	child.UserRating = int32(mf.Rating)

	format, _ := getTranscoding(ctx)
	if mf.Suffix != "" && format != "" && mf.Suffix != format {
		child.TranscodedSuffix = format
		child.TranscodedContentType = mime.TypeByExtension("." + format)
	}
	child.BookmarkPosition = mf.BookmarkPosition
	child.Comment = mf.Comment
	child.Bpm = int32(mf.Bpm)
	return child
}

func fakePath(mf model.MediaFile) string {
	filename := mapSlashToDash(mf.Title)
	if mf.TrackNumber != 0 {
		filename = fmt.Sprintf("%02d - %s", mf.TrackNumber, filename)
	}
	return fmt.Sprintf("%s/%s/%s.%s", mapSlashToDash(mf.AlbumArtist), mapSlashToDash(mf.Album), filename, mf.Suffix)
}

func mapSlashToDash(target string) string {
	return strings.ReplaceAll(target, "/", "_")
}

func childrenFromMediaFiles(ctx context.Context, mfs model.MediaFiles) []responses.Child {
	children := make([]responses.Child, len(mfs))
	for i, mf := range mfs {
		children[i] = childFromMediaFile(ctx, mf)
	}
	return children
}

func childFromAlbum(_ context.Context, al model.Album) responses.Child {
	child := responses.Child{}
	child.Id = al.ID
	child.IsDir = true
	child.Title = al.Name
	child.Name = al.Name
	child.Album = al.Name
	child.Artist = al.AlbumArtist
	child.Year = int32(al.MaxYear)
	child.Genre = al.Genre
	child.Genres = itemGenresFromGenres(al.Genres)
	child.CoverArt = al.CoverArtID().String()
	child.Created = &al.CreatedAt
	child.Parent = al.AlbumArtistID
	child.ArtistId = al.AlbumArtistID
	child.Duration = int32(al.Duration)
	child.SongCount = int32(al.SongCount)
	if al.Starred {
		child.Starred = &al.StarredAt
	}
	child.PlayCount = al.PlayCount
	if al.PlayCount > 0 {
		child.Played = &al.PlayDate
	}
	child.UserRating = int32(al.Rating)
	return child
}

func childrenFromAlbums(ctx context.Context, als model.Albums) []responses.Child {
	children := make([]responses.Child, len(als))
	for i, al := range als {
		children[i] = childFromAlbum(ctx, al)
	}
	return children
}

func itemGenresFromGenres(genres model.Genres) []responses.ItemGenre {
	itemGenres := make([]responses.ItemGenre, len(genres))
	for i, g := range genres {
		itemGenres[i] = responses.ItemGenre{Name: g.Name}
	}
	return itemGenres
}
