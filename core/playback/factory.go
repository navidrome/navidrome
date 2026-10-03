package playback

import (
	"context"

	"github.com/navidrome/navidrome/core/playback/mpv"
	"github.com/navidrome/navidrome/model"
)

type TrackFactory func(
	ctx context.Context,
	playbackDone chan bool,
	deviceName string,
	mf model.MediaFile,
) (Track, error)

func defaultTrackFactory(
	ctx context.Context,
	playbackDone chan bool,
	deviceName string,
	mf model.MediaFile,
) (Track, error) {
	return mpv.NewTrack(ctx, playbackDone, deviceName, mf)
}
