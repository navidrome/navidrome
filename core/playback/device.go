package playback

import (
	"fmt"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
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
	Prepared             bool
	PlaybackQueue        *Queue
	Gain                 float32
	SampleRate           beep.SampleRate
	PlaybackDone         chan bool
	TrackSwitcherStarted bool
}

type DeviceStatus struct {
	CurrentIndex int
	Playing      bool
	Gain         float32
	Position     int
}

var EmptyStatus = DeviceStatus{CurrentIndex: -1, Playing: false, Gain: 0.5, Position: 0}

func (pd *PlaybackDevice) getStatus() DeviceStatus {
	return DeviceStatus{
		CurrentIndex: pd.PlaybackQueue.Index,
		Playing:      pd.isPlaying(),
		Gain:         pd.Gain,
		Position:     pd.Position(),
	}
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
		Prepared:             false,
		Gain:                 0.5,
		PlaybackQueue:        NewQueue(),
		PlaybackDone:         make(chan bool),
		TrackSwitcherStarted: false,
	}
}

func (pd *PlaybackDevice) String() string {
	return fmt.Sprintf("Name: %s, Gain: %.4f, Prepared: %t", pd.Name, pd.Gain, pd.Prepared)
}

func (pd *PlaybackDevice) Get() (model.MediaFiles, DeviceStatus, error) {
	log.Debug("processing Get action")
	return pd.PlaybackQueue.Get(), pd.getStatus(), nil
}

func (pd *PlaybackDevice) Status() (DeviceStatus, error) {
	log.Debug(fmt.Sprintf("processing Status action on: %s, queue: %s", pd, pd.PlaybackQueue))
	return pd.getStatus(), nil
}

// set is similar to a clear followed by a add, but will not change the currently playing track.
func (pd *PlaybackDevice) Set(ids []string) (DeviceStatus, error) {
	pd.Clear()
	return pd.Add(ids)
}

func (pd *PlaybackDevice) Start() (DeviceStatus, error) {
	log.Debug("processing Start action")

	currentTrack := pd.PlaybackQueue.Current()
	if currentTrack == nil {
		return EmptyStatus, nil
	}

	if !pd.Prepared {
		pd.loadTrack(*currentTrack)
	}

	if !pd.TrackSwitcherStarted {
		go func() {
			pd.trackSwitcher()
		}()
		pd.TrackSwitcherStarted = true
	}

	err := pd.SetPosition()
	if err != nil {
		return DeviceStatus{}, fmt.Errorf("could not set position to %d", pd.PlaybackQueue.Offset)
	}

	pd.Play()
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Stop() (DeviceStatus, error) {
	log.Debug("processing Stop action")
	pd.Pause()
	return pd.getStatus(), nil
}
func (pd *PlaybackDevice) Skip(index int, offset int) (DeviceStatus, error) {
	log.Debug("processing Skip action", "index", index, "offset", offset)

	wasPlaying := pd.isPlaying()

	if wasPlaying {
		pd.Pause()
	}
	pd.PlaybackQueue.SetIndex(index)
	pd.PlaybackQueue.SetOffset(offset)
	pd.Prepared = false

	if wasPlaying {
		pd.Start()
	}

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
	pd.PlaybackQueue.Clear()
	pd.Prepared = false
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
	difference := gain - pd.Gain
	log.Debug(fmt.Sprintf("processing SetGain action. Actual gain: %f, gain to set: %f, difference: %f", pd.Gain, gain, difference))

	pd.adjustVolume(float64(difference) * 5)
	pd.Gain = gain

	return pd.getStatus(), nil
}

func (pd *PlaybackDevice) adjustVolume(value float64) {
	speaker.Lock()
	pd.Volume.Volume += value
	speaker.Unlock()
}

func (pd *PlaybackDevice) Play() {
	speaker.Lock()
	pd.Ctrl.Paused = false
	speaker.Unlock()
}

func (pd *PlaybackDevice) Pause() {
	speaker.Lock()
	pd.Ctrl.Paused = true
	speaker.Unlock()
}

func (pd *PlaybackDevice) isPlaying() bool {
	return pd.Prepared && !pd.Ctrl.Paused
}

func (pd *PlaybackDevice) loadTrack(mf model.MediaFile) {
	contentType := mf.ContentType()
	log.Debug("loading track", "trackname", mf.Path, "mediatype", contentType)

	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error

	switch contentType {
	case "audio/mpeg":
		streamer, format, err = decodeMp3(mf.Path)
	case "audio/x-wav":
		streamer, format, err = decodeWAV(mf.Path)
	case "audio/mp4":
		streamer, format, err = decodeFLAC(*pd.ParentPlaybackServer.GetCtx(), mf.Path)
	default:
		log.Error("unsupported content type", "contentType", contentType)
		return
	}

	if err != nil {
		log.Error(err)
		return
	}

	log.Debug("Setting up audio device")
	pd.Ctrl = &beep.Ctrl{Streamer: streamer, Paused: true}
	pd.Volume = &effects.Volume{Streamer: pd.Ctrl, Base: 2}
	pd.Prepared = true
	pd.SampleRate = format.SampleRate

	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Error(err)
	}

	go func() {
		speaker.Play(beep.Seq(pd.Volume, beep.Callback(func() {
			pd.endOfStreamCallback()
		})))
	}()

}

func (pd *PlaybackDevice) endOfStreamCallback() {
	log.Info("Hitting end-of-stream")
	pd.PlaybackDone <- true
	log.Info("sended playbackDone event to channel")
}

func (pd *PlaybackDevice) trackSwitcher() {
	log.Info("Starting trackSwitcher goroutine")
	for {
		log.Info("waiting for switching event on channel")
		<-pd.PlaybackDone
		log.Info("track switching detected")
		pd.Pause()
		log.Info("Did pausing stream")
	}
}

// Position returns the playback position in seconds
func (pd *PlaybackDevice) Position() int {
	streamer, ok := pd.Ctrl.Streamer.(beep.StreamSeeker)
	if ok {
		position := pd.SampleRate.D(streamer.Position())
		posSecs := position.Round(time.Second).Seconds()
		return int(posSecs)
	}
	return 0
}

func (pd *PlaybackDevice) SetPosition() error {
	streamer, ok := pd.Ctrl.Streamer.(beep.StreamSeeker)
	if ok {
		sampleRatePerSecond := pd.SampleRate.N(time.Second)
		nextPosition := sampleRatePerSecond * pd.PlaybackQueue.Offset
		log.Debug("Samplerate per second", "samplerate", sampleRatePerSecond)
		return streamer.Seek(nextPosition)
	}
	return fmt.Errorf("streamer is not seekable")
}
