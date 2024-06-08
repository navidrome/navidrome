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

	mf.Participations = md.participations()
	mf.Artist = md.displayArtist(mf)
	mf.AlbumArtist = md.displayAlbumArtist(mf)

	//mf.Genre = md.String(Genre)
	mf.Compilation = md.Bool(Compilation)
	mf.TrackNumber, _ = md.NumAndTotal(TrackNumber)
	mf.DiscNumber, _ = md.NumAndTotal(DiscNumber)
	mf.DiscSubtitle = md.String(DiscSubtitle)
	origDate := md.Date(OriginalDate)
	mf.OriginalYear, mf.OriginalDate = origDate.Year(), string(origDate)
	relDate := md.Date(ReleaseDate)
	mf.ReleaseYear, mf.ReleaseDate = relDate.Year(), string(relDate)
	mf.Year, mf.Date = relDate.Year(), string(relDate) // TODO Remove?
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

	mf.PID = md.trackPID(mf)
	mf.AlbumID = md.albumID()

	return mf
}
