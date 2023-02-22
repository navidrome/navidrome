package playback

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/utils/singleton"
)

type PlaybackServer interface {
	Run(ctx context.Context) error
	GetDeviceForUser(user string) (*PlaybackDevice, error)
	GetMediaFile(id string) (*model.MediaFile, error)
	GetCtx() *context.Context
}

type playbackServer struct {
	ctx             *context.Context
	datastore       model.DataStore
	playbackDevices []PlaybackDevice
}

func GetInstance() PlaybackServer {
	return singleton.GetInstance(func() *playbackServer {
		return &playbackServer{}
	})
}

func (ps *playbackServer) Run(ctx context.Context) error {
	ps.datastore = persistence.New(db.Db())
	devices, err := ps.initDeviceStatus(conf.Server.Jukebox.Devices, conf.Server.Jukebox.Default)
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

func (ps *playbackServer) GetCtx() *context.Context {
	return ps.ctx
}

func (ps *playbackServer) initDeviceStatus(devices []conf.AudioDeviceDefinition, defaultDevice string) ([]PlaybackDevice, error) {
	pbDevices := make([]PlaybackDevice, len(devices))
	defaultDeviceFound := false

	for idx, audioDevice := range devices {
		if len(audioDevice) != 3 {
			return []PlaybackDevice{}, fmt.Errorf("audio device definition ought to contain 3 fields, found: %d ", len(audioDevice))
		}

		pbDevices[idx] = *NewPlaybackDevice(ps, audioDevice[0], audioDevice[1], audioDevice[2])

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
	for idx, audioDevice := range ps.playbackDevices {
		if audioDevice.Default {
			return &ps.playbackDevices[idx], nil
		}
	}
	return &PlaybackDevice{}, fmt.Errorf("no default device found")
}

func (ps *playbackServer) GetMediaFile(id string) (*model.MediaFile, error) {
	return ps.datastore.MediaFile(*ps.ctx).Get(id)
}

func (ps *playbackServer) GetDeviceForUser(user string) (*PlaybackDevice, error) {
	log.Debug("processing GetDevice")
	// README: here we might plug-in the user-device mapping one fine day
	device, err := ps.getDefaultDevice()
	if err != nil {
		return &PlaybackDevice{}, err
	}
	device.User = user
	return device, nil
}
