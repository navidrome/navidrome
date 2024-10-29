package metadata

import (
	"math"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

func (md Metadata) ToMediaFile() model.MediaFile {
	mf := model.MediaFile{
		Tags: md.tags,
	}
	mf.Title = md.mapTrackTitle()
	mf.Album = md.mapAlbumName()
	mf.SortTitle = md.String(model.TagTitleSort)
	mf.SortAlbumName = md.String(model.TagAlbumSort)
	mf.OrderTitle = str.SanitizeFieldForSorting(mf.Title)
	mf.OrderAlbumName = str.SanitizeFieldForSortingNoArticle(mf.Album)
	mf.Compilation = md.Bool(model.TagCompilation)

	mf.TrackNumber, _ = md.NumAndTotal(model.TagTrackNumber)
	mf.DiscNumber, _ = md.NumAndTotal(model.TagDiscNumber)
	mf.DiscSubtitle = md.String(model.TagDiscSubtitle)
	origDate := md.Date(model.TagOriginalDate)
	mf.OriginalYear, mf.OriginalDate = origDate.Year(), string(origDate)
	relDate := md.Date(model.TagReleaseDate)
	mf.ReleaseYear, mf.ReleaseDate = relDate.Year(), string(relDate)
	date := md.Date(model.TagRecordingDate)
	mf.Year, mf.Date = date.Year(), string(date)
	mf.CatalogNum = md.String(model.TagCatalogNumber)
	mf.MbzRecordingID = md.String(model.TagMusicBrainzRecordingID)
	mf.MbzReleaseTrackID = md.String(model.TagMusicBrainzTrackID)
	mf.MbzAlbumID = md.String(model.TagMusicBrainzAlbumID)
	mf.MbzReleaseGroupID = md.String(model.TagMusicBrainzReleaseGroupID)
	mf.RgAlbumPeak = md.Float(model.TagReplayGainAlbumPeak)
	mf.RgAlbumGain = md.Float(model.TagReplayGainAlbumGain)
	mf.RgTrackPeak = md.Float(model.TagReplayGainTrackPeak)
	mf.RgTrackGain = md.Float(model.TagReplayGainTrackGain)
	mf.Comment = md.String(model.TagComment)
	mf.Lyrics = md.String(model.TagLyrics)
	mf.Bpm = int(math.Round(md.Float(model.TagBPM)))
	mf.HasCoverArt = md.HasPicture()
	mf.Duration = md.Length()
	mf.BitRate = md.AudioProperties().BitRate
	mf.SampleRate = md.AudioProperties().SampleRate
	mf.Channels = md.AudioProperties().Channels
	mf.Path = md.FilePath()
	mf.Suffix = md.Suffix()
	mf.Size = md.Size()
	mf.BirthTime = md.BirthTime()
	mf.UpdatedAt = md.ModTime()

	mf.Participations = md.mapParticipations()
	mf.Artist = md.mapDisplayArtist(mf)
	mf.AlbumArtist = md.mapDisplayAlbumArtist(mf)

	mf.PID = md.trackPID(mf)
	mf.AlbumID = md.albumID(mf)

	// BFR Use PIDs for matching albums (AlbumPID method), but it does not need to be saved in the DB
	// BFR Album must also have a ArtistPID method, not saved to the DB as well.

	// BFR These IDs will go away once the UI handle multiple participants.
	// BFR For Legacy Subsonic compatibility, we will set them in the API handlers
	mf.ArtistID = mf.Participations.First(model.RoleArtist).ID
	mf.AlbumArtistID = mf.Participations.First(model.RoleAlbumArtist).ID

	// BFR What to do with sort/order artist names?
	mf.OrderArtistName = mf.Participations.First(model.RoleArtist).OrderArtistName
	mf.OrderAlbumArtistName = mf.Participations.First(model.RoleAlbumArtist).OrderArtistName
	mf.SortArtistName = mf.Participations.First(model.RoleArtist).SortArtistName
	mf.SortAlbumArtistName = mf.Participations.First(model.RoleAlbumArtist).SortArtistName

	// Remote tags that are first-class fields in the MediaFile struct
	removedTags := []model.TagName{
		model.TagAlbum, model.TagTitle, model.TagTrackNumber, model.TagDiscNumber, model.TagDiscSubtitle,
		model.TagGenre, model.TagMood, model.TagComment, model.TagAlbumSort, model.TagAlbumVersion,
		model.TagTitleSort, model.TagCompilation, model.TagGrouping, model.TagLyrics, model.TagRecordLabel,
		model.TagReleaseType, model.TagMedia, model.TagCatalogNumber, model.TagBPM, model.TagOriginalDate,
		model.TagReleaseDate, model.TagRecordingDate,

		// MusicBrainz IDs
		model.TagMusicBrainzRecordingID, model.TagMusicBrainzTrackID, model.TagMusicBrainzAlbumID,
		model.TagMusicBrainzReleaseGroupID, model.TagMusicBrainzAlbumArtistID, model.TagMusicBrainzArtistID,
		model.TagReplayGainAlbumPeak, model.TagReplayGainAlbumGain, model.TagReplayGainTrackPeak, model.TagReplayGainTrackGain,

		// Roles
		model.TagComposer, model.TagConductor, model.TagArranger, model.TagLyricist, model.TagRemixer,
		model.TagEngineer, model.TagMixer, model.TagProducer, model.TagDirector, model.TagDJMixer,
		model.TagPerformer, model.TagAlbumArtist, model.TagAlbumArtists, model.TagAlbumArtistSort,
		model.TagAlbumArtistsSort, model.TagTrackArtist, model.TagTrackArtists, model.TagTrackArtistSort,
		model.TagTrackArtistsSort, model.TagComposerSort,
	}
	for _, tag := range removedTags {
		delete(mf.Tags, tag)
	}

	return mf
}
