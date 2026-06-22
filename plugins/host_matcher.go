package plugins

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/utils/slice"
)

type matcherServiceImpl struct {
	ds model.DataStore
}

func newMatcherService(ds model.DataStore) host.MatcherService {
	return &matcherServiceImpl{ds: ds}
}

func (s *matcherServiceImpl) MatchSongs(ctx context.Context, songs []host.MatchSong) ([]*host.Track, error) {
	results := make([]*host.Track, len(songs))
	if len(songs) == 0 {
		return results, nil
	}

	agentSongs := slice.Map(songs, func(s host.MatchSong) agents.Song {
		return agents.Song{
			ID:         s.ID,
			Name:       s.Name,
			MBID:       s.MBID,
			ISRC:       s.ISRC,
			Artist:     s.Artist,
			ArtistMBID: s.ArtistMBID,
			Album:      s.Album,
			AlbumMBID:  s.AlbumMBID,
			Duration:   s.DurationMs, // agents.Song.Duration is ms, same unit as host.MatchSong.DurationMs
		}
	})

	matched, err := matcher.New(s.ds).MatchSongsIndexed(ctx, agentSongs)
	if err != nil {
		return nil, err
	}
	for i, mf := range matched {
		results[i] = toTrack(&mf)
	}
	return results, nil
}

// toTrack projects an internal MediaFile into the public Track DTO.
func toTrack(mf *model.MediaFile) *host.Track {
	t := &host.Track{
		ID:                mf.ID,
		LibraryID:         int32(mf.LibraryID),
		LibraryName:       mf.LibraryName,
		Path:              mf.Path,
		Missing:           mf.Missing,
		Title:             mf.Title,
		Album:             mf.Album,
		Artist:            mf.Artist,
		AlbumArtist:       mf.AlbumArtist,
		AlbumID:           mf.AlbumID,
		SortTitle:         mf.SortTitle,
		SortAlbumName:     mf.SortAlbumName,
		SortArtistName:    mf.SortArtistName,
		TrackNumber:       int32(mf.TrackNumber),
		DiscNumber:        int32(mf.DiscNumber),
		DiscSubtitle:      mf.DiscSubtitle,
		Year:              int32(mf.Year),
		Date:              mf.Date,
		OriginalYear:      int32(mf.OriginalYear),
		OriginalDate:      mf.OriginalDate,
		ReleaseYear:       int32(mf.ReleaseYear),
		ReleaseDate:       mf.ReleaseDate,
		Size:              mf.Size,
		Suffix:            mf.Suffix,
		Duration:          float64(mf.Duration),
		BitRate:           int32(mf.BitRate),
		SampleRate:        int32(mf.SampleRate),
		Channels:          int32(mf.Channels),
		Codec:             mf.Codec,
		Comment:           mf.Comment,
		ExplicitStatus:    mf.ExplicitStatus,
		CatalogNum:        mf.CatalogNum,
		Compilation:       mf.Compilation,
		HasCoverArt:       mf.HasCoverArt,
		MbzRecordingID:    mf.MbzRecordingID,
		MbzReleaseTrackID: mf.MbzReleaseTrackID,
		MbzAlbumID:        mf.MbzAlbumID,
		MbzReleaseGroupID: mf.MbzReleaseGroupID,
		MbzAlbumType:      mf.MbzAlbumType,
		MbzAlbumComment:   mf.MbzAlbumComment,
		BirthTime:         unixOrZero(mf.BirthTime),
		CreatedAt:         unixOrZero(mf.CreatedAt),
		UpdatedAt:         unixOrZero(mf.UpdatedAt),
	}
	if mf.BitDepth != nil {
		t.BitDepth = int32(*mf.BitDepth)
	}
	if mf.BPM != nil {
		t.BPM = int32(*mf.BPM)
	}
	if mf.RGAlbumGain != nil {
		t.RGAlbumGain = *mf.RGAlbumGain
	}
	if mf.RGAlbumPeak != nil {
		t.RGAlbumPeak = *mf.RGAlbumPeak
	}
	if mf.RGTrackGain != nil {
		t.RGTrackGain = *mf.RGTrackGain
	}
	if mf.RGTrackPeak != nil {
		t.RGTrackPeak = *mf.RGTrackPeak
	}
	for _, g := range mf.Genres {
		t.Genres = append(t.Genres, g.Name)
	}
	if len(mf.Tags) > 0 {
		t.Tags = make(map[string][]string, len(mf.Tags))
		for name, values := range mf.Tags {
			t.Tags[string(name)] = values
		}
	}
	if len(mf.Participants) > 0 {
		t.Participants = make(map[string][]host.Artist, len(mf.Participants))
		for role, participants := range mf.Participants {
			artists := make([]host.Artist, 0, len(participants))
			for _, p := range participants {
				artists = append(artists, host.Artist{
					ID:          p.ID,
					Name:        p.Name,
					SortName:    p.SortArtistName,
					MbzArtistID: p.MbzArtistID,
					SubRole:     p.SubRole,
				})
			}
			t.Participants[role.String()] = artists
		}
	}
	return t
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

var _ host.MatcherService = (*matcherServiceImpl)(nil)
