package plugins

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
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
	name   string
	plugin *plugin
}

// IsAuthorized checks if the user is authorized with this scrobbler
func (s *ScrobblerPlugin) IsAuthorized(ctx context.Context, userId string) bool {
	username := getUsernameFromContext(ctx)
	input := scrobblerAuthInput{
		UserID:   userId,
		Username: username,
	}

	result, err := callPluginFunction[scrobblerAuthInput, scrobblerAuthOutput](ctx, s.plugin, FuncScrobblerIsAuthorized, input)
	if err != nil {
		return false
	}

	return result.Authorized
}

// NowPlaying sends a now playing notification to the scrobbler
func (s *ScrobblerPlugin) NowPlaying(ctx context.Context, userId string, track *model.MediaFile, position int) error {
	username := getUsernameFromContext(ctx)
	input := scrobblerNowPlayingInput{
		UserID:   userId,
		Username: username,
		Track:    mediaFileToTrackInfo(track),
		Position: position,
	}

	result, err := callPluginFunction[scrobblerNowPlayingInput, scrobblerOutput](ctx, s.plugin, FuncScrobblerNowPlaying, input)
	if err != nil {
		return err
	}

	return mapScrobblerError(result)
}

// Scrobble submits a scrobble to the scrobbler
func (s *ScrobblerPlugin) Scrobble(ctx context.Context, userId string, sc scrobbler.Scrobble) error {
	username := getUsernameFromContext(ctx)
	input := scrobblerScrobbleInput{
		UserID:    userId,
		Username:  username,
		Track:     mediaFileToTrackInfo(&sc.MediaFile),
		Timestamp: sc.TimeStamp.Unix(),
	}

	result, err := callPluginFunction[scrobblerScrobbleInput, scrobblerOutput](ctx, s.plugin, FuncScrobblerScrobble, input)
	if err != nil {
		return err
	}

	return mapScrobblerError(result)
}

// getUsernameFromContext extracts the username from the request context
func getUsernameFromContext(ctx context.Context) string {
	if user, ok := request.UserFrom(ctx); ok {
		return user.UserName
	}
	return ""
}

// mediaFileToTrackInfo converts a model.MediaFile to scrobblerTrackInfo
func mediaFileToTrackInfo(mf *model.MediaFile) scrobblerTrackInfo {
	return scrobblerTrackInfo{
		ID:                mf.ID,
		Title:             mf.Title,
		Album:             mf.Album,
		Artist:            mf.Artist,
		AlbumArtist:       mf.AlbumArtist,
		Duration:          mf.Duration,
		TrackNumber:       mf.TrackNumber,
		DiscNumber:        mf.DiscNumber,
		MbzRecordingID:    mf.MbzRecordingID,
		MbzAlbumID:        mf.MbzAlbumID,
		MbzArtistID:       mf.MbzArtistID,
		MbzReleaseGroupID: mf.MbzReleaseGroupID,
		MbzAlbumArtistID:  mf.MbzAlbumArtistID,
		MbzReleaseTrackID: mf.MbzReleaseTrackID,
	}
}

// mapScrobblerError converts the plugin output error to a scrobbler error
func mapScrobblerError(output scrobblerOutput) error {
	switch output.ErrorType {
	case scrobblerErrorNone, "":
		return nil
	case scrobblerErrorNotAuthorized:
		if output.Error != "" {
			return fmt.Errorf("%w: %s", scrobbler.ErrNotAuthorized, output.Error)
		}
		return scrobbler.ErrNotAuthorized
	case scrobblerErrorRetryLater:
		if output.Error != "" {
			return fmt.Errorf("%w: %s", scrobbler.ErrRetryLater, output.Error)
		}
		return scrobbler.ErrRetryLater
	case scrobblerErrorUnrecoverable:
		if output.Error != "" {
			return fmt.Errorf("%w: %s", scrobbler.ErrUnrecoverable, output.Error)
		}
		return scrobbler.ErrUnrecoverable
	default:
		if output.Error != "" {
			return fmt.Errorf("unknown error type %q: %s", output.ErrorType, output.Error)
		}
		return fmt.Errorf("unknown error type: %s", output.ErrorType)
	}
}

// Verify interface implementation at compile time
var _ scrobbler.Scrobbler = (*ScrobblerPlugin)(nil)
