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
	Get(user string) responses.JukeboxPlaylist
	Status(user string) responses.JukeboxStatus
	Set(user string, id string) responses.JukeboxStatus
	Start(user string) responses.JukeboxStatus
	Stop(user string) responses.JukeboxStatus
	Skip(user string, index int64, offset int64) responses.JukeboxStatus
	Add(user string, id string) responses.JukeboxStatus
	Clear(user string) responses.JukeboxStatus
	Remove(user string, index int64) responses.JukeboxStatus
	Shuffle(user string) responses.JukeboxStatus
	SetGain(user string, gain float64) responses.JukeboxStatus
}

func GetInstance() PlaybackServer {
	return singleton.GetInstance(func() *playbackServer {
		return &playbackServer{}
	})
}

type playbackServer struct {
	ctx *context.Context
}

func (ps *playbackServer) Run(ctx context.Context) error {
	err := verifyConfiguration(conf.Server.Jukebox.Devices, conf.Server.Jukebox.Default)
	if err != nil {
		return err
	}
	log.Info(ctx, "Using audio device: "+conf.Server.Jukebox.Default)

	ps.ctx = &ctx

	<-ctx.Done()
	return nil
}

func (ps *playbackServer) Get(user string) responses.JukeboxPlaylist {
	log.Debug("processing Get action")
	return responses.JukeboxPlaylist{}
}

func (ps *playbackServer) Status(user string) responses.JukeboxStatus {
	log.Debug("processing Status action")
	return responses.JukeboxStatus{}
}
func (ps *playbackServer) Set(user string, id string) responses.JukeboxStatus {
	log.Debug("processing Set action")
	return responses.JukeboxStatus{}
}
func (ps *playbackServer) Start(user string) responses.JukeboxStatus {
	log.Debug("processing Start action")
	playSong("tests/fixtures/test.mp3")
	return responses.JukeboxStatus{}
}
func (ps *playbackServer) Stop(user string) responses.JukeboxStatus {
	log.Debug("processing Stop action")
	return responses.JukeboxStatus{}
}
func (ps *playbackServer) Skip(user string, index int64, offset int64) responses.JukeboxStatus {
	log.Debug("processing Skip action")
	return responses.JukeboxStatus{}
}
func (ps *playbackServer) Add(user string, id string) responses.JukeboxStatus {
	log.Debug("processing Add action")
	return responses.JukeboxStatus{}
}
func (ps *playbackServer) Clear(user string) responses.JukeboxStatus {
	log.Debug("processing Clear action")
	return responses.JukeboxStatus{}
}
func (ps *playbackServer) Remove(user string, index int64) responses.JukeboxStatus {
	log.Debug("processing Remove action")
	return responses.JukeboxStatus{}
}
func (ps *playbackServer) Shuffle(user string) responses.JukeboxStatus {
	log.Debug("processing Shuffle action")
	return responses.JukeboxStatus{}
}
func (ps *playbackServer) SetGain(user string, gain float64) responses.JukeboxStatus {
	log.Debug("processing SetGain action")
	return responses.JukeboxStatus{}
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
