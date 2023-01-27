package playback

import (
	"context"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
)

type PlaybackDevice struct {
	Ctx           context.Context
	DataStore     model.DataStore
	Default       bool
	User          string
	Name          string
	Method        string
	DeviceName    string
	Playlist      responses.JukeboxPlaylist
	JukeboxStatus responses.JukeboxStatus
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

	mf, err := pd.DataStore.MediaFile(pd.Ctx).Get(id)
	if err != nil {
		return responses.JukeboxStatus{}, err
	}

	log.Debug("Found mediafile: " + mf.Path)

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
