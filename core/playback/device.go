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
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
)

type PlaybackDevice struct {
	Ctx        context.Context
	DataStore  model.DataStore
	Default    bool
	User       string
	Name       string
	Method     string
	DeviceName string
	Ctrl       *beep.Ctrl
}

func (pd *PlaybackDevice) Get(user string) (responses.JukeboxPlaylist, error) {
	log.Debug("processing Get action")
	return responses.JukeboxPlaylist{}, nil
}

func (pd *PlaybackDevice) Status(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Status action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Set(user string, ids []string) (responses.JukeboxStatus, error) {
	log.Debug("processing Set action.")

	mf, err := pd.DataStore.MediaFile(pd.Ctx).Get(ids[0])
	if err != nil {
		return responses.JukeboxStatus{}, err
	}

	log.Debug("Found mediafile: " + mf.Path)

	pd.prepareSong(mf.Path)

	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Start(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Start action")
	pd.playHead()
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Stop(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Stop action")
	pd.pauseHead()
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Skip(user string, index int, offset int) (responses.JukeboxStatus, error) {
	log.Debug("processing Skip action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Add(user string, ids []string) (responses.JukeboxStatus, error) {
	log.Debug("processing Add action")
	// pd.Playlist.Entry = append(pd.Playlist.Entry, child)
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Clear(user string) (responses.JukeboxStatus, error) {
	log.Debug("processing Clear action")
	return responses.JukeboxStatus{}, nil
}
func (pd *PlaybackDevice) Remove(user string, index int) (responses.JukeboxStatus, error) {
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

func (pd *PlaybackDevice) playHead() {
	speaker.Lock()
	pd.Ctrl.Paused = false
	speaker.Unlock()
}

func (pd *PlaybackDevice) pauseHead() {
	speaker.Lock()
	pd.Ctrl.Paused = true
	speaker.Unlock()
}

func (pd *PlaybackDevice) prepareSong(songname string) {
	log.Debug("Playing song: " + songname)
	f, err := os.Open(songname)
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	pd.Ctrl.Streamer = streamer
	pd.Ctrl.Paused = true

	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		speaker.Play(pd.Ctrl)
	}()

}

func getTranscoding(ctx context.Context) (format string, bitRate int) {
	if trc, ok := request.TranscodingFrom(ctx); ok {
		format = trc.TargetFormat
	}
	if plr, ok := request.PlayerFrom(ctx); ok {
		bitRate = plr.MaxBitRate
	}
	return
}
