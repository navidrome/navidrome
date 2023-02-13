package playback

import (
	"context"
	"fmt"

	"github.com/faiface/beep"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/utils/singleton"
)

type PlaybackServer interface {
	Run(ctx context.Context) error
	GetDevice(user string) (*PlaybackDevice, error)
}

type playbackServer struct {
	ctx             *context.Context
	playbackDevices []PlaybackDevice
}

func GetInstance() PlaybackServer {
	return singleton.GetInstance(func() *playbackServer {
		return &playbackServer{}
	})
}

func (ps *playbackServer) Run(ctx context.Context) error {
	dataStore := persistence.New(db.Db())

	devices, err := initDeviceStatus(ctx, dataStore, conf.Server.Jukebox.Devices, conf.Server.Jukebox.Default)
	ps.playbackDevices = devices

	if err != nil {
		return err
	}
	log.Info(ctx, fmt.Sprintf("%d audio devices found", len(conf.Server.Jukebox.Devices)))
	log.Info(ctx, "Using default audio device: "+conf.Server.Jukebox.Default)

	ps.ctx = &ctx

	<-ctx.Done()
	return nil
}

func initDeviceStatus(ctx context.Context, ds model.DataStore, devices []conf.AudioDeviceDefinition, defaultDevice string) ([]PlaybackDevice, error) {
	pbDevices := make([]PlaybackDevice, len(devices))
	defaultDeviceFound := false

	for idx, audioDevice := range devices {
		if len(audioDevice) != 3 {
			return []PlaybackDevice{}, fmt.Errorf("audio device definition ought to contain 3 fields, found: %d ", len(audioDevice))
		}

		pbDevices[idx] = PlaybackDevice{
			DataStore:  ds,
			Ctx:        ctx,
			User:       "",
			Name:       audioDevice[0],
			Method:     audioDevice[1],
			DeviceName: audioDevice[2],
			Ctrl:       &beep.Ctrl{},
		}

		if audioDevice[0] == defaultDevice {
			pbDevices[idx].Default = true
			defaultDeviceFound = true
		}
	}

	if !defaultDeviceFound {
		return []PlaybackDevice{}, fmt.Errorf("default device name not found: %s ", defaultDevice)
	}
	return pbDevices, nil
}

func (ps *playbackServer) getDefaultDevice() (*PlaybackDevice, error) {
	for _, audioDevice := range ps.playbackDevices {
		if audioDevice.Default {
			return &audioDevice, nil
		}
	}
	return &PlaybackDevice{}, fmt.Errorf("no default device found")
}

func (ps *playbackServer) GetDevice(user string) (*PlaybackDevice, error) {
	log.Debug("processing GetDevice")
	// README: here we might plug-in the user-device mapping one fine day
	return ps.getDefaultDevice()
}
