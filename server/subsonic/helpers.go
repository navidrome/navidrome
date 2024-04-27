package subsonic

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/public"
	"github.com/navidrome/navidrome/server/subsonic/responses"
)

func newResponse() *responses.Subsonic {
	return &responses.Subsonic{
		Status:        responses.StatusOK,
		Version:       Version,
		Type:          consts.AppName,
		ServerVersion: consts.Version,
		OpenSubsonic:  true,
	}
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

// errSubsonic and Unwrap are used to allow `errors.Is(err, errSubsonic)` to work
var errSubsonic = errors.New("subsonic API error")

func (e subError) Unwrap() error {
	return fmt.Errorf("%w: %d", errSubsonic, e.code)
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
		artist.Starred = a.StarredAt
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
		SortName:       a.SortArtistName,
	}
	if a.Starred {
		artist.Starred = a.StarredAt
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
	child.Genres = buildItemGenres(mf.Genres)
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
		child.Played = mf.PlayDate
	}
	if mf.Starred {
		child.Starred = mf.StarredAt
	}
	child.UserRating = int32(mf.Rating)

	format, _ := getTranscoding(ctx)
	if mf.Suffix != "" && format != "" && mf.Suffix != format {
		child.TranscodedSuffix = format
		child.TranscodedContentType = mime.TypeByExtension("." + format)
	}
	child.BookmarkPosition = mf.BookmarkPosition
	child.Comment = mf.Comment
	child.SortName = mf.SortTitle
	child.Bpm = int32(mf.Bpm)
	child.MediaType = responses.MediaTypeSong
	child.MusicBrainzId = mf.MbzRecordingID
	child.ReplayGain = responses.ReplayGain{
		TrackGain: mf.RgTrackGain,
		AlbumGain: mf.RgAlbumGain,
		TrackPeak: mf.RgTrackPeak,
		AlbumPeak: mf.RgAlbumPeak,
	}
	child.ChannelCount = int32(mf.Channels)
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
	child.Genres = buildItemGenres(al.Genres)
	child.CoverArt = al.CoverArtID().String()
	child.Created = &al.CreatedAt
	child.Parent = al.AlbumArtistID
	child.ArtistId = al.AlbumArtistID
	child.Duration = int32(al.Duration)
	child.SongCount = int32(al.SongCount)
	if al.Starred {
		child.Starred = al.StarredAt
	}
	child.PlayCount = al.PlayCount
	if al.PlayCount > 0 {
		child.Played = al.PlayDate
	}
	child.UserRating = int32(al.Rating)
	child.SortName = al.SortAlbumName
	child.MediaType = responses.MediaTypeAlbum
	child.MusicBrainzId = al.MbzAlbumID
	return child
}

func childrenFromAlbums(ctx context.Context, als model.Albums) []responses.Child {
	children := make([]responses.Child, len(als))
	for i, al := range als {
		children[i] = childFromAlbum(ctx, al)
	}
	return children
}

// toItemDate converts a string date in the formats 'YYYY-MM-DD', 'YYYY-MM' or 'YYYY' to an OS ItemDate
func toItemDate(date string) responses.ItemDate {
	itemDate := responses.ItemDate{}
	if date == "" {
		return itemDate
	}
	parts := strings.Split(date, "-")
	if len(parts) > 2 {
		itemDate.Day, _ = strconv.Atoi(parts[2])
	}
	if len(parts) > 1 {
		itemDate.Month, _ = strconv.Atoi(parts[1])
	}
	itemDate.Year, _ = strconv.Atoi(parts[0])

	return itemDate
}

func buildItemGenres(genres model.Genres) []responses.ItemGenre {
	itemGenres := make([]responses.ItemGenre, len(genres))
	for i, g := range genres {
		itemGenres[i] = responses.ItemGenre{Name: g.Name}
	}
	return itemGenres
}

func buildDiscSubtitles(_ context.Context, a model.Album) responses.DiscTitles {
	if len(a.Discs) == 0 {
		return nil
	}
	discTitles := responses.DiscTitles{}
	for num, title := range a.Discs {
		discTitles = append(discTitles, responses.DiscTitle{Disc: num, Title: title})
	}
	sort.Slice(discTitles, func(i, j int) bool {
		return discTitles[i].Disc < discTitles[j].Disc
	})
	return discTitles
}

func buildAlbumsID3(ctx context.Context, albums model.Albums) []responses.AlbumID3 {
	res := make([]responses.AlbumID3, len(albums))
	for i, album := range albums {
		res[i] = buildAlbumID3(ctx, album)
	}
	return res
}

func buildAlbumID3(ctx context.Context, album model.Album) responses.AlbumID3 {
	dir := responses.AlbumID3{}
	dir.Id = album.ID
	dir.Name = album.Name
	dir.Artist = album.AlbumArtist
	dir.ArtistId = album.AlbumArtistID
	dir.CoverArt = album.CoverArtID().String()
	dir.SongCount = int32(album.SongCount)
	dir.Duration = int32(album.Duration)
	dir.PlayCount = album.PlayCount
	if album.PlayCount > 0 {
		dir.Played = album.PlayDate
	}
	dir.Year = int32(album.MaxYear)
	dir.Genre = album.Genre
	dir.Genres = buildItemGenres(album.Genres)
	dir.DiscTitles = buildDiscSubtitles(ctx, album)
	dir.UserRating = int32(album.Rating)
	if !album.CreatedAt.IsZero() {
		dir.Created = &album.CreatedAt
	}
	if album.Starred {
		dir.Starred = album.StarredAt
	}
	dir.MusicBrainzId = album.MbzAlbumID
	dir.IsCompilation = album.Compilation
	dir.SortName = album.SortAlbumName
	dir.OriginalReleaseDate = toItemDate(album.OriginalDate)
	dir.ReleaseDate = toItemDate(album.ReleaseDate)
	return dir
}

func buildStructuredLyric(mf *model.MediaFile, lyrics model.Lyrics) responses.StructuredLyric {
	lines := make([]responses.Line, len(lyrics.Line))

	for i, line := range lyrics.Line {
		lines[i] = responses.Line{
			Start: line.Start,
			Value: line.Value,
		}
	}

	structured := responses.StructuredLyric{
		DisplayArtist: lyrics.DisplayArtist,
		DisplayTitle:  lyrics.DisplayTitle,
		Lang:          lyrics.Lang,
		Line:          lines,
		Offset:        lyrics.Offset,
		Synced:        lyrics.Synced,
	}

	if structured.DisplayArtist == "" {
		structured.DisplayArtist = mf.Artist
	}
	if structured.DisplayTitle == "" {
		structured.DisplayTitle = mf.Title
	}

	return structured
}

func buildLyricsList(mf *model.MediaFile, lyricsList model.LyricList) *responses.LyricsList {
	lyricList := make(responses.StructuredLyrics, len(lyricsList))

	for i, lyrics := range lyricsList {
		lyricList[i] = buildStructuredLyric(mf, lyrics)
	}

	res := &responses.LyricsList{
		StructuredLyrics: lyricList,
	}
	return res
}
