package playback

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/singleton"
)

type PlaybackServer interface {
	Run(ctx context.Context)
}

func GetInstance() PlaybackServer {
	return singleton.GetInstance(func() *playbackServer {
		return &playbackServer{}
	})
}

type playbackServer struct {
}

func (s *playbackServer) Run(ctx context.Context) {
	log.Info(ctx, "Verify devices")
	<-ctx.Done()
}
