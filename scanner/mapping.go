package scanner

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner/metadata"
	"github.com/navidrome/navidrome/utils/str"
)

type MediaFileMapper struct {
	rootFolder string
	genres     model.GenreRepository
}

func NewMediaFileMapper(rootFolder string, genres model.GenreRepository) *MediaFileMapper {
	return &MediaFileMapper{
		rootFolder: rootFolder,
		genres:     genres,
	}
}

// TODO Move most of these mapping functions to setters in the model.MediaFile
func (s MediaFileMapper) ToMediaFile(md metadata.Tags) model.MediaFile {
	mf := &model.MediaFile{}
	mf.ID = s.trackID(md)
	mf.Year, mf.Date, mf.OriginalYear, mf.OriginalDate, mf.ReleaseYear, mf.ReleaseDate = s.mapDates(md)
	mf.Title = s.mapTrackTitle(md)
	mf.Album = md.Album()
	mf.AlbumID = s.albumID(md, mf.ReleaseDate)
	mf.Album = s.mapAlbumName(md)
	mf.ArtistID = s.artistID(md)
	mf.Artist = s.mapArtistName(md)
	mf.AlbumArtistID = s.albumArtistID(md)
	mf.AlbumArtist = s.mapAlbumArtistName(md)
	mf.Genre, mf.Genres = s.mapGenres(md.Genres())
	mf.Compilation = md.Compilation()
	mf.TrackNumber, _ = md.TrackNumber()
	mf.DiscNumber, _ = md.DiscNumber()
	mf.DiscSubtitle = md.DiscSubtitle()
	mf.Duration = md.Duration()
	mf.BitRate = md.BitRate()
	mf.SampleRate = md.SampleRate()
	mf.Channels = md.Channels()
	mf.Path = md.FilePath()
	mf.Suffix = md.Suffix()
	mf.Size = md.Size()
	mf.HasCoverArt = md.HasPicture()
	mf.SortTitle = md.SortTitle()
	mf.SortAlbumName = md.SortAlbum()
	mf.SortArtistName = md.SortArtist()
	mf.SortAlbumArtistName = md.SortAlbumArtist()
	mf.OrderTitle = str.SanitizeFieldForSorting(mf.Title)
	mf.OrderAlbumName = str.SanitizeFieldForSortingNoArticle(mf.Album)
	mf.OrderArtistName = str.SanitizeFieldForSortingNoArticle(mf.Artist)
	mf.OrderAlbumArtistName = str.SanitizeFieldForSortingNoArticle(mf.AlbumArtist)
	mf.CatalogNum = md.CatalogNum()
	mf.MbzRecordingID = md.MbzRecordingID()
	mf.MbzReleaseTrackID = md.MbzReleaseTrackID()
	mf.MbzAlbumID = md.MbzAlbumID()
	mf.MbzArtistID = md.MbzArtistID()
	mf.MbzAlbumArtistID = md.MbzAlbumArtistID()
	mf.MbzAlbumType = md.MbzAlbumType()
	mf.MbzAlbumComment = md.MbzAlbumComment()
	mf.RgAlbumGain = md.RGAlbumGain()
	mf.RgAlbumPeak = md.RGAlbumPeak()
	mf.RgTrackGain = md.RGTrackGain()
	mf.RgTrackPeak = md.RGTrackPeak()
	mf.Comment = str.SanitizeText(md.Comment())
	mf.Lyrics = md.Lyrics()
	mf.Bpm = md.Bpm()
	mf.CreatedAt = md.BirthTime()
	mf.UpdatedAt = md.ModificationTime()

	return *mf
}

func (s MediaFileMapper) mapTrackTitle(md metadata.Tags) string {
	if md.Title() == "" {
		s := strings.TrimPrefix(md.FilePath(), s.rootFolder+string(os.PathSeparator))
		e := filepath.Ext(s)
		return strings.TrimSuffix(s, e)
	}
	return md.Title()
}

func (s MediaFileMapper) mapAlbumArtistName(md metadata.Tags) string {
	switch {
	case md.AlbumArtist() != "":
		return md.AlbumArtist()
	case md.Compilation():
		return consts.VariousArtists
	case md.Artist() != "":
		return md.Artist()
	default:
		return consts.UnknownArtist
	}
}

func (s MediaFileMapper) mapArtistName(md metadata.Tags) string {
	if md.Artist() != "" {
		return md.Artist()
	}
	return consts.UnknownArtist
}

func (s MediaFileMapper) mapAlbumName(md metadata.Tags) string {
	name := md.Album()
	if name == "" {
		return consts.UnknownAlbum
	}
	return name
}

func (s MediaFileMapper) trackID(md metadata.Tags) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(md.FilePath())))
}

func (s MediaFileMapper) albumID(md metadata.Tags, releaseDate string) string {
	albumPath := strings.ToLower(fmt.Sprintf("%s\\%s", s.mapAlbumArtistName(md), s.mapAlbumName(md)))
	if !conf.Server.Scanner.GroupAlbumReleases {
		if len(releaseDate) != 0 {
			albumPath = fmt.Sprintf("%s\\%s", albumPath, releaseDate)
		}
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(albumPath)))
}

func (s MediaFileMapper) artistID(md metadata.Tags) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(s.mapArtistName(md)))))
}

func (s MediaFileMapper) albumArtistID(md metadata.Tags) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(s.mapAlbumArtistName(md)))))
}

func (s MediaFileMapper) mapGenres(genres []string) (string, model.Genres) {
	var result model.Genres
	unique := map[string]struct{}{}
	var all []string
	for i := range genres {
		gs := strings.FieldsFunc(genres[i], func(r rune) bool {
			return strings.ContainsRune(conf.Server.Scanner.GenreSeparators, r)
		})
		for j := range gs {
			g := strings.TrimSpace(gs[j])
			key := strings.ToLower(g)
			if _, ok := unique[key]; ok {
				continue
			}
			all = append(all, g)
			unique[key] = struct{}{}
		}
	}
	for _, g := range all {
		genre := model.Genre{Name: g}
		_ = s.genres.Put(&genre)
		result = append(result, genre)
	}
	if len(result) == 0 {
		return "", nil
	}
	return result[0].Name, result
}

func (s MediaFileMapper) mapDates(md metadata.Tags) (year int, date string,
	originalYear int, originalDate string,
	releaseYear int, releaseDate string) {
	// Start with defaults
	year, date = md.Date()
	originalYear, originalDate = md.OriginalDate()
	releaseYear, releaseDate = md.ReleaseDate()

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
