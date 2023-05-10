package playback

import (
	"context"
	"fmt"
	"os"
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
	TrackLoaded          bool
	ActiveStream         beep.StreamSeekCloser
	TempfileToCleanup    string
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

// NewPlaybackDevice creates a new playback device which implements all the basic Jukebox mode commands defined here:
// http://www.subsonic.org/pages/api.jsp#jukeboxControl
func NewPlaybackDevice(playbackServer PlaybackServer, name string, method string, deviceName string) *PlaybackDevice {
	return &PlaybackDevice{
		ParentPlaybackServer: playbackServer,
		User:                 "",
		Name:                 name,
		Method:               method,
		DeviceName:           deviceName,
		Ctrl:                 &beep.Ctrl{Paused: true},
		Volume:               &effects.Volume{},
		TrackLoaded:          false,
		ActiveStream:         nil,
		Gain:                 0.5,
		PlaybackQueue:        NewQueue(),
		PlaybackDone:         make(chan bool),
		TrackSwitcherStarted: false,
	}
}

func (pd *PlaybackDevice) String() string {
	return fmt.Sprintf("Name: %s, Gain: %.4f, Prepared: %t", pd.Name, pd.Gain, pd.TrackLoaded)
}

func (pd *PlaybackDevice) Get(ctx context.Context) (model.MediaFiles, DeviceStatus, error) {
	log.Debug(ctx, "processing Get action")
	return pd.PlaybackQueue.Get(), pd.getStatus(), nil
}

func (pd *PlaybackDevice) Status(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, fmt.Sprintf("processing Status action on: %s, queue: %s", pd, pd.PlaybackQueue))
	return pd.getStatus(), nil
}

// set is similar to a clear followed by a add, but will not change the currently playing track.
func (pd *PlaybackDevice) Set(ctx context.Context, ids []string) (DeviceStatus, error) {
	_, err := pd.Clear(ctx)
	if err != nil {
		log.Error(ctx, "error setting tracks", ids)
		return pd.getStatus(), err
	}
	return pd.Add(ctx, ids)
}

func (pd *PlaybackDevice) Start(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "processing Start action")
	return pd.startTrack()
}

func (pd *PlaybackDevice) startTrack() (DeviceStatus, error) {
	currentTrack := pd.PlaybackQueue.Current()
	if currentTrack == nil {
		log.Debug("startTrack() no current track found")
		return EmptyStatus, nil
	}

	if pd.TrackLoaded {
		pd.play()
		return pd.getStatus(), nil
	} else {
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

	pd.play()
	return pd.getStatus(), nil
}

func (pd *PlaybackDevice) Stop(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "processing Stop action")
	pd.pause()
	return pd.getStatus(), nil
}

func (pd *PlaybackDevice) Skip(ctx context.Context, index int, offset int) (DeviceStatus, error) {
	log.Debug(ctx, "processing Skip action", "index", index, "offset", offset)

	wasPlaying := pd.isPlaying()

	if wasPlaying {
		pd.pause()
	}

	if index != pd.PlaybackQueue.Index {
		pd.closeTrack()
		pd.PlaybackQueue.SetIndex(index)
	}

	err := pd.PlaybackQueue.SetOffset(offset)
	if err != nil {
		log.Error(ctx, "error setting offset", err)
		return pd.getStatus(), err
	}

	if wasPlaying {
		_, err = pd.Start(ctx)
		if err != nil {
			log.Error(ctx, "error starting new track after skipping")
			return pd.getStatus(), err
		}
	}

	return pd.getStatus(), nil
}

func (pd *PlaybackDevice) Add(ctx context.Context, ids []string) (DeviceStatus, error) {
	log.Debug(ctx, "processing Add action")

	items := model.MediaFiles{}

	for _, id := range ids {
		mf, err := pd.ParentPlaybackServer.GetMediaFile(id)
		if err != nil {
			return DeviceStatus{}, err
		}
		log.Debug(ctx, "Found mediafile: "+mf.Path)
		items = append(items, *mf)
	}
	pd.PlaybackQueue.Add(items)

	return pd.getStatus(), nil
}

func (pd *PlaybackDevice) Clear(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, fmt.Sprintf("processing Clear action on: %s", pd))
	pd.PlaybackQueue.Clear()
	pd.closeTrack()
	return pd.getStatus(), nil
}

