package playback

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/log"
)

type Track interface {
	IsPlaying() bool
	SetVolume(value float32) // Used to control the playback volume. A float value between 0.0 and 1.0.
	Pause()
	Unpause()
	Position() int
	SetPosition(offset int) error
	Close()
	String() string
	LoadFile(append bool, playNow bool)
}

type playbackDevice struct {
	serviceCtx           context.Context
	ParentPlaybackServer PlaybackServer
	Default              bool
	User                 string
	Name                 string
	DeviceName           string
	PlaybackQueue        *Queue
	Gain                 float32
	PlaybackDone         chan bool
	ActiveTrack          Track
}

type DeviceStatus struct {
	CurrentIndex int
	Playing      bool
	Gain         float32
	Position     int
}

const DefaultGain float32 = 1.0

func (pd *playbackDevice) getStatus() DeviceStatus {
	currentIndex := pd.ParentPlaybackServer.GetPlaylistPosition()

	pos := 0
	isPlaying := false
	if currentIndex >= 0 {
		pos = pd.ParentPlaybackServer.Position()
		isPlaying = pd.isPlaying()
	}
	return DeviceStatus{
		CurrentIndex: currentIndex,
		Playing:      isPlaying,
		Gain:         pd.Gain,
		Position:     pos,
	}
}

// NewPlaybackDevice creates a new playback device which implements all the basic Jukebox mode commands defined here:
// http://www.subsonic.org/pages/api.jsp#jukeboxControl
// Starts the trackSwitcher goroutine for the device.
func NewPlaybackDevice(ctx context.Context, playbackServer PlaybackServer, name string, deviceName string) *playbackDevice {
	return &playbackDevice{
		serviceCtx:           ctx,
		ParentPlaybackServer: playbackServer,
		User:                 "",
		Name:                 name,
		DeviceName:           deviceName,
		Gain:                 DefaultGain,
		PlaybackQueue:        NewQueue(),
		PlaybackDone:         make(chan bool),
	}
}

func (pd *playbackDevice) String() string {
	return fmt.Sprintf("Name: %s, Gain: %.4f", pd.Name, pd.Gain)
}

func (pd *playbackDevice) Get(ctx context.Context) ([]string, DeviceStatus, error) {
	log.Debug(ctx, "Processing Get action", "device", pd)
	return pd.ParentPlaybackServer.GetPlaylistIDs(), pd.getStatus(), nil
}

func (pd *playbackDevice) Status(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, fmt.Sprintf("processing Status action on: %s", pd))
	return pd.getStatus(), nil
}

// Set is similar to a clear followed by a add, but will not change the currently playing track.
func (pd *playbackDevice) Set(ctx context.Context, ids []string) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Set action", "ids", ids, "device", pd)

	_, err := pd.Clear(ctx)
	if err != nil {
		log.Error(ctx, "error setting tracks", ids)
		return pd.getStatus(), err
	}
	return pd.Add(ctx, ids)
}

func (pd *playbackDevice) Start(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Start action", "device", pd)
	_, err := pd.ParentPlaybackServer.Start()
	if err != nil {
		log.Error(ctx, "error starting playback", err)
		return pd.getStatus(), err
	}

	return pd.getStatus(), nil
}

func (pd *playbackDevice) Stop(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Stop action", "device", pd)
	_, err := pd.ParentPlaybackServer.Stop()
	if err != nil {
		log.Error(ctx, "error stopping playback", err)
		return pd.getStatus(), err
	}
	return pd.getStatus(), nil
}

func (pd *playbackDevice) Skip(ctx context.Context, index int, offset int) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Skip action", "index", index, "offset", offset, "device", pd)
	_, err := pd.ParentPlaybackServer.Skip(index, offset)
	if err != nil {
		log.Error(ctx, "error skipping to track", index, offset)
		return pd.getStatus(), err
	}
	return pd.getStatus(), nil
}

func (pd *playbackDevice) Add(ctx context.Context, ids []string) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Add action", "ids", ids, "device", pd)
	if len(ids) < 1 {
		return pd.getStatus(), nil
	}

	for _, id := range ids {
		mf, err := pd.ParentPlaybackServer.GetMediaFile(id)
		if err != nil {
			return pd.getStatus(), err
		}
		log.Debug(ctx, "Found mediafile: "+mf.Path)
		_, err = pd.ParentPlaybackServer.LoadFile(mf, true, false)
		if err != nil {
			return pd.getStatus(), err
		}
	}

	return pd.getStatus(), nil
}

func (pd *playbackDevice) Clear(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Clear action", "device", pd)
	_, err := pd.ParentPlaybackServer.Clear()
	if err != nil {
		return pd.getStatus(), err
	}
	return pd.getStatus(), nil
}

func (pd *playbackDevice) Remove(ctx context.Context, index int) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Remove action", "index", index, "device", pd)
	_, err := pd.ParentPlaybackServer.Remove(index)
	if err != nil {
		return pd.getStatus(), err
	}
	return pd.getStatus(), nil
}

func (pd *playbackDevice) Shuffle(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Shuffle action", "device", pd)
	_, err := pd.ParentPlaybackServer.Shuffle()
	if err != nil {
		return pd.getStatus(), err
	}
	return pd.getStatus(), nil
}

// SetGain is used to control the playback volume. A float value between 0.0 and 1.0.
func (pd *playbackDevice) SetGain(ctx context.Context, gain float32) (DeviceStatus, error) {
	log.Debug(ctx, "Processing SetGain action", "newGain", gain, "device", pd)
	_, err := pd.ParentPlaybackServer.SetGain(gain)
	if err != nil {
		return pd.getStatus(), err
	}
	pd.Gain = gain

	return pd.getStatus(), nil
}

func (pd *playbackDevice) isPlaying() bool {
	return pd.ParentPlaybackServer.IsPlaying()
}
