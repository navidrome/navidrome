package playback

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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
}

type playbackDevice struct {
	mu                   sync.RWMutex
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
	trackFactory         TrackFactory
	startTrackSwitcher   sync.Once
}

// receiverVolumeCallbackTrack is an optional internal capability implemented
// by Cast-backed tracks to surface receiver-originated volume changes.
type receiverVolumeCallbackTrack interface {
	SetReceiverVolumeHandler(func(float32))
}

type DeviceStatus struct {
	CurrentIndex int
	Playing      bool
	Gain         float32
	Position     int
}

const DefaultGain float32 = 1.0

func (pd *playbackDevice) getStatus() DeviceStatus {
	pd.mu.RLock()
	index := pd.PlaybackQueue.Index
	gain := pd.Gain
	track := pd.ActiveTrack
	pd.mu.RUnlock()

	pos := 0
	playing := false
	if track != nil {
		pos = track.Position()
		playing = track.IsPlaying()
	}
	return DeviceStatus{
		CurrentIndex: index,
		Playing:      playing,
		Gain:         gain,
		Position:     pos,
	}
}

// NewPlaybackDevice creates a new playback device which implements all the basic Jukebox mode commands defined here:
// http://www.subsonic.org/pages/api.jsp#jukeboxControl
// Starts the trackSwitcher goroutine for the device.
func NewPlaybackDevice(ctx context.Context, playbackServer PlaybackServer, name string, deviceName string, trackFactory TrackFactory) *playbackDevice {
	return &playbackDevice{
		serviceCtx:           ctx,
		ParentPlaybackServer: playbackServer,
		User:                 "",
		Name:                 name,
		DeviceName:           deviceName,
		Gain:                 DefaultGain,
		PlaybackQueue:        NewQueue(),
		PlaybackDone:         make(chan bool),
		trackFactory:         trackFactory,
	}
}

func (pd *playbackDevice) String() string {
	pd.mu.RLock()
	name := pd.Name
	gain := pd.Gain
	track := pd.ActiveTrack
	pd.mu.RUnlock()
	return fmt.Sprintf("Name: %s, Gain: %.4f, Loaded track: %s", name, gain, track)
}

func (pd *playbackDevice) Get(ctx context.Context) (model.MediaFiles, DeviceStatus, error) {
	log.Debug(ctx, "Processing Get action", "device", pd)
	pd.mu.RLock()
	items := append(model.MediaFiles(nil), pd.PlaybackQueue.Get()...)
	pd.mu.RUnlock()
	return items, pd.getStatus(), nil
}

func (pd *playbackDevice) Status(ctx context.Context) (DeviceStatus, error) {
	pd.mu.RLock()
	queueString := pd.PlaybackQueue.String()
	pd.mu.RUnlock()
	log.Debug(ctx, fmt.Sprintf("processing Status action on: %s, queue: %s", pd, queueString))
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

	pd.startTrackSwitcher.Do(func() {
		log.Info(ctx, "Starting trackSwitcher goroutine")
		// Start one trackSwitcher goroutine with each device
		go func() {
			pd.trackSwitcherGoroutine()
		}()
	})

	pd.mu.RLock()
	activeTrack := pd.ActiveTrack
	queueEmpty := pd.PlaybackQueue.IsEmpty()
	queueIndex := pd.PlaybackQueue.Index
	pd.mu.RUnlock()

	if activeTrack != nil {
		if activeTrack.IsPlaying() {
			log.Debug("trying to start an already playing track")
		} else {
			activeTrack.Unpause()
		}
	} else if !queueEmpty {
		err := pd.switchActiveTrackByIndex(queueIndex)
		if err != nil {
			return pd.getStatus(), err
		}
		pd.mu.RLock()
		activeTrack = pd.ActiveTrack
		pd.mu.RUnlock()
		if activeTrack != nil {
			activeTrack.Unpause()
		}
	}

	return pd.getStatus(), nil
}

func (pd *playbackDevice) Stop(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Stop action", "device", pd)
	pd.mu.RLock()
	activeTrack := pd.ActiveTrack
	pd.mu.RUnlock()
	if activeTrack != nil {
		activeTrack.Pause()
	}
	return pd.getStatus(), nil
}

func (pd *playbackDevice) Skip(ctx context.Context, index int, offset int) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Skip action", "index", index, "offset", offset, "device", pd)

	pd.mu.RLock()
	activeTrack := pd.ActiveTrack
	currentIndex := pd.PlaybackQueue.Index
	pd.mu.RUnlock()

	wasPlaying := activeTrack != nil && activeTrack.IsPlaying()
	if activeTrack != nil && wasPlaying {
		activeTrack.Pause()
	}

	if index != currentIndex && activeTrack != nil {
		pd.mu.Lock()
		if pd.ActiveTrack == activeTrack {
			pd.ActiveTrack = nil
		}
		pd.mu.Unlock()
		activeTrack.Close()
		activeTrack = nil
	}

	if activeTrack == nil {
		err := pd.switchActiveTrackByIndex(index)
		if err != nil {
			return pd.getStatus(), err
		}
		pd.mu.RLock()
		activeTrack = pd.ActiveTrack
		pd.mu.RUnlock()
	}

	if activeTrack == nil {
		return pd.getStatus(), errors.New("could not get current track")
	}
	if err := activeTrack.SetPosition(offset); err != nil {
		log.Error(ctx, "error setting position", err)
		return pd.getStatus(), err
	}

	if wasPlaying {
		if !activeTrack.IsPlaying() {
			activeTrack.Unpause()
		}
	}

	return pd.getStatus(), nil
}

