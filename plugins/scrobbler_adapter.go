package plugins

import (
	"context"
	"strings"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

// CapabilityScrobbler indicates the plugin can receive scrobble events.
// Detected when the plugin exports at least one of the scrobbler functions.
const CapabilityScrobbler Capability = "Scrobbler"

// Scrobbler function names (snake_case as per design)
const (
	FuncScrobblerIsAuthorized = "nd_scrobbler_is_authorized"
	FuncScrobblerNowPlaying   = "nd_scrobbler_now_playing"
	FuncScrobblerScrobble     = "nd_scrobbler_scrobble"
)

func init() {
	registerCapability(
		CapabilityScrobbler,
		FuncScrobblerIsAuthorized,
		FuncScrobblerNowPlaying,
		FuncScrobblerScrobble,
	)
}

// ScrobblerPlugin is an adapter that wraps an Extism plugin and implements
// the scrobbler.Scrobbler interface for scrobbling to external services.
type ScrobblerPlugin struct {
	name           string
	plugin         *plugin
	allowedUserIDs []string            // User IDs this plugin can access (from DB configuration)
	allUsers       bool                // If true, plugin can access all users
	userIDMap      map[string]struct{} // Cached map for fast lookups
}

// IsAuthorized checks if the user is authorized with this scrobbler.
// First checks if the user is allowed to use this plugin (server-side),
// then delegates to the plugin for service-specific authorization.
func (s *ScrobblerPlugin) IsAuthorized(ctx context.Context, userId string) bool {
	// First check server-side authorization based on plugin configuration
	if !s.isUserAllowed(userId) {
		return false
	}

	// Then delegate to the plugin for service-specific authorization
	username := getUsernameFromContext(ctx)
	input := capabilities.IsAuthorizedRequest{
		Username: username,
	}

	result, err := callPluginFunction[capabilities.IsAuthorizedRequest, bool](ctx, s.plugin, FuncScrobblerIsAuthorized, input)
	if err != nil {
		return false
	}

	return result
}

// isUserAllowed checks if the given user ID is allowed to use this plugin.
func (s *ScrobblerPlugin) isUserAllowed(userId string) bool {
	if s.allUsers {
		return true
	}
	if len(s.allowedUserIDs) == 0 {
		return false
	}
	_, ok := s.userIDMap[userId]
	return ok
}

// NowPlaying sends a now playing notification to the scrobbler
func (s *ScrobblerPlugin) NowPlaying(ctx context.Context, userId string, track *model.MediaFile, position int) error {
	username := getUsernameFromContext(ctx)
	input := capabilities.NowPlayingRequest{
		Username: username,
		Track:    mediaFileToTrackInfo(track),
		Position: int32(position),
	}

	err := callPluginFunctionNoOutput(ctx, s.plugin, FuncScrobblerNowPlaying, input)
	return mapScrobblerError(err)
}

// Scrobble submits a scrobble to the scrobbler
func (s *ScrobblerPlugin) Scrobble(ctx context.Context, userId string, sc scrobbler.Scrobble) error {
	username := getUsernameFromContext(ctx)
	input := capabilities.ScrobbleRequest{
		Username:  username,
		Track:     mediaFileToTrackInfo(&sc.MediaFile),
		Timestamp: sc.TimeStamp.Unix(),
	}

	err := callPluginFunctionNoOutput(ctx, s.plugin, FuncScrobblerScrobble, input)
	return mapScrobblerError(err)
}

// getUsernameFromContext extracts the username from the request context
func getUsernameFromContext(ctx context.Context) string {
	if user, ok := request.UserFrom(ctx); ok {
		return user.UserName
	}
	return ""
}

// mediaFileToTrackInfo converts a model.MediaFile to capabilities.TrackInfo
func mediaFileToTrackInfo(mf *model.MediaFile) capabilities.TrackInfo {
	return capabilities.TrackInfo{
		ID:                mf.ID,
		Title:             mf.Title,
		Album:             mf.Album,
		Artist:            mf.Artist,
		AlbumArtist:       mf.AlbumArtist,
		Artists:           participantsToArtistRefs(mf.Participants[model.RoleArtist]),
		AlbumArtists:      participantsToArtistRefs(mf.Participants[model.RoleAlbumArtist]),
		Duration:          mf.Duration,
		TrackNumber:       int32(mf.TrackNumber),
		DiscNumber:        int32(mf.DiscNumber),
		MBZRecordingID:    mf.MbzRecordingID,
		MBZAlbumID:        mf.MbzAlbumID,
		MBZReleaseGroupID: mf.MbzReleaseGroupID,
		MBZReleaseTrackID: mf.MbzReleaseTrackID,
	}
}

// participantsToArtistRefs converts a ParticipantList to a slice of ArtistRef
func participantsToArtistRefs(participants model.ParticipantList) []capabilities.ArtistRef {
	refs := make([]capabilities.ArtistRef, len(participants))
	for i, p := range participants {
		refs[i] = capabilities.ArtistRef{
			ID:   p.ID,
			Name: p.Name,
			MBID: p.MbzArtistID,
		}
	}
	return refs
}

// mapScrobblerError converts plugin errors to scrobbler errors based on error message, as errors are returned as
// strings from plugins.
func mapScrobblerError(err error) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, capabilities.ScrobblerErrorNotAuthorized.Error()):
		return scrobbler.ErrNotAuthorized
	case strings.Contains(errMsg, capabilities.ScrobblerErrorRetryLater.Error()):
		return scrobbler.ErrRetryLater
	case strings.Contains(errMsg, capabilities.ScrobblerErrorUnrecoverable.Error()):
		return scrobbler.ErrUnrecoverable
	default:
		return scrobbler.ErrUnrecoverable
	}
}

// Verify interface implementation at compile time
var _ scrobbler.Scrobbler = (*ScrobblerPlugin)(nil)
