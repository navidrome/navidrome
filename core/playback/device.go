package playback

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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
	PlaybackQueue        *Queue
	Gain                 float32
}

type DeviceStatus struct {
	CurrentIndex int
	Playing      bool
	Gain         float32
	Position     int
}

func NewPlaybackDevice(playbackServer PlaybackServer, name string, method string, deviceName string) *PlaybackDevice {
	return &PlaybackDevice{
		ParentPlaybackServer: playbackServer,
		User:                 "",
		Name:                 name,
		Method:               method,
		DeviceName:           deviceName,
		Ctrl:                 &beep.Ctrl{Paused: true},
		Volume:               &effects.Volume{},
		Gain:                 0,
		PlaybackQueue:        NewQueue(),
	}
}

func (pd *PlaybackDevice) String() string {
	return fmt.Sprintf("Name: %s, Gain: %f", pd.Name, pd.Gain)
}

func (pd *PlaybackDevice) Get() (model.MediaFiles, DeviceStatus, error) {
	log.Debug("processing Get action")
	return pd.PlaybackQueue.Get(), pd.getStatus(), nil
}

func (pd *PlaybackDevice) Status() (DeviceStatus, error) {
	log.Debug(fmt.Sprintf("processing Status action on: %s, queue: %s", pd, pd.PlaybackQueue))
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Set(ids []string) (DeviceStatus, error) {
	pd.Clear()
	return pd.Add(ids)
}

func (pd *PlaybackDevice) Start() (DeviceStatus, error) {
	log.Debug("processing Start action")

	currentSong := pd.PlaybackQueue.Current()
	if currentSong == nil {
		return DeviceStatus{}, fmt.Errorf("there is no current song")
	}

	pd.prepareSong(currentSong.Path)
	pd.playHead()
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Stop() (DeviceStatus, error) {
	log.Debug("processing Stop action")
	pd.pauseHead()
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Skip(index int, offset int) (DeviceStatus, error) {
	log.Debug("processing Skip action")
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Add(ids []string) (DeviceStatus, error) {
	log.Debug("processing Add action")

	items := model.MediaFiles{}

	for _, id := range ids {
		mf, err := pd.ParentPlaybackServer.GetMediaFile(id)
		if err != nil {
			return DeviceStatus{}, err
		}
		log.Debug("Found mediafile: " + mf.Path)
		items = append(items, *mf)
	}
	pd.PlaybackQueue.Add(items)

	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Clear() (DeviceStatus, error) {
	log.Debug(fmt.Sprintf("processing Clear action on: %s", pd))
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Remove(index int) (DeviceStatus, error) {
	log.Debug("processing Remove action")
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Shuffle() (DeviceStatus, error) {
	log.Debug("processing Shuffle action")
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) SetGain(gain float32) (DeviceStatus, error) {
	log.Debug(fmt.Sprintf("processing SetGain action on: %s", pd))

	pd.Gain = gain

	speaker.Lock()
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

	pd.Ctrl = &beep.Ctrl{Streamer: streamer, Paused: true}
	pd.Volume = &effects.Volume{Streamer: pd.Ctrl, Base: 2}

	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		speaker.Play(pd.Volume)
	}()

}

func (pd *PlaybackDevice) getStatus() DeviceStatus {
	return DeviceStatus{
		CurrentIndex: pd.PlaybackQueue.Index,
		Playing:      !pd.Ctrl.Paused,
		Gain:         pd.Gain,
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
