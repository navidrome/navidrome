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
	Get(user string) (responses.JukeboxPlaylist, error)
	Status(user string) (responses.JukeboxStatus, error)
	Set(user string, id string) (responses.JukeboxStatus, error)
	Start(user string) (responses.JukeboxStatus, error)
	Stop(user string) (responses.JukeboxStatus, error)
	Skip(user string, index int64, offset int64) (responses.JukeboxStatus, error)
	Add(user string, id string) (responses.JukeboxStatus, error)
	Clear(user string) (responses.JukeboxStatus, error)
	Remove(user string, index int64) (responses.JukeboxStatus, error)
	Shuffle(user string) (responses.JukeboxStatus, error)
	SetGain(user string, gain float64) (responses.JukeboxStatus, error)
}

type PlaybackDevice struct {
	Owner      string
	Name       string
	Method     string
	DeviceName string
	Playlist   responses.JukeboxPlaylist
	Status     responses.JukeboxStatus
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
	err := verifyConfiguration(conf.Server.Jukebox.Devices, conf.Server.Jukebox.Default)
	if err != nil {
		return err
	}

	ps.playbackDevices = initDeviceStatus(conf.Server.Jukebox.Devices)
	log.Info(ctx, fmt.Sprintf("%d audio devices found", len(conf.Server.Jukebox.Devices)))
	log.Info(ctx, "Using default audio device: "+conf.Server.Jukebox.Default)

	ps.ctx = &ctx

	<-ctx.Done()
	return nil
}

func initDeviceStatus(devices []conf.AudioDeviceDefinition) []PlaybackDevice {
	pbDevices := make([]PlaybackDevice, len(devices))
	for idx, audioDevice := range devices {
		pbDevices[idx] = PlaybackDevice{
			Owner:      "",
			Name:       audioDevice[0],
			Method:     audioDevice[1],
			DeviceName: audioDevice[2],
			Playlist:   responses.JukeboxPlaylist{},
			Status:     responses.JukeboxStatus{},
		}
	}
	return pbDevices
}

func (ps *playbackServer) Get(user string) (responses.JukeboxPlaylist, error) {
	log.Debug("processing Get action")
	return responses.JukeboxPlaylist{}, nil
}

func (ps *playbackServer) Status(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Status action")
	return responses.JukeboxStatus{}, nil
}
func (ps *playbackServer) Set(user string, id string) (responses.JukeboxStatus, error) {
	log.Debug("processing Set action")
	return responses.JukeboxStatus{}, nil
}
func (ps *playbackServer) Start(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Start action")
	// playSong("tests/fixtures/test.mp3")
	return responses.JukeboxStatus{}, nil
}
func (ps *playbackServer) Stop(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Stop action")
	return responses.JukeboxStatus{}, nil
}
func (ps *playbackServer) Skip(user string, index int64, offset int64) (responses.JukeboxStatus, error) {
	log.Debug("processing Skip action")
	return responses.JukeboxStatus{}, nil
}
func (ps *playbackServer) Add(user string, id string) (responses.JukeboxStatus, error) {
	log.Debug("processing Add action")
	return responses.JukeboxStatus{}, nil
}
func (ps *playbackServer) Clear(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Clear action")
	return responses.JukeboxStatus{}, nil
}
func (ps *playbackServer) Remove(user string, index int64) (responses.JukeboxStatus, error) {
	log.Debug("processing Remove action")
	return responses.JukeboxStatus{}, nil
}
func (ps *playbackServer) Shuffle(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Shuffle action")
	return responses.JukeboxStatus{}, nil
}
func (ps *playbackServer) SetGain(user string, gain float64) (responses.JukeboxStatus, error) {
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
