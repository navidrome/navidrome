package metadata

import (
	"cmp"
	"encoding/json"
	"maps"
	"math"
	"strconv"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

func (md Metadata) ToMediaFile(libID int, folderID string) model.MediaFile {
	mf := model.MediaFile{
		LibraryID: libID,
		FolderID:  folderID,
		Tags:      maps.Clone(md.tags),
	}

	// Title and Album
	mf.Title = md.mapTrackTitle()
	mf.Album = md.mapAlbumName()
	mf.SortTitle = md.String(model.TagTitleSort)
	mf.SortAlbumName = md.String(model.TagAlbumSort)
	mf.OrderTitle = str.SanitizeFieldForSorting(mf.Title)
	mf.OrderAlbumName = str.SanitizeFieldForSortingNoArticle(mf.Album)
	mf.Compilation = md.Bool(model.TagCompilation)

	// Disc and Track info
	mf.TrackNumber, _ = md.NumAndTotal(model.TagTrackNumber)
	mf.DiscNumber, _ = md.NumAndTotal(model.TagDiscNumber)
	mf.DiscSubtitle = md.String(model.TagDiscSubtitle)
	mf.CatalogNum = md.String(model.TagCatalogNumber)
	mf.Comment = md.String(model.TagComment)
	mf.BPM = int(math.Round(md.Float(model.TagBPM)))
	mf.Lyrics = md.mapLyrics()
	mf.ExplicitStatus = md.mapExplicitStatusTag()

	// Dates
	date, origDate, relDate := md.mapDates()
	mf.OriginalYear, mf.OriginalDate = origDate.Year(), string(origDate)
	mf.ReleaseYear, mf.ReleaseDate = relDate.Year(), string(relDate)
	mf.Year, mf.Date = date.Year(), string(date)

	// MBIDs
	mf.MbzRecordingID = md.String(model.TagMusicBrainzRecordingID)
	mf.MbzReleaseTrackID = md.String(model.TagMusicBrainzTrackID)
	mf.MbzAlbumID = md.String(model.TagMusicBrainzAlbumID)
	mf.MbzReleaseGroupID = md.String(model.TagMusicBrainzReleaseGroupID)
	mf.MbzAlbumType = md.String(model.TagReleaseType)

	// ReplayGain
	mf.RGAlbumPeak = md.NullableFloat(model.TagReplayGainAlbumPeak)
	mf.RGAlbumGain = md.mapGain(model.TagReplayGainAlbumGain, model.TagR128AlbumGain)
	mf.RGTrackPeak = md.NullableFloat(model.TagReplayGainTrackPeak)
	mf.RGTrackGain = md.mapGain(model.TagReplayGainTrackGain, model.TagR128TrackGain)

	// General properties
	mf.HasCoverArt = md.HasPicture()
	mf.Duration = md.Length()
	mf.BitRate = md.AudioProperties().BitRate
	mf.SampleRate = md.AudioProperties().SampleRate
	mf.BitDepth = md.AudioProperties().BitDepth
	mf.Channels = md.AudioProperties().Channels
	mf.Path = md.FilePath()
	mf.Suffix = md.Suffix()
	mf.Size = md.Size()
	mf.BirthTime = md.BirthTime()
	mf.UpdatedAt = md.ModTime()

	mf.Participants = md.mapParticipants()
	mf.Artist = md.mapDisplayArtist()
	mf.AlbumArtist = md.mapDisplayAlbumArtist(mf)

	// Persistent IDs
	mf.PID = md.trackPID(mf)
	mf.AlbumID = md.albumID(mf, conf.Server.PID.Album)

	// BFR These IDs will go away once the UI handle multiple participants.
	// BFR For Legacy Subsonic compatibility, we will set them in the API handlers
	mf.ArtistID = mf.Participants.First(model.RoleArtist).ID
	mf.AlbumArtistID = mf.Participants.First(model.RoleAlbumArtist).ID

	// BFR What to do with sort/order artist names?
	mf.OrderArtistName = mf.Participants.First(model.RoleArtist).OrderArtistName
	mf.OrderAlbumArtistName = mf.Participants.First(model.RoleAlbumArtist).OrderArtistName
	mf.SortArtistName = mf.Participants.First(model.RoleArtist).SortArtistName
	mf.SortAlbumArtistName = mf.Participants.First(model.RoleAlbumArtist).SortArtistName

	// Don't store tags that are first-class fields (and are not album-level tags) in the
	// MediaFile struct. This is to avoid redundancy in the DB
	//
	// Remove all tags from the main section that are not flagged as album tags
	for tag, conf := range model.TagMainMappings() {
		if !conf.Album {
			delete(mf.Tags, tag)
		}
	}

	return mf
}

func (md Metadata) AlbumID(mf model.MediaFile, pidConf string) string {
	return md.albumID(mf, pidConf)
}

func (md Metadata) mapGain(rg, r128 model.TagName) *float64 {
	v := md.Gain(rg)
	if v != nil {
		return v
	}
	r128value := md.String(r128)
	if r128value != "" {
		var v, err = strconv.Atoi(r128value)
		if err != nil {
			return nil
		}
		// Convert Q7.8 to float
		value := float64(v) / 256.0
		// Adding 5 dB to normalize with ReplayGain level
		value += 5
		return &value
	}
	return nil
}

func (md Metadata) mapLyrics() string {
	rawLyrics := md.Pairs(model.TagLyrics)

	lyricList := make(model.LyricList, 0, len(rawLyrics))

	for _, raw := range rawLyrics {
		lang := raw.Key()
		text := raw.Value()

		lyrics, err := model.ToLyrics(lang, text)
		if err != nil {
			log.Warn("Unexpected failure occurred when parsing lyrics", "file", md.filePath, err)
			continue
		}
		if !lyrics.IsEmpty() {
			lyricList = append(lyricList, *lyrics)
		}
	}

	res, err := json.Marshal(lyricList)
	if err != nil {
		log.Warn("Unexpected error occurred when serializing lyrics", "file", md.filePath, err)
		return ""
	}
	return string(res)
}

func (md Metadata) mapExplicitStatusTag() string {
	switch md.first(model.TagExplicitStatus) {
	case "1", "4":
		return "e"
	case "2":
		return "c"
	default:
		return ""
	}
}

func (md Metadata) mapDates() (date Date, originalDate Date, releaseDate Date) {
	// Start with defaults
	date = md.Date(model.TagRecordingDate)
	originalDate = md.Date(model.TagOriginalDate)
	releaseDate = md.Date(model.TagReleaseDate)

	// For some historic reason, taggers have been writing the Release Date of an album to the Date tag,
	// and leave the Release Date tag empty.
	legacyMappings := (originalDate != "") &&
		(releaseDate == "") &&
		(date >= originalDate)
	if legacyMappings {
		return originalDate, originalDate, date
	}
	// when there's no Date, first fall back to Original Date, then to Release Date.
	date = cmp.Or(date, originalDate, releaseDate)
	return date, originalDate, releaseDate
}
