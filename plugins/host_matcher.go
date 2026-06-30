package plugins

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/plugins/types"
	"github.com/navidrome/navidrome/utils/slice"
)

type matcherServiceImpl struct {
	ds                model.DataStore
	hasFilesystemPerm bool
}

func newMatcherService(ds model.DataStore, hasFilesystemPerm bool) host.MatcherService {
	return &matcherServiceImpl{ds: ds, hasFilesystemPerm: hasFilesystemPerm}
}

func (s *matcherServiceImpl) MatchSongs(ctx context.Context, songs []types.SongRef) ([]*types.Track, error) {
	results := make([]*types.Track, len(songs))
	if len(songs) == 0 {
		return results, nil
	}

	agentSongs := slice.Map(songs, toAgentSong)

	matched, err := matcher.New(s.ds).MatchSongsIndexed(ctx, agentSongs)
	if err != nil {
		return nil, err
	}
	for i, mf := range matched {
		results[i] = s.toTrack(&mf)
	}
	return results, nil
}

// toTrack projects an internal MediaFile into the public Track DTO. The file
// Path is only exposed when the plugin holds library filesystem permission,
// matching how the Library host service gates path access.
func (s *matcherServiceImpl) toTrack(mf *model.MediaFile) *types.Track {
	t := &types.Track{
		ID:                mf.ID,
		LibraryID:         int32(mf.LibraryID),
		LibraryName:       mf.LibraryName,
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
		BitDepth:          ptrInt32(mf.BitDepth),
		Channels:          int32(mf.Channels),
		Codec:             mf.Codec,
		Comment:           mf.Comment,
		BPM:               ptrInt32(mf.BPM),
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
		RGAlbumGain:       mf.RGAlbumGain,
		RGAlbumPeak:       mf.RGAlbumPeak,
		RGTrackGain:       mf.RGTrackGain,
		RGTrackPeak:       mf.RGTrackPeak,
		BirthTime:         unixOrZero(mf.BirthTime),
		CreatedAt:         unixOrZero(mf.CreatedAt),
		UpdatedAt:         unixOrZero(mf.UpdatedAt),
	}
	if s.hasFilesystemPerm {
		t.Path = mf.Path
	}
	if len(mf.Genres) > 0 {
		t.Genres = slice.Map(mf.Genres, func(g model.Genre) string { return g.Name })
	}
	if len(mf.Tags) > 0 {
		t.Tags = make(map[string][]string, len(mf.Tags))
		for name, values := range mf.Tags {
			t.Tags[string(name)] = values
		}
	}
	if len(mf.Participants) > 0 {
		t.Participants = make(map[string][]types.ArtistRef, len(mf.Participants))
		for role, participants := range mf.Participants {
			t.Participants[role.String()] = slice.Map(participants, func(p model.Participant) types.ArtistRef {
				return types.ArtistRef{
					ID:       p.ID,
					Name:     p.Name,
					MBID:     p.MbzArtistID,
					SortName: p.SortArtistName,
					SubRole:  p.SubRole,
				}
			})
		}
	}
	return t
}

// toAgentSong converts a plugin-facing SongRef into the internal agents.Song the
// matcher consumes. Duration is normalized to milliseconds via DurationInMs, and
// artists are resolved via agentArtists.
func toAgentSong(s types.SongRef) agents.Song {
	return agents.Song{
		ID:        s.ID,
		Name:      s.Name,
		MBID:      s.MBID,
		ISRC:      s.ISRC,
		Album:     s.Album,
		AlbumMBID: s.AlbumMBID,
		Duration:  s.DurationInMs(), // agents.Song.Duration is ms; DurationInMs prefers DurationMs over the deprecated seconds field
		Artists:   agentArtists(s),
	}
}

// agentArtists maps a SongRef's artist information to agents.Artist. The richer
// Artists list takes precedence (per types.SongRef); otherwise the scalar
// Artist/ArtistMBID pair is used as a single-element list. Returns nil when no
// artist information is present.
func agentArtists(s types.SongRef) []agents.Artist {
	if len(s.Artists) > 0 {
		return slice.Map(s.Artists, func(a types.ArtistRef) agents.Artist {
			return agents.Artist{ID: a.ID, Name: a.Name, MBID: a.MBID}
		})
	}
	if s.Artist != "" || s.ArtistMBID != "" {
		return []agents.Artist{{Name: s.Artist, MBID: s.ArtistMBID}}
	}
	return nil
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// ptrInt32 maps a nullable *int from the model to a nullable *int32 for the DTO,
// preserving nil so plugins can tell "no value" from a real 0.
func ptrInt32(p *int) *int32 {
	if p == nil {
		return nil
	}
	v := int32(*p)
	return &v
}

var _ host.MatcherService = (*matcherServiceImpl)(nil)
