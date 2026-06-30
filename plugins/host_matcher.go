package plugins

import (
	"context"
	"fmt"
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
	allowedUsers      []string // user IDs this plugin may scope a match to
	allUsers          bool     // if true, the plugin may scope to any user
	libs              libraryAccess
}

func newMatcherService(ds model.DataStore, hasFilesystemPerm bool, allowedUsers []string, allUsers bool, libs libraryAccess) host.MatcherService {
	return &matcherServiceImpl{
		ds:                ds,
		hasFilesystemPerm: hasFilesystemPerm,
		allowedUsers:      allowedUsers,
		allUsers:          allUsers,
		libs:              libs,
	}
}

func (s *matcherServiceImpl) MatchSongs(ctx context.Context, songs []types.SongRef, opts host.MatchOptions) ([]*types.Track, error) {
	results := make([]*types.Track, len(songs))
	if len(songs) == 0 {
		return results, nil
	}

	// Scope the match to a user when requested: their annotations are loaded and
	// their library access applies. Without a username the match runs unscoped.
	scoped := opts.Username != ""
	if scoped {
		usr, err := s.resolveUser(ctx, opts.Username)
		if err != nil {
			return nil, err
		}
		ctx = request.WithUser(ctx, *usr)
	}

	agentSongs := slice.Map(songs, songRefToAgentSong)

	matched, err := matcher.New(s.ds).MatchSongsIndexed(ctx, agentSongs)
	if err != nil {
		return nil, err
	}
	for i, mf := range matched {
		// Enforce the plugin's library access: never return a track the plugin
		// may not see, leaving that index unmatched (nil).
		if !s.libs.contains(mf.LibraryID) {
			continue
		}
		results[i] = s.toTrack(&mf, scoped)
	}
	return results, nil
}

// resolveUser looks up the user for a match request and enforces the plugin's
// user-access permission (allowedUsers/allUsers). It returns an error for an
// unknown username or one the plugin is not allowed to act as.
func (s *matcherServiceImpl) resolveUser(ctx context.Context, username string) (*model.User, error) {
	usr, err := s.ds.User(ctx).FindByUsername(username)
	if err != nil || usr == nil {
		return nil, fmt.Errorf("matcher: user %q not found", username)
	}
	if !s.allUsers && !slices.Contains(s.allowedUsers, usr.ID) {
		return nil, fmt.Errorf("matcher: plugin is not allowed to match as user %q", username)
	}
	return usr, nil
}

// toTrack projects an internal MediaFile into the public Track DTO. The file
// Path is only exposed when the plugin holds library filesystem permission,
// matching how the Library host service gates path access. Per-user annotations
// are only exposed when the match was scoped to a user.
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

// unixPtr maps a nullable *time.Time to a nullable Unix-epoch-seconds pointer,
// returning nil for a nil or zero time so plugins can tell "no value" apart from
// the epoch.
func unixPtr(t *time.Time) *int64 {
	if t == nil || t.IsZero() {
		return nil
	}
	v := t.Unix()
	return &v
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
