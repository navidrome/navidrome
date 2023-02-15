package playback

import (
	"context"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
)

type PlaybackDevice struct {
	ParentPlaybackServer PlaybackServer
	Default              bool
	User                 string
	Name                 string
	Method               string
	DeviceName           string
	Ctrl                 *beep.Ctrl
	Volume               *effects.Volume
	PlaybackQueue        Queue
}

type DeviceStatus struct {
	CurrentIndex int
	Playing      bool
	Gain         float64
	Position     int
}

func NewPlaybackDevice(playbackServer PlaybackServer, name string, method string, deviceName string) *PlaybackDevice {
	return &PlaybackDevice{
		ParentPlaybackServer: playbackServer,
		User:                 "",
		Name:                 name,
		Method:               method,
		DeviceName:           deviceName,
		Ctrl:                 &beep.Ctrl{},
		Volume:               &effects.Volume{},
		PlaybackQueue:        Queue{},
	}
}

func (pd *PlaybackDevice) Get(user string) (responses.JukeboxPlaylist, error) {
	log.Debug("processing Get action")
	return responses.JukeboxPlaylist{}, nil
}

func (pd *PlaybackDevice) Status(user string) (DeviceStatus, error) {
	log.Debug("processing Status action")
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Set(user string, ids []string) (DeviceStatus, error) {
	log.Debug("processing Set action.")

	mf, err := pd.ParentPlaybackServer.GetMediaFile(ids[0])
	if err != nil {
		return DeviceStatus{}, err
	}

	log.Debug("Found mediafile: " + mf.Path)

	pd.prepareSong(mf.Path)

	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Start(user string) (DeviceStatus, error) {
	log.Debug("processing Start action")
	pd.playHead()
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Stop(user string) (DeviceStatus, error) {
	log.Debug("processing Stop action")
	pd.pauseHead()
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Skip(user string, index int, offset int) (DeviceStatus, error) {
	log.Debug("processing Skip action")
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Add(user string, ids []string) (DeviceStatus, error) {
	log.Debug("processing Add action")
	// pd.Playlist.Entry = append(pd.Playlist.Entry, child)
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Clear(user string) (DeviceStatus, error) {
	log.Debug("processing Clear action")
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Remove(user string, index int) (DeviceStatus, error) {
	log.Debug("processing Remove action")
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Shuffle(user string) (DeviceStatus, error) {
	log.Debug("processing Shuffle action")
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) SetGain(user string, gain float64) (DeviceStatus, error) {
	log.Debug("processing SetGain action")

	speaker.Lock()
	// pd.Volume.Silent = !pd.Volume.Silent
	pd.Volume.Volume -= 0.1
	speaker.Unlock()

	return pd.getStatus(), nil
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

func (pd *PlaybackDevice) getStatus() DeviceStatus {
	return DeviceStatus{
		CurrentIndex: 0,
		Playing:      !pd.Ctrl.Paused,
		Gain:         0,
		Position:     pd.Position(),
	}
}

func (pd *PlaybackDevice) Position() int {
	streamer, ok := pd.Ctrl.Streamer.(beep.StreamSeeker)
	if ok {
		return streamer.Position()
	}
	return 0
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
