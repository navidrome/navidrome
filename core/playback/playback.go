package playback

import (
	"context"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/singleton"

	"os"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

type PlaybackServer interface {
	Run(ctx context.Context) error
	GetDevice(user string) (*PlaybackDevice, error)
}

type PlaybackDevice struct {
	Default       bool
	User          string
	Name          string
	Method        string
	DeviceName    string
	Playlist      responses.JukeboxPlaylist
	JukeboxStatus responses.JukeboxStatus
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
	devices, err := initDeviceStatus(conf.Server.Jukebox.Devices, conf.Server.Jukebox.Default)
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

func initDeviceStatus(devices []conf.AudioDeviceDefinition, defaultDevice string) ([]PlaybackDevice, error) {
	pbDevices := make([]PlaybackDevice, len(devices))
	defaultDeviceFound := false

	for idx, audioDevice := range devices {
		if len(audioDevice) != 3 {
			return []PlaybackDevice{}, fmt.Errorf("audio device definition ought to contain 3 fields, found: %d ", len(audioDevice))
		}

		pbDevices[idx] = PlaybackDevice{
			User:          "",
			Name:          audioDevice[0],
			Method:        audioDevice[1],
			DeviceName:    audioDevice[2],
			Playlist:      responses.JukeboxPlaylist{},
			JukeboxStatus: responses.JukeboxStatus{},
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

func (pd *PlaybackDevice) Get(user string) (responses.JukeboxPlaylist, error) {
	log.Debug("processing Get action")
	return responses.JukeboxPlaylist{}, nil
}

func (pd *PlaybackDevice) Status(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Status action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Set(user string, id string) (responses.JukeboxStatus, error) {
	log.Debug("processing Set action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Start(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Start action")
	playSong("tests/fixtures/test.mp3")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Stop(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Stop action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Skip(user string, index int64, offset int64) (responses.JukeboxStatus, error) {
	log.Debug("processing Skip action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Add(user string, id string) (responses.JukeboxStatus, error) {
	log.Debug("processing Add action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Clear(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Clear action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Remove(user string, index int64) (responses.JukeboxStatus, error) {
	log.Debug("processing Remove action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Shuffle(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Shuffle action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) SetGain(user string, gain float64) (responses.JukeboxStatus, error) {
	log.Debug("processing SetGain action")
	return responses.JukeboxStatus{}, nil
}

func playSong(songname string) {
	log.Debug("Playing song: " + songname)
	f, err := os.Open(songname)
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	defer streamer.Close()

	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done
}
