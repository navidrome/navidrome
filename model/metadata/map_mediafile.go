package metadata

import (
	"math"

	"github.com/navidrome/navidrome/model"
)

func (md Metadata) ToMediaFile() model.MediaFile {
	mf := model.MediaFile{
		Tags: md.tags,
	}
	mf.ID = md.trackID()
	mf.PID = mf.ID
	mf.Title = md.String(Title)
	mf.Album = md.String(Album)
	mf.Artist = md.String(TrackArtist)
	mf.AlbumArtist = md.String(AlbumArtist)
	mf.Genre = md.String(Genre)
	mf.Compilation = md.Bool(Compilation)
	mf.TrackNumber, _ = md.NumAndTotal(TrackNumber)
	mf.DiscNumber, _ = md.NumAndTotal(DiscNumber)
	mf.Duration = md.Length()
	mf.BitRate = md.AudioProperties().BitRate
	mf.SampleRate = md.AudioProperties().SampleRate
	mf.Channels = md.AudioProperties().Channels
	mf.Path = md.FilePath() // TODO Use relative path
	mf.Suffix = md.Suffix()
	mf.Size = md.Size()
	mf.HasCoverArt = md.HasPicture()
	mf.SortTitle = md.String(TitleSort)
	mf.SortAlbumName = md.String(AlbumSort)
	mf.SortArtistName = md.String(TrackArtistSort)
	mf.SortAlbumArtistName = md.String(AlbumArtistSort)
	origDate := md.Date(OriginalDate)
	mf.OriginalYear, mf.OriginalDate = origDate.Year(), string(origDate)
	relDate := md.Date(ReleaseDate)
	mf.ReleaseYear, mf.ReleaseDate = relDate.Year(), string(relDate)
	mf.Year, mf.Date = relDate.Year(), string(relDate) // TODO Remove?
	mf.CatalogNum = md.String(CatalogNumber)
	mf.MbzRecordingID = md.String(MusicBrainzRecordingID)
	mf.MbzReleaseTrackID = md.String(MusicBrainzReleaseTrackID)
	mf.MbzAlbumID = md.String(MusicBrainzAlbumID)
	mf.MbzArtistID = md.String(MusicBrainzArtistID)
	mf.MbzAlbumArtistID = md.String(MusicBrainzAlbumArtistID)
	mf.RgAlbumPeak = md.Float(ReplayGainAlbumPeak)
	mf.RgAlbumGain = md.Float(ReplayGainAlbumGain)
	mf.RgTrackPeak = md.Float(ReplayGainTrackPeak)
	mf.RgTrackGain = md.Float(ReplayGainTrackGain)
	mf.Comment = md.String(Comment)
	mf.Lyrics = md.String(Lyrics)
	mf.Bpm = int(math.Round(md.Float(BPM)))
	mf.CreatedAt = md.BirthTime()
	mf.UpdatedAt = md.ModTime()

	return mf
}
