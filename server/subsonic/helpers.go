package subsonic

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/public"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/number"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
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
	code     int32
	messages []interface{}
}

func newError(code int32, message ...interface{}) error {
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

func sortName(sortName, orderName string) string {
	if conf.Server.PreferSortTags {
		return cmp.Or(
			sortName,
			orderName,
		)
	}
	return orderName
}

func getArtistAlbumCount(a *model.Artist) int32 {
	// If ArtistParticipations are set, then `getArtist` will return albums
	// where the artist is an album artist OR artist. Use the custom stat
	// main credit for this calculation.
	// Otherwise, return just the roles as album artist (precise)
	if conf.Server.Subsonic.ArtistParticipations {
		mainCreditStats := a.Stats[model.RoleMainCredit]
		return int32(mainCreditStats.AlbumCount)
	} else {
		albumStats := a.Stats[model.RoleAlbumArtist]
		return int32(albumStats.AlbumCount)
	}
}

func toArtist(r *http.Request, a model.Artist) responses.Artist {
	artist := responses.Artist{
		Id:             a.ID,
		Name:           a.Name,
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
		AlbumCount:     getArtistAlbumCount(&a),
		CoverArt:       a.CoverArtID().String(),
		ArtistImageUrl: public.ImageURL(r, a.CoverArtID(), 600),
		UserRating:     int32(a.Rating),
	}
	if a.Starred {
		artist.Starred = a.StarredAt
	}
	artist.OpenSubsonicArtistID3 = toOSArtistID3(r.Context(), a)
	return artist
}

func toOSArtistID3(ctx context.Context, a model.Artist) *responses.OpenSubsonicArtistID3 {
	player, _ := request.PlayerFrom(ctx)
	if strings.Contains(conf.Server.Subsonic.LegacyClients, player.Client) {
		return nil
	}
	artist := responses.OpenSubsonicArtistID3{
		MusicBrainzId: a.MbzArtistID,
		SortName:      sortName(a.SortArtistName, a.OrderArtistName),
	}
	artist.Roles = slice.Map(a.Roles(), func(r model.Role) string { return r.String() })
	return &artist
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

func toItemGenres(genres model.Genres) []responses.ItemGenre {
	itemGenres := make([]responses.ItemGenre, len(genres))
	for i, g := range genres {
		itemGenres[i] = responses.ItemGenre{Name: g.Name}
	}
	return itemGenres
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

func childFromMediaFile(ctx context.Context, mf model.MediaFile) responses.Child {
	child := responses.Child{}
	child.Id = mf.ID
	child.Title = mf.FullTitle()
	child.IsDir = false
	child.Parent = mf.AlbumID
	child.Album = mf.Album
	child.Year = int32(mf.Year)
	child.Artist = mf.Artist
	child.Genre = mf.Genre
	child.Track = int32(mf.TrackNumber)
	child.Duration = int32(mf.Duration)
	child.Size = mf.Size
	child.Suffix = mf.Suffix
	child.BitRate = int32(mf.BitRate)
	child.CoverArt = mf.CoverArtID().String()
	child.ContentType = mf.ContentType()
	player, ok := request.PlayerFrom(ctx)
	if ok && player.ReportRealPath {
		child.Path = mf.AbsolutePath()
	} else {
		child.Path = fakePath(mf)
	}
	child.DiscNumber = int32(mf.DiscNumber)
	child.Created = &mf.BirthTime
	child.AlbumId = mf.AlbumID
	child.ArtistId = mf.ArtistID
	child.Type = "music"
	child.PlayCount = mf.PlayCount
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
	child.OpenSubsonicChild = osChildFromMediaFile(ctx, mf)
	return child
}

func osChildFromMediaFile(ctx context.Context, mf model.MediaFile) *responses.OpenSubsonicChild {
	player, _ := request.PlayerFrom(ctx)
	if strings.Contains(conf.Server.Subsonic.LegacyClients, player.Client) {
		return nil
	}
	child := responses.OpenSubsonicChild{}
	if mf.PlayCount > 0 {
		child.Played = mf.PlayDate
	}
	child.Comment = mf.Comment
	child.SortName = sortName(mf.SortTitle, mf.OrderTitle)
	child.BPM = int32(mf.BPM)
	child.MediaType = responses.MediaTypeSong
	child.MusicBrainzId = mf.MbzRecordingID
	child.Isrc = mf.Tags.Values(model.TagISRC)
	child.ReplayGain = responses.ReplayGain{
		TrackGain: mf.RGTrackGain,
		AlbumGain: mf.RGAlbumGain,
		TrackPeak: mf.RGTrackPeak,
		AlbumPeak: mf.RGAlbumPeak,
	}
	child.ChannelCount = int32(mf.Channels)
	child.SamplingRate = int32(mf.SampleRate)
	child.BitDepth = int32(mf.BitDepth)
	child.Genres = toItemGenres(mf.Genres)
	child.Moods = mf.Tags.Values(model.TagMood)
	child.DisplayArtist = mf.Artist
	child.Artists = artistRefs(mf.Participants[model.RoleArtist])
	child.DisplayAlbumArtist = mf.AlbumArtist
	child.AlbumArtists = artistRefs(mf.Participants[model.RoleAlbumArtist])
	var contributors []responses.Contributor
	child.DisplayComposer = mf.Participants[model.RoleComposer].Join(consts.ArtistJoiner)
	for role, participants := range mf.Participants {
		if role == model.RoleArtist || role == model.RoleAlbumArtist {
			continue
		}
		for _, participant := range participants {
			contributors = append(contributors, responses.Contributor{
				Role:    role.String(),
				SubRole: participant.SubRole,
				Artist: responses.ArtistID3Ref{
					Id:   participant.ID,
					Name: participant.Name,
				},
			})
		}
	}
	child.Contributors = contributors
	child.ExplicitStatus = mapExplicitStatus(mf.ExplicitStatus)
	return &child
}

func artistRefs(participants model.ParticipantList) []responses.ArtistID3Ref {
	return slice.Map(participants, func(p model.Participant) responses.ArtistID3Ref {
		return responses.ArtistID3Ref{
			Id:   p.ID,
			Name: p.Name,
		}
	})
}

func fakePath(mf model.MediaFile) string {
	builder := strings.Builder{}

	builder.WriteString(fmt.Sprintf("%s/%s/", sanitizeSlashes(mf.AlbumArtist), sanitizeSlashes(mf.Album)))
	if mf.DiscNumber != 0 {
		builder.WriteString(fmt.Sprintf("%02d-", mf.DiscNumber))
	}
	if mf.TrackNumber != 0 {
		builder.WriteString(fmt.Sprintf("%02d - ", mf.TrackNumber))
	}
	builder.WriteString(fmt.Sprintf("%s.%s", sanitizeSlashes(mf.FullTitle()), mf.Suffix))
	return builder.String()
}

func sanitizeSlashes(target string) string {
	return strings.ReplaceAll(target, "/", "_")
}

func childFromAlbum(ctx context.Context, al model.Album) responses.Child {
	child := responses.Child{}
	child.Id = al.ID
	child.IsDir = true
	child.Title = al.Name
	child.Name = al.Name
	child.Album = al.Name
	child.Artist = al.AlbumArtist
	child.Year = int32(cmp.Or(al.MaxOriginalYear, al.MaxYear))
	child.Genre = al.Genre
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
	child.UserRating = int32(al.Rating)
	child.OpenSubsonicChild = osChildFromAlbum(ctx, al)
	return child
}

func osChildFromAlbum(ctx context.Context, al model.Album) *responses.OpenSubsonicChild {
	player, _ := request.PlayerFrom(ctx)
	if strings.Contains(conf.Server.Subsonic.LegacyClients, player.Client) {
		return nil
	}
	child := responses.OpenSubsonicChild{}
	if al.PlayCount > 0 {
		child.Played = al.PlayDate
	}
	child.MediaType = responses.MediaTypeAlbum
	child.MusicBrainzId = al.MbzAlbumID
	child.Genres = toItemGenres(al.Genres)
	child.Moods = al.Tags.Values(model.TagMood)
	child.DisplayArtist = al.AlbumArtist
	child.Artists = artistRefs(al.Participants[model.RoleAlbumArtist])
	child.DisplayAlbumArtist = al.AlbumArtist
	child.AlbumArtists = artistRefs(al.Participants[model.RoleAlbumArtist])
	child.ExplicitStatus = mapExplicitStatus(al.ExplicitStatus)
	child.SortName = sortName(al.SortAlbumName, al.OrderAlbumName)
	return &child
}

// toItemDate converts a string date in the formats 'YYYY-MM-DD', 'YYYY-MM' or 'YYYY' to an OS ItemDate
func toItemDate(date string) responses.ItemDate {
	itemDate := responses.ItemDate{}
	if date == "" {
		return itemDate
	}
	parts := strings.Split(date, "-")
	if len(parts) > 2 {
		itemDate.Day = number.ParseInt[int32](parts[2])
	}
	if len(parts) > 1 {
		itemDate.Month = number.ParseInt[int32](parts[1])
	}
	itemDate.Year = number.ParseInt[int32](parts[0])

	return itemDate
}

func buildDiscSubtitles(a model.Album) []responses.DiscTitle {
	if len(a.Discs) == 0 {
		return nil
	}
	var discTitles []responses.DiscTitle
	for num, title := range a.Discs {
		discTitles = append(discTitles, responses.DiscTitle{Disc: int32(num), Title: title})
	}
	if len(discTitles) == 1 && discTitles[0].Title == "" {
		return nil
	}
	sort.Slice(discTitles, func(i, j int) bool {
		return discTitles[i].Disc < discTitles[j].Disc
	})
	return discTitles
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
	dir.Year = int32(cmp.Or(album.MaxOriginalYear, album.MaxYear))
	dir.Genre = album.Genre
	if !album.CreatedAt.IsZero() {
		dir.Created = &album.CreatedAt
	}
	if album.Starred {
		dir.Starred = album.StarredAt
	}
	dir.OpenSubsonicAlbumID3 = buildOSAlbumID3(ctx, album)
	return dir
}

func buildOSAlbumID3(ctx context.Context, album model.Album) *responses.OpenSubsonicAlbumID3 {
	player, _ := request.PlayerFrom(ctx)
	if strings.Contains(conf.Server.Subsonic.LegacyClients, player.Client) {
		return nil
	}
	dir := responses.OpenSubsonicAlbumID3{}
	if album.PlayCount > 0 {
		dir.Played = album.PlayDate
	}
	dir.UserRating = int32(album.Rating)
	dir.RecordLabels = slice.Map(album.Tags.Values(model.TagRecordLabel), func(s string) responses.RecordLabel {
		return responses.RecordLabel{Name: s}
	})
	dir.MusicBrainzId = album.MbzAlbumID
	dir.Genres = toItemGenres(album.Genres)
	dir.Artists = artistRefs(album.Participants[model.RoleAlbumArtist])
	dir.DisplayArtist = album.AlbumArtist
	dir.ReleaseTypes = album.Tags.Values(model.TagReleaseType)
	dir.Moods = album.Tags.Values(model.TagMood)
	dir.SortName = sortName(album.SortAlbumName, album.OrderAlbumName)
	dir.OriginalReleaseDate = toItemDate(album.OriginalDate)
	dir.ReleaseDate = toItemDate(album.ReleaseDate)
	dir.IsCompilation = album.Compilation
	dir.DiscTitles = buildDiscSubtitles(album)
	dir.ExplicitStatus = mapExplicitStatus(album.ExplicitStatus)
	if len(album.Tags.Values(model.TagAlbumVersion)) > 0 {
		dir.Version = album.Tags.Values(model.TagAlbumVersion)[0]
	}

	return &dir
}

func mapExplicitStatus(explicitStatus string) string {
	switch explicitStatus {
	case "c":
		return "clean"
	case "e":
		return "explicit"
	}
	return ""
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

// getUserAccessibleLibraries returns the list of libraries the current user has access to.
func getUserAccessibleLibraries(ctx context.Context) []model.Library {
	user := getUser(ctx)
	return user.Libraries
}

// selectedMusicFolderIds retrieves the music folder IDs from the request parameters.
// If no IDs are provided, it returns all libraries the user has access to (based on the user found in the context).
// If the parameter is required and not present, it returns an error.
// If any of the provided library IDs are invalid (don't exist or user doesn't have access), returns ErrorDataNotFound.
func selectedMusicFolderIds(r *http.Request, required bool) ([]int, error) {
	p := req.Params(r)
	musicFolderIds, err := p.Ints("musicFolderId")

	// If the parameter is not present, it returns an error if it is required.
	if errors.Is(err, req.ErrMissingParam) && required {
		return nil, err
	}

	// Get user's accessible libraries for validation
	libraries := getUserAccessibleLibraries(r.Context())
	accessibleLibraryIds := slice.Map(libraries, func(lib model.Library) int { return lib.ID })

	if len(musicFolderIds) > 0 {
		// Validate all provided library IDs - if any are invalid, return an error
		for _, id := range musicFolderIds {
			if !slices.Contains(accessibleLibraryIds, id) {
				return nil, newError(responses.ErrorDataNotFound, "Library %d not found or not accessible", id)
			}
		}
		return musicFolderIds, nil
	}

	// If no musicFolderId is provided, return all libraries the user has access to.
	return accessibleLibraryIds, nil
}
