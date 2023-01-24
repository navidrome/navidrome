package playback

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/singleton"
)

type PlaybackServer interface {
	Run(ctx context.Context) error
}

func GetInstance() PlaybackServer {
	return singleton.GetInstance(func() *playbackServer {
		return &playbackServer{}
	})
}

type playbackServer struct {
}

func (s *playbackServer) Run(ctx context.Context) error {
	err := verifyConfiguration(conf.Server.Jukebox.Devices, conf.Server.Jukebox.Default)
	if err != nil {
		return err
	}
	log.Info(ctx, "Using audio device: "+conf.Server.Jukebox.Default)

	<-ctx.Done()
	return nil
}

func verifyConfiguration(devices []conf.AudioDeviceDefinition, defaultDevice string) error {
	for _, audioDevice := range devices {
		if len(audioDevice) != 3 {
			return fmt.Errorf("audio device definition ought to contain 3 fields, found: %d ", len(audioDevice))
		}
		if audioDevice[0] == defaultDevice {
			return nil
		}
	}
	return fmt.Errorf("default audio device not found in list of devices: %s", defaultDevice)
}