func (pd *playbackDevice) Add(ctx context.Context, ids []string) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Add action", "ids", ids, "device", pd)
	if len(ids) < 1 {
		return pd.getStatus(), nil
	}

	items := model.MediaFiles{}

	for _, id := range ids {
		mf, err := pd.ParentPlaybackServer.GetMediaFile(id)
		if err != nil {
			return DeviceStatus{}, err
		}
		log.Debug(ctx, "Found mediafile: "+mf.Path)
		items = append(items, *mf)
	}
	pd.mu.Lock()
	pd.PlaybackQueue.Add(items)
	pd.mu.Unlock()

	return pd.getStatus(), nil
}

func (pd *playbackDevice) Clear(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Clear action", "device", pd)
	pd.mu.Lock()
	activeTrack := pd.ActiveTrack
	pd.ActiveTrack = nil
	pd.PlaybackQueue.Clear()
	pd.mu.Unlock()
	if activeTrack != nil {
		activeTrack.Pause()
		activeTrack.Close()
	}
	return pd.getStatus(), nil
}

func (pd *playbackDevice) Remove(ctx context.Context, index int) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Remove action", "index", index, "device", pd)
	pd.mu.RLock()
	activeTrack := pd.ActiveTrack
	queueIndex := pd.PlaybackQueue.Index
	pd.mu.RUnlock()
	// pausing if attempting to remove running track
	if activeTrack != nil && activeTrack.IsPlaying() && queueIndex == index {
		_, err := pd.Stop(ctx)
		if err != nil {
			log.Error(ctx, "error stopping running track")
			return pd.getStatus(), err
		}
	}

	pd.mu.Lock()
	if index > -1 && index < pd.PlaybackQueue.Size() {
		pd.PlaybackQueue.Remove(index)
	} else {
		pd.mu.Unlock()
		log.Error(ctx, "Index to remove out of range: "+fmt.Sprint(index))
		return pd.getStatus(), nil
	}
	pd.mu.Unlock()
	return pd.getStatus(), nil
}

func (pd *playbackDevice) Shuffle(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Shuffle action", "device", pd)
	pd.mu.Lock()
	if pd.PlaybackQueue.Size() > 1 {
		pd.PlaybackQueue.Shuffle()
	}
	pd.mu.Unlock()
	return pd.getStatus(), nil
}

// SetGain is used to control the playback volume. A float value between 0.0 and 1.0.
func (pd *playbackDevice) SetGain(ctx context.Context, gain float32) (DeviceStatus, error) {
	log.Debug(ctx, "Processing SetGain action", "newGain", gain, "device", pd)

	pd.mu.RLock()
	activeTrack := pd.ActiveTrack
	pd.mu.RUnlock()
	if activeTrack != nil {
		activeTrack.SetVolume(gain)
	}
	pd.mu.Lock()
	pd.Gain = gain
	pd.mu.Unlock()

	return pd.getStatus(), nil
}

func (pd *playbackDevice) trackSwitcherGoroutine() {
	log.Debug("Started trackSwitcher goroutine", "device", pd)
	for {
		select {
		case <-pd.PlaybackDone:
			log.Debug("Track switching detected")
			pd.mu.Lock()
			activeTrack := pd.ActiveTrack
			pd.ActiveTrack = nil
			atLast := pd.PlaybackQueue.IsAtLastElement()
			nextIndex := pd.PlaybackQueue.Index
			if !atLast {
				pd.PlaybackQueue.IncreaseIndex()
				nextIndex = pd.PlaybackQueue.Index
				log.Debug("Switching to next song", "queue", pd.PlaybackQueue.String())
			}
			pd.mu.Unlock()

			if activeTrack != nil {
				activeTrack.Close()
			}

			if atLast {
				log.Debug("There is no song left in the playlist. Finish.")
				continue
			}

			if err := pd.switchActiveTrackByIndex(nextIndex); err != nil {
				log.Error("Error switching track", err)
				continue
			}
			pd.mu.RLock()
			activeTrack = pd.ActiveTrack
			pd.mu.RUnlock()
			if activeTrack != nil {
				activeTrack.Unpause()
			}
		case <-pd.serviceCtx.Done():
			log.Debug("Stopping trackSwitcher goroutine", "device", pd.Name)
			return
		}
	}
}

func (pd *playbackDevice) switchActiveTrackByIndex(index int) error {
	pd.mu.Lock()
	pd.PlaybackQueue.SetIndex(index)
	currentTrack := pd.PlaybackQueue.Current()
	gain := pd.Gain
	deviceName := pd.DeviceName
	serviceCtx := pd.serviceCtx
	playbackDone := pd.PlaybackDone
	trackFactory := pd.trackFactory
	pd.mu.Unlock()
	if currentTrack == nil {
		return errors.New("could not get current track")
	}

	track, err := trackFactory(serviceCtx, playbackDone, deviceName, *currentTrack)
	if err != nil {
		return err
	}
	pd.registerReceiverVolumeHandler(track)
	pd.mu.Lock()
	pd.ActiveTrack = track
	pd.mu.Unlock()
	track.SetVolume(gain)
	return nil
}

func (pd *playbackDevice) registerReceiverVolumeHandler(track Track) {
	volumeTrack, ok := track.(receiverVolumeCallbackTrack)
	if !ok {
		return
	}
	volumeTrack.SetReceiverVolumeHandler(func(level float32) {
		pd.applyReceiverVolume(track, level)
	})
}

func (pd *playbackDevice) applyReceiverVolume(origin Track, level float32) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	if pd.ActiveTrack != origin {
		return
	}
	pd.Gain = level
}
