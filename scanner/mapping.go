package scanner

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deluan/sanitize"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner/metadata"
	"github.com/navidrome/navidrome/utils"
	"golang.org/x/exp/slices"
)

type mediaFileMapper struct {
	rootFolder string
	genres     model.GenreRepository
}

func newMediaFileMapper(rootFolder string, genres model.GenreRepository) *mediaFileMapper {
	return &mediaFileMapper{
		rootFolder: rootFolder,
		genres:     genres,
	}
}

// TODO Move most of these mapping functions to setters in the model.MediaFile
func (s mediaFileMapper) toMediaFile(md metadata.Tags) model.MediaFile {
	mf := &model.MediaFile{}
	// file properties
	mf.Duration = md.Duration()
	mf.BitRate = md.BitRate()
	mf.Channels = md.Channels()
	mf.Path = md.FilePath()
	mf.Suffix = md.Suffix()
	mf.Size = md.Size()
	mf.HasCoverArt = md.HasPicture()
	mf.CreatedAt = md.BirthTime()
	mf.UpdatedAt = md.ModificationTime()
	// classification
	mf.Classical = s.detectClassical(md)
	mf.Compilation = md.Compilation()
	// track, disc, album
	mf.TrackNumber, _ = md.TrackNumber()
	mf.Title, mf.WorkTitle = s.mapTrackTitle(md, mf.Classical)
	mf.DiscNumber, _ = md.DiscNumber()
	mf.DiscSubtitle = md.DiscSubtitle()
	mf.Album = s.mapAlbumName(md)
	// dates
	mf.Year, mf.Date, mf.OriginalYear, mf.OriginalDate, mf.ReleaseYear, mf.ReleaseDate = s.mapDates(md)
	// artists
	mf.Artist, mf.AlbumArtist, mf.ArtistID, mf.AlbumArtistID, mf.AllArtistIDs = s.mapArtists(md, mf.Classical)
	// sort+order names
	mf.SortTitle = md.SortTitle()
	mf.SortAlbumName = md.SortAlbum()
	mf.SortArtistName = md.SortArtist()
	mf.SortAlbumArtistName = md.SortAlbumArtist()
	mf.OrderTitle = strings.TrimSpace(sanitize.Accents(mf.Title))
	mf.OrderAlbumName = sanitizeFieldForSorting(mf.Album)
	mf.OrderArtistName = sanitizeFieldForSorting(mf.Artist)
	mf.OrderAlbumArtistName = sanitizeFieldForSorting(mf.AlbumArtist)
	// additional metadata
	mf.Genre, mf.Genres = s.mapGenres(md.Genres())
	mf.CatalogNum = md.CatalogNum()
	mf.Lyrics = utils.SanitizeText(md.Lyrics())
	mf.Bpm = md.Bpm()
	mf.Comment = utils.SanitizeText(md.Comment())
	// MusicBrainz tags
	mf.MbzRecordingID = md.MbzRecordingID()
	mf.MbzReleaseTrackID = md.MbzReleaseTrackID()
	mf.MbzAlbumID = md.MbzAlbumID()
	mf.MbzArtistID = md.MbzArtistID()
	mf.MbzAlbumArtistID = md.MbzAlbumArtistID()
	mf.MbzAlbumType = md.MbzAlbumType()
	mf.MbzAlbumComment = md.MbzAlbumComment()
	// ReplayGain tags
	mf.RGAlbumGain = md.RGAlbumGain()
	mf.RGAlbumPeak = md.RGAlbumPeak()
	mf.RGTrackGain = md.RGTrackGain()
	mf.RGTrackPeak = md.RGTrackPeak()
	// generate track & album IDs
	mf.ID = s.trackID(md)
	if conf.Server.Scanner.GroupAlbumReleases {
		mf.AlbumID = s.albumID(mf.Album, mf.AlbumArtistID)
	} else {
		mf.AlbumID = s.albumID(mf.Album, mf.AlbumArtistID, mf.ReleaseDate)
	}

	return *mf
}

func sanitizeFieldForSorting(originalValue string) string {
	// note: to be fixed after multi-artists database refactoring
	v := strings.Split(originalValue, " · ")[0]
	v = sanitize.Accents(v)
	return utils.NoArticle(v)
}

func (s mediaFileMapper) mapTrackTitle(md metadata.Tags, classical bool) (string, string) {
	title := md.Title()
	//subTitle := md.SubTitle()
	work := md.Work()
	movementName := md.MovementName()
	composers := md.Composers()
	arrangers := md.Arrangers()
	if classical {
		if movementName != "" {
			movementIndex, _ := md.MovementNumber()
			if movementIndex > 0 {
				title = strings.Join([]string{utils.IntToRoman(movementIndex), movementName}, ". ")
			}
		}
		if len(composers) > 0 || len(arrangers) > 0 {
			people := append(composers, arrangers...)
			peopleText := strings.Join(people, " · ")
			work = strings.Join([]string{work, peopleText}, " ♫ ")
		}
	}

	if title == "" {
		s := strings.TrimPrefix(md.FilePath(), s.rootFolder+string(os.PathSeparator))
		e := filepath.Ext(s)
		return strings.TrimSuffix(s, e), work
	}

	return title, work
}