func (pd *PlaybackDevice) Remove(ctx context.Context, index int) (DeviceStatus, error) {
	log.Debug(ctx, "processing Remove action")
	// pausing if attempting to remove running track
	if pd.isPlaying() && pd.PlaybackQueue.Index == index {
		_, err := pd.Stop(ctx)
		if err != nil {
			log.Error(ctx, "error stopping running track")
			return pd.getStatus(), err
		}
	}

	if index > -1 && index < pd.PlaybackQueue.Size() {
		pd.PlaybackQueue.Remove(index)
	} else {
		log.Error(ctx, "Index to remove out of range: "+fmt.Sprint(index))
	}
	return pd.getStatus(), nil
}

func (pd *PlaybackDevice) Shuffle(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "processing Shuffle action")
	if pd.PlaybackQueue.Size() > 1 {
		pd.PlaybackQueue.Shuffle()
	}
	return pd.getStatus(), nil
}

func (pd *PlaybackDevice) SetGain(ctx context.Context, gain float32) (DeviceStatus, error) {
	difference := gain - pd.Gain
	log.Debug(ctx, fmt.Sprintf("processing SetGain action. Actual gain: %f, gain to set: %f, difference: %f", pd.Gain, gain, difference))

	pd.adjustVolume(float64(difference) * 5)
	pd.Gain = gain

	return pd.getStatus(), nil
}

func (pd *PlaybackDevice) adjustVolume(value float64) {
	speaker.Lock()
	pd.Volume.Volume += value
	speaker.Unlock()
}

func (pd *PlaybackDevice) play() {
	speaker.Lock()
	pd.Ctrl.Paused = false
	speaker.Unlock()
}

func (pd *PlaybackDevice) pause() {
	speaker.Lock()
	pd.Ctrl.Paused = true
	speaker.Unlock()
}

func (pd *PlaybackDevice) isPlaying() bool {
	return pd.TrackLoaded && !pd.Ctrl.Paused
}

func (pd *PlaybackDevice) loadTrack(mf model.MediaFile) {
	contentType := mf.ContentType()
	log.Debug("loading track", "trackname", mf.Path, "mediatype", contentType)

	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error
	var tmpfileToCleanup = ""

	switch contentType {
	case "audio/mpeg":
		streamer, format, err = decodeMp3(mf.Path)
	case "audio/x-wav":
		streamer, format, err = decodeWAV(mf.Path)
	case "audio/mp4":
		streamer, format, tmpfileToCleanup, err = decodeFLAC(*pd.ParentPlaybackServer.GetCtx(), mf.Path)
	default:
		log.Error("unsupported content type", "contentType", contentType)
		return
	}

	if err != nil {
		log.Error(err)
		return
	}

	// save running stream for closing when switching tracks
	pd.ActiveStream = streamer
	pd.TempfileToCleanup = tmpfileToCleanup

	log.Debug("Setting up audio device")
	pd.Ctrl = &beep.Ctrl{Streamer: streamer, Paused: true}
	pd.Volume = &effects.Volume{Streamer: pd.Ctrl, Base: 2}
	pd.TrackLoaded = true
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
}

func (pd *PlaybackDevice) trackSwitcher() {
	log.Info("Starting trackSwitcher goroutine")
	for {
		<-pd.PlaybackDone
		log.Info("track switching detected")
		// pd.pause()
		pd.closeTrack()

		if !pd.PlaybackQueue.IsAtLastElement() {
			log.Debug("Switching to next song", "queue", pd.PlaybackQueue.String())
			pd.PlaybackQueue.IncreaseIndex()
			err := pd.PlaybackQueue.SetOffset(0)
			if err != nil {
				log.Error("error setting offset of next track to zero")
			}
			_, err = pd.startTrack()
			if err != nil {
				log.Error("error starting track #", pd.PlaybackQueue.Index)
			}
		}
	}
}

func (pd *PlaybackDevice) closeTrack() {
	pd.TrackLoaded = false
	if pd.ActiveStream != nil {
		log.Debug("closing activ stream")
		pd.ActiveStream.Close()
	}

	if pd.TempfileToCleanup != "" {
		log.Debug("Removing tempfile", "tmpfilename", pd.TempfileToCleanup)
		err := os.Remove(pd.TempfileToCleanup)
		if err != nil {
			log.Error("error cleaning up tempfile: ", pd.TempfileToCleanup)
		}
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
		log.Debug("SetPosition", "samplerate", sampleRatePerSecond, "nextPosition", nextPosition)
		return streamer.Seek(nextPosition)
	}
	return fmt.Errorf("streamer is not seekable")
}
