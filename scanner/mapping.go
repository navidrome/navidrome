package scanner

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deluan/sanitize"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner/metadata"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/slice"
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
	mf.ID = s.trackID(md)
	mf.Year, mf.Date, mf.OriginalYear, mf.OriginalDate, mf.ReleaseYear, mf.ReleaseDate = s.mapDates(md)
	mf.Title = s.mapTrackTitle(md)
	mf.Album = s.mapAlbumName(md)
	mf.Artist, mf.AlbumArtist, mf.ArtistID, mf.AlbumArtistID, mf.AllArtistIDs = s.mapArtists(md)
	mf.AlbumID = s.albumID(mf.Album, mf.AlbumArtistID, mf.ReleaseDate)
	mf.Genre, mf.Genres = s.mapGenres(md.Genres())
	mf.Compilation = md.Compilation()
	mf.TrackNumber, _ = md.TrackNumber()
	mf.DiscNumber, _ = md.DiscNumber()
	mf.DiscSubtitle = md.DiscSubtitle()
	mf.Duration = md.Duration()
	mf.BitRate = md.BitRate()
	mf.Channels = md.Channels()
	mf.Path = md.FilePath()
	mf.Suffix = md.Suffix()
	mf.Size = md.Size()
	mf.HasCoverArt = md.HasPicture()
	mf.SortTitle = md.SortTitle()
	mf.SortAlbumName = md.SortAlbum()
	mf.SortArtistName = md.SortArtist()
	mf.SortAlbumArtistName = md.SortAlbumArtist()
	mf.OrderTitle = strings.TrimSpace(sanitize.Accents(mf.Title))
	mf.OrderAlbumName = sanitizeFieldForSorting(mf.Album)
	mf.OrderArtistName = sanitizeFieldForSorting(mf.Artist)
	mf.OrderAlbumArtistName = sanitizeFieldForSorting(mf.AlbumArtist)
	mf.CatalogNum = md.CatalogNum()
	mf.MbzRecordingID = md.MbzRecordingID()
	mf.MbzReleaseTrackID = md.MbzReleaseTrackID()
	mf.MbzAlbumID = md.MbzAlbumID()
	mf.MbzArtistID = md.MbzArtistID()
	mf.MbzAlbumArtistID = md.MbzAlbumArtistID()
	mf.MbzAlbumType = md.MbzAlbumType()
	mf.MbzAlbumComment = md.MbzAlbumComment()
	mf.RGAlbumGain = md.RGAlbumGain()
	mf.RGAlbumPeak = md.RGAlbumPeak()
	mf.RGTrackGain = md.RGTrackGain()
	mf.RGTrackPeak = md.RGTrackPeak()
	mf.Comment = utils.SanitizeText(md.Comment())
	mf.Lyrics = utils.SanitizeText(md.Lyrics())
	mf.Bpm = md.Bpm()
	mf.CreatedAt = time.Now()
	mf.UpdatedAt = md.ModificationTime()

	return *mf
}

func sanitizeFieldForSorting(originalValue string) string {
	v := strings.Split(originalValue, "·")[0]
	v = strings.TrimSpace(sanitize.Accents(v))
	return utils.NoArticle(v)
}

func (s mediaFileMapper) mapTrackTitle(md metadata.Tags) string {
	if md.Title() == "" {
		s := strings.TrimPrefix(md.FilePath(), s.rootFolder+string(os.PathSeparator))
		e := filepath.Ext(s)
		return strings.TrimSuffix(s, e)
	}
	return md.Title()
}

func (s mediaFileMapper) mapArtists(md metadata.Tags) (string, string, string, string, string) {
	artists := utils.SanitizeChars(md.Artist())
	artistsWithoutRemixers := slice.RemoveDuplicateStr(artists)
	albumArtists := utils.SanitizeChars(md.AlbumArtist())

	if conf.Server.Scanner.RemixerToArtist {
		remixers := utils.SanitizeChars(md.Remixer())
		artists = append(artists, remixers...)
	}

	if !conf.Server.Scanner.MultipleArtists {
		artists = artists[:1]
		albumArtists = albumArtists[:1]
	}

	var artistName, artistID string
	switch {
	case len(artists) > 1:
		artists = slice.RemoveDuplicateStr(artists)
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
	case len(albumArtists) > 0:
		albumArtists = slice.RemoveDuplicateStr(albumArtists)
		albumArtistName = strings.Join(albumArtists, " · ")
		albumArtistID = fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(albumArtists[0])))) // ID only on first artist
	case md.Compilation():
		albumArtistName = consts.VariousArtists
		albumArtistID = consts.VariousArtistsID
	default:
		albumArtistName = strings.Join(artistsWithoutRemixers, " · ")
		albumArtistID = artistID
	}

	allArtists := slice.RemoveDuplicateStr(append(artists, albumArtists...))
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

func (s mediaFileMapper) albumID(albumName string, albumArtistID string, releaseDate string) string {
	albumPath := strings.ToLower(fmt.Sprintf("%s\\%s", albumArtistID, albumName))
	if !conf.Server.Scanner.GroupAlbumReleases && len(releaseDate) != 0 {
		albumPath = fmt.Sprintf("%s\\%s", albumPath, releaseDate)
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(albumPath)))
}

func (s mediaFileMapper) mapGenres(genres []string) (string, model.Genres) {
	genres = utils.SanitizeChars(genres)
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

	// when there's no Date, first fall back to Original Date, then to Release Date.
	if year == 0 {
		if originalYear > 0 {
			year, date = originalYear, originalDate
		} else {
			year, date = releaseYear, releaseDate
		}
	}
	return year, date, originalYear, originalDate, releaseYear, releaseDate
}