func (s mediaFileMapper) mapArtists(md metadata.Tags, classical bool) (string, string, string, string, string) {
	// note: to be fixed after multi-artists database refactoring
	artists := utils.SanitizeProblematicChars(md.Artist())
	albumArtists := utils.SanitizeProblematicChars(md.AlbumArtist())
	var artistsWithoutAdditionals []string
	if conf.Server.Scanner.RemixerToArtist {
		artistsWithoutAdditionals = artists
		remixers := utils.SanitizeProblematicChars(md.Remixer())
		artists = append(artists, remixers...)
	}

	if classical {
		artistsWithoutAdditionals = artists
		composers := utils.SanitizeProblematicChars(md.Composers())
		artists = append(artists, composers...)
		conductors := utils.SanitizeProblematicChars(md.Conductors())
		artists = append(artists, conductors...)
		performers := utils.SanitizeProblematicChars(md.Performers())
		slices.Sort(performers)
		artists = append(artists, performers...)
	}

	if !conf.Server.Scanner.MultipleArtists {
		artists = artists[:1]
		albumArtists = albumArtists[:1]
	}

	var artistName, artistID string
	switch {
	case len(artists) > 1:
		artists = utils.RemoveDuplicateStr(artists)
		artistName = strings.Join(artists, " · ")
		artistID = fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(artists[0])))) // ID only on first artist
	case len(artists) == 1:
		artistName = artists[0]
		artistID = fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(artistName))))
	default:
		artistName = consts.UnknownArtist
		artistID = consts.UnknownArtistID
	}

	var albumArtistName, albumArtistID string
	switch {
	case md.Compilation() && len(albumArtists) == 0:
		albumArtistName = consts.VariousArtists
		albumArtistID = consts.VariousArtistsID
	case len(albumArtists) > 1:
		albumArtists = utils.RemoveDuplicateStr(albumArtists)
		albumArtistName = strings.Join(albumArtists, " · ")
		albumArtistID = fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(albumArtists[0])))) // ID only on first artist
	case len(albumArtists) == 1:
		albumArtistName = albumArtists[0]
		albumArtistID = fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(albumArtistName))))
	default:
		if conf.Server.Scanner.RemixerToArtist || classical {
			//if there's no album artist, use track artist without remixer!
			artists = artistsWithoutAdditionals
		}
		artists = utils.RemoveDuplicateStr(artists)
		albumArtistName = strings.Join(artists, " · ")
		albumArtistID = artistID
	}

	allArtists := utils.RemoveDuplicateStr(append(artists, albumArtists...))
	var allArtistIDs []string
	for i := range allArtists {
		allArtistIDs = append(allArtistIDs, fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(allArtists[i])))))
	}

	return artistName, albumArtistName, artistID, albumArtistID, strings.Join(allArtistIDs, " ")
}

func (s mediaFileMapper) mapAlbumName(md metadata.Tags) string {
	name := md.Album()
	if name == "" {
		return consts.UnknownAlbum
	}
	return name
}

func (s mediaFileMapper) trackID(md metadata.Tags) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(md.FilePath())))
}

func (s mediaFileMapper) albumID(values ...string) string {
	var albumPath = strings.Builder{}
	for _, value := range values {
		if value != "" {
			fmt.Fprintf(&albumPath, "\\%s", value)
		}
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(albumPath.String())))
}

func (s mediaFileMapper) mapGenres(genres []string) (string, model.Genres) {
	genres = utils.SanitizeProblematicChars(genres)
	var result model.Genres
	for _, g := range genres {
		genre := model.Genre{Name: g}
		_ = s.genres.Put(&genre)
		result = append(result, genre)
	}
	if len(result) == 0 {
		return "", nil
	}
	return result[0].Name, result
}

func (s mediaFileMapper) mapDates(md metadata.Tags) (int, string, int, string, int, string) {
	year, date := md.Date()
	originalYear, originalDate := md.OriginalDate()
	releaseYear, releaseDate := md.ReleaseDate()

	// MusicBrainz Picard writes the Release Date of an album to the Date tag, and leaves the Release Date tag empty
	taggedLikePicard := (originalYear != 0) &&
		(releaseYear == 0) &&
		(year >= originalYear)
	if taggedLikePicard {
		return originalYear, originalDate, originalYear, originalDate, year, date
	}
	return year, date, originalYear, originalDate, releaseYear, releaseDate
}

func (s mediaFileMapper) detectClassical(md metadata.Tags) bool {
	// the tag "is_classical" or "showmovement" forces a track to be Classical
	// autodetect criteria:
	// - if Movement + Composer are both populated -> true
	// - if one of the Genres is "Classical" -> true
	classical := md.Classical()
	if !classical && conf.Server.Scanner.AutoDetectClassical {
		if md.MovementName() != "" && len(md.Composers()) > 0 {
			return true
		}
		if slices.Contains(md.Genres(), "Classical") {
			return true
		}
	}
	return classical
}
