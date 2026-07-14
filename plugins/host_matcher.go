package plugins

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/plugins/types"
	"github.com/navidrome/navidrome/utils/slice"
)

type matcherServiceImpl struct {
	ds                model.DataStore
	hasFilesystemPerm bool
	users             userAccess
	libs              libraryAccess
}

func newMatcherService(ds model.DataStore, hasFilesystemPerm bool, users userAccess, libs libraryAccess) host.MatcherService {
	return &matcherServiceImpl{
		ds:                ds,
		hasFilesystemPerm: hasFilesystemPerm,
		users:             users,
		libs:              libs,
	}
}

func (s *matcherServiceImpl) MatchSongs(ctx context.Context, songs []types.SongRef, opts host.MatchOptions) ([]*types.Track, error) {
	results := make([]*types.Track, len(songs))
	if len(songs) == 0 {
		return results, nil
	}

	// Fail closed when the plugin has no library scope, rather than matching nothing.
	if !s.libs.configured() {
		return nil, fmt.Errorf("matcher: no libraries configured for this plugin")
	}

	// Set the user context explicitly so the match never inherits the request user
	// of whatever invoked the plugin: a username scopes to that user (loading their
	// annotations and library access), and an unscoped match runs as admin so only
	// the plugin's own library scope constrains the results.
	scoped := opts.Username != ""
	if scoped {
		usr, err := s.users.resolve(ctx, s.ds, opts.Username)
		if err != nil {
			return nil, fmt.Errorf("matcher: %w", err)
		}
		ctx = request.WithUser(ctx, *usr)
	} else {
		ctx = adminContext(ctx)
	}

	agentSongs := slice.Map(songs, songRefToAgentSong)

	matched, err := matcher.New(s.ds).MatchSongsIndexed(ctx, agentSongs)
	if err != nil {
		return nil, err
	}
	// The plugin's library scope is a second, independent authorization on top of the context
	// user's own access, so it's applied here rather than in-query: the unscoped path runs as
	// admin (which applyLibraryFilter skips), and folding s.libs into the context user would
	// conflate the two scopes instead of intersecting them.
	for i, mf := range matched {
		// Drop tracks outside the plugin's library scope, leaving that index unmatched.
		if !s.libs.contains(mf.LibraryID) {
			continue
		}
		results[i] = s.toTrack(&mf, scoped)
	}
	return results, nil
}

// toTrack projects a MediaFile into the public Track DTO. Path needs filesystem
// permission; per-user annotations are only set for a scoped match.
func (s *matcherServiceImpl) toTrack(mf *model.MediaFile, scoped bool) *types.Track {
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
		AverageRating:     mf.AverageRating, // aggregate, not user-scoped
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
		// Flatten the role→artists map into a role-tagged list, in stable role order.
		roles := slices.SortedFunc(maps.Keys(mf.Participants), func(a, b model.Role) int {
			return cmp.Compare(a.String(), b.String())
		})
		for _, role := range roles {
			for _, p := range mf.Participants[role] {
				t.Participants = append(t.Participants, types.ArtistRef{
					ID:       p.ID,
					Name:     p.Name,
					MBID:     p.MbzArtistID,
					SortName: p.SortArtistName,
					Role:     role.String(),
					SubRole:  p.SubRole,
				})
			}
		}
	}
	if scoped {
		t.Starred = mf.Starred
		t.StarredAt = unixPtr(mf.StarredAt)
		t.Rating = int32(mf.Rating)
		t.PlayCount = mf.PlayCount
		t.PlayDate = unixPtr(mf.PlayDate)
	}
	return t
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// unixPtr maps a nullable time to Unix seconds, keeping nil distinct from the epoch.
func unixPtr(t *time.Time) *int64 {
	if t == nil || t.IsZero() {
		return nil
	}
	return new(t.Unix())
}

// ptrInt32 narrows a nullable *int to *int32, keeping nil distinct from a real 0.
func ptrInt32(p *int) *int32 {
	if p == nil {
		return nil
	}
	return new(int32(*p))
}

var _ host.MatcherService = (*matcherServiceImpl)(nil)
