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
	mf.SortTitle = md.String(TitleSort)
	mf.SortAlbumName = md.String(AlbumSort)
	mf.OrderTitle = str.SanitizeFieldForSorting(mf.Title)
	mf.OrderAlbumName = str.SanitizeFieldForSortingNoArticle(mf.Album)
	mf.Compilation = md.Bool(Compilation)

	mf.TrackNumber, _ = md.NumAndTotal(TrackNumber)
	mf.DiscNumber, _ = md.NumAndTotal(DiscNumber)
	mf.DiscSubtitle = md.String(DiscSubtitle)
	origDate := md.Date(OriginalDate)
	mf.OriginalYear, mf.OriginalDate = origDate.Year(), string(origDate)
	relDate := md.Date(ReleaseDate)
	mf.ReleaseYear, mf.ReleaseDate = relDate.Year(), string(relDate)
	date := md.Date(RecordingDate)
	mf.Year, mf.Date = date.Year(), string(date)
	mf.CatalogNum = md.String(CatalogNumber)
	mf.MbzRecordingID = md.String(MusicBrainzRecordingID)
	mf.MbzReleaseTrackID = md.String(MusicBrainzTrackID)
	mf.MbzAlbumID = md.String(MusicBrainzAlbumID)
	mf.RgAlbumPeak = md.Float(ReplayGainAlbumPeak)
	mf.RgAlbumGain = md.Float(ReplayGainAlbumGain)
	mf.RgTrackPeak = md.Float(ReplayGainTrackPeak)
	mf.RgTrackGain = md.Float(ReplayGainTrackGain)
	mf.Comment = md.String(Comment)
	mf.Lyrics = md.String(Lyrics)
	mf.Bpm = int(math.Round(md.Float(BPM)))
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
	mf.AlbumID = md.albumID()

	// TODO These IDs will go away once the UI handle multiple participants.
	// For Legacy Subsonic compatibility, we will set them in the API handlers
	// FIXME This is wrong? See Yussef and Caetano
	mf.ArtistID = mf.Participations.First(model.RoleArtist).ID
	mf.AlbumArtistID = mf.Participations.First(model.RoleAlbumArtist).ID

	return mf
}
