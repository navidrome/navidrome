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
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/publicurl"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
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
	messages []any
}

func newError(code int32, message ...any) error {
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
		ArtistImageUrl: publicurl.ImageURL(r, a.CoverArtID(), 600),
	}
	if conf.Server.Subsonic.EnableAverageRating {
		artist.AverageRating = a.AverageRating
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
		ArtistImageUrl: publicurl.ImageURL(r, a.CoverArtID(), 600),
		UserRating:     int32(a.Rating),
	}
	if conf.Server.Subsonic.EnableAverageRating {
		artist.AverageRating = a.AverageRating
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

func isClientInList(clientList, client string) bool {
	if clientList == "" || client == "" {
		return false
	}
	clients := strings.SplitSeq(clientList, ",")
	for c := range clients {
		if strings.TrimSpace(c) == client {
			return true
		}
	}
	return false
}

func childFromMediaFile(ctx context.Context, mf model.MediaFile) responses.Child {
	child := responses.Child{}
	child.Id = mf.ID
	child.Title = mf.FullTitle()
	child.IsDir = false

	player, ok := request.PlayerFrom(ctx)
	if ok && isClientInList(conf.Server.Subsonic.MinimalClients, player.Client) {
		return child
	}

	child.Parent = mf.AlbumID
	child.Album = mf.FullAlbumName()
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

	if ok && player.ReportRealPath {
		child.Path = mf.AbsolutePath()
	} else {
		child.Path = fakePath(mf)
	}
	child.DiscNumber = int32(mf.DiscNumber)
	child.Created = new(mf.BirthTime)
	child.AlbumId = mf.AlbumID
	child.ArtistId = mf.ArtistID
	child.Type = "music"
	child.PlayCount = mf.PlayCount
	if mf.Starred {
		child.Starred = mf.StarredAt
	}
	child.UserRating = int32(mf.Rating)
	if conf.Server.Subsonic.EnableAverageRating {
		child.AverageRating = mf.AverageRating
	}

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
	player, ok := request.PlayerFrom(ctx)
	if ok && isClientInList(conf.Server.Subsonic.LegacyClients, player.Client) {
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
	child.Groupings = mf.Tags.Values(model.TagGrouping)
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

	builder.WriteString(fmt.Sprintf("%s/%s/", sanitizeSlashes(mf.AlbumArtist), sanitizeSlashes(mf.FullAlbumName())))
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

// albumCreatedAt returns a best-effort timestamp for the album's `created`
// field, which is required by the OpenSubsonic spec but may be zero on legacy
// DB rows. Falls back to UpdatedAt → ImportedAt; can still return zero if all
// three are unset.
func albumCreatedAt(al model.Album) time.Time {
	if !al.CreatedAt.IsZero() {
		return al.CreatedAt
	}
	if !al.UpdatedAt.IsZero() {
		return al.UpdatedAt
	}
	return al.ImportedAt
}

func childFromAlbum(ctx context.Context, al model.Album) responses.Child {
	child := responses.Child{}
	child.Id = al.ID
	child.IsDir = true
	fullName := al.FullName()
	child.Title = fullName
	child.Name = fullName
	child.Album = fullName
	child.Artist = al.AlbumArtist
	child.Year = int32(cmp.Or(al.MaxOriginalYear, al.MaxYear))
	child.Genre = al.Genre
	child.CoverArt = al.CoverArtID().String()
	child.Created = new(albumCreatedAt(al))
	child.Parent = al.AlbumArtistID
	child.ArtistId = al.AlbumArtistID
	child.Duration = int32(al.Duration)
	child.SongCount = int32(al.SongCount)
	if al.Starred {
		child.Starred = al.StarredAt
	}
	child.PlayCount = al.PlayCount
	child.UserRating = int32(al.Rating)
	if conf.Server.Subsonic.EnableAverageRating {
		child.AverageRating = al.AverageRating
	}
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
	child.Groupings = al.Tags.Values(model.TagGrouping)
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
	// Hoist UpdatedAt to a single stack-local so &updatedAt doesn't force the
	// whole model.Album parameter onto the heap.
	updatedAt := a.UpdatedAt
	for num, title := range a.Discs {
		artID := model.NewArtworkID(model.KindDiscArtwork,
			model.DiscArtworkID(a.ID, num), &updatedAt)
		discTitles = append(discTitles, responses.DiscTitle{
			Disc:     int32(num),
			Title:    title,
			CoverArt: artID.String(),
		})
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
	dir.Name = album.FullName()
	dir.Artist = album.AlbumArtist
	dir.ArtistId = album.AlbumArtistID
	dir.CoverArt = album.CoverArtID().String()
	dir.SongCount = int32(album.SongCount)
	dir.Duration = int32(album.Duration)
	dir.PlayCount = album.PlayCount
	dir.Year = int32(cmp.Or(album.MaxOriginalYear, album.MaxYear))
	dir.Genre = album.Genre
	dir.Created = albumCreatedAt(album)
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
	if conf.Server.Subsonic.EnableAverageRating {
		dir.AverageRating = album.AverageRating
	}
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

func buildStructuredLyric(mf *model.MediaFile, lyrics model.Lyrics, enhanced bool) responses.StructuredLyric {
	// V1 line-level shape: collapse one CueLine per logical moment (lowest
	// AgentID wins when multiple agents share an Index).
	lines := buildV1Lines(lyrics.CueLine)

	structured := responses.StructuredLyric{
		DisplayArtist: lyrics.DisplayArtist,
		DisplayTitle:  lyrics.DisplayTitle,
		Lang:          lyrics.Lang,
		Line:          lines,
		Offset:        lyrics.Offset,
		Synced:        lyrics.Synced,
	}

	if enhanced {
		structured.Kind = string(lyrics.Kind)
		if len(lyrics.Agents) > 0 {
			structured.Agents = make([]responses.Agent, len(lyrics.Agents))
			for i, a := range lyrics.Agents {
				structured.Agents[i] = responses.Agent{ID: a.ID, Name: a.Name, Role: a.Role}
			}
		}
		structured.CueLine = buildV2CueLines(lyrics.CueLine)
	}

	if structured.DisplayArtist == "" {
		structured.DisplayArtist = mf.Artist
	}
	if structured.DisplayTitle == "" {
		structured.DisplayTitle = mf.Title
	}

	return structured
}

// buildV1Lines collapses canonical CueLines into the v1 wire shape. When
// multiple cuelines share an Index (overlapping vocals), only the first one
// (lowest agentID) is included to keep v1-only clients deterministic.
func buildV1Lines(cueLines []model.CueLine) []responses.Line {
	if len(cueLines) == 0 {
		return nil
	}
	lines := make([]responses.Line, 0, len(cueLines))
	seenIndex := -1
	for _, cl := range cueLines {
		if cl.Index == seenIndex {
			continue
		}
		seenIndex = cl.Index
		lines = append(lines, responses.Line{
			Start: cl.Start,
			Value: cl.Value,
		})
	}
	return lines
}

// buildV2CueLines emits the v2 cueLine[] structure. Line-only cuelines (no
// per-word data) are skipped
func buildV2CueLines(cueLines []model.CueLine) []responses.CueLine {
	if len(cueLines) == 0 {
		return nil
	}
	out := make([]responses.CueLine, 0, len(cueLines))
	for _, cl := range cueLines {
		if len(cl.Cue) == 0 {
			continue
		}
		out = append(out, responses.CueLine{
			Index:   cl.Index,
			Start:   cl.Start,
			End:     cl.End,
			Value:   cl.Value,
			AgentID: cl.AgentID,
			Cue:     buildCue(cl),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// buildCue maps each model.Cue to one responses.Cue with inclusive UTF-8
// ByteStart/ByteEnd offsets into cl.Value. 
func buildCue(cl model.CueLine) []responses.Cue {
	if len(cl.Cue) == 0 {
		return nil
	}

	ends := make([]*int64, len(cl.Cue))
	for i, c := range cl.Cue {
		ends[i] = c.End
	}
	if ends[len(ends)-1] == nil && cl.End != nil {
		e := *cl.End
		ends[len(ends)-1] = &e
	}
	allHaveEnd := true
	anyEnd := false
	for _, e := range ends {
		if e != nil {
			anyEnd = true
		} else {
			allHaveEnd = false
		}
	}
	if anyEnd && !allHaveEnd {
		for i := range ends {
			ends[i] = nil
		}
	}

	cues := make([]responses.Cue, len(cl.Cue))
	var byteCursor int64
	for i, c := range cl.Cue {
		valueBytes := int64(len(c.Value))
		bs := byteCursor
		be := bs
		if valueBytes > 0 {
			be = bs + valueBytes - 1
			byteCursor = be + 1
		}
		bsCopy, beCopy := bs, be

		cues[i] = responses.Cue{
			Start:     c.Start,
			End:       ends[i],
			Value:     c.Value,
			ByteStart: &bsCopy,
			ByteEnd:   &beCopy,
		}
	}
	return cues
}

func buildLyricsList(mf *model.MediaFile, lyricsList model.LyricList, enhanced bool) *responses.LyricsList {
	lyricList := make(responses.StructuredLyrics, len(lyricsList))

	for i, lyrics := range lyricsList {
		lyricList[i] = buildStructuredLyric(mf, lyrics, enhanced)
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
