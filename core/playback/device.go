package playback

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/navidrome/navidrome/core/playback/mpv"
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
	mutex                sync.Mutex
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
	startTrackSwitcher   sync.Once
}

type DeviceStatus struct {
	CurrentIndex int
	Playing      bool
	Gain         float32
	Position     int
}

const DefaultGain float32 = 1.0

// getStatusLocked must be called with pd.mutex held.
func (pd *playbackDevice) getStatusLocked() DeviceStatus {
	pos := 0
	if pd.ActiveTrack != nil {
		pos = pd.ActiveTrack.Position()
	}
	return DeviceStatus{
		CurrentIndex: pd.PlaybackQueue.Index,
		Playing:      pd.isPlayingLocked(),
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
	return fmt.Sprintf("Name: %s, Gain: %.4f, Loaded track: %s", pd.Name, pd.Gain, pd.ActiveTrack)
}

func (pd *playbackDevice) Get(ctx context.Context) (model.MediaFiles, DeviceStatus, error) {
	log.Debug(ctx, "Processing Get action", "device", pd)
	pd.mutex.Lock()
	defer pd.mutex.Unlock()
	return pd.PlaybackQueue.Get(), pd.getStatusLocked(), nil
}

func (pd *playbackDevice) Status(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, fmt.Sprintf("processing Status action on: %s, queue: %s", pd, pd.PlaybackQueue))
	pd.mutex.Lock()
	defer pd.mutex.Unlock()
	return pd.getStatusLocked(), nil
}

// Set is similar to a clear followed by a add, but will not change the currently playing track.
func (pd *playbackDevice) Set(ctx context.Context, ids []string) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Set action", "ids", ids, "device", pd)

	pd.mutex.Lock()
	defer pd.mutex.Unlock()

	pd.clearLocked()
	return pd.addLocked(ctx, ids)
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

	pd.mutex.Lock()
	defer pd.mutex.Unlock()
	return pd.startLocked(ctx)
}

func (pd *playbackDevice) startLocked(ctx context.Context) (DeviceStatus, error) {
	if pd.ActiveTrack != nil {
		if pd.isPlayingLocked() {
			log.Debug("trying to start an already playing track")
		} else {
			pd.ActiveTrack.Unpause()
		}
	} else {
		if !pd.PlaybackQueue.IsEmpty() {
			err := pd.switchActiveTrackByIndexLocked(pd.PlaybackQueue.Index)
			if err != nil {
				return pd.getStatusLocked(), err
			}
			pd.ActiveTrack.Unpause()
		}
	}

	return pd.getStatusLocked(), nil
}

func (pd *playbackDevice) Stop(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Stop action", "device", pd)
	pd.mutex.Lock()
	defer pd.mutex.Unlock()
	return pd.stopLocked(ctx)
}

func (pd *playbackDevice) stopLocked(ctx context.Context) (DeviceStatus, error) {
	if pd.ActiveTrack != nil {
		pd.ActiveTrack.Pause()
	}
	return pd.getStatusLocked(), nil
}

func (pd *playbackDevice) Skip(ctx context.Context, index int, offset int) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Skip action", "index", index, "offset", offset, "device", pd)

	pd.mutex.Lock()
	defer pd.mutex.Unlock()

	wasPlaying := pd.isPlayingLocked()

	if pd.ActiveTrack != nil && wasPlaying {
		pd.ActiveTrack.Pause()
	}

	if index != pd.PlaybackQueue.Index && pd.ActiveTrack != nil {
		pd.ActiveTrack.Close()
		pd.ActiveTrack = nil
	}

	if pd.ActiveTrack == nil {
		err := pd.switchActiveTrackByIndexLocked(index)
		if err != nil {
			return pd.getStatusLocked(), err
		}
	}

	err := pd.ActiveTrack.SetPosition(offset)
	if err != nil {
		log.Error(ctx, "error setting position", err)
		return pd.getStatusLocked(), err
	}

	if wasPlaying {
		_, err = pd.startLocked(ctx)
		if err != nil {
			log.Error(ctx, "error starting new track after skipping")
			return pd.getStatusLocked(), err
		}
	}

	return pd.getStatusLocked(), nil
}

func (pd *playbackDevice) Add(ctx context.Context, ids []string) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Add action", "ids", ids, "device", pd)
	pd.mutex.Lock()
	defer pd.mutex.Unlock()
	return pd.addLocked(ctx, ids)
}

func (pd *playbackDevice) addLocked(ctx context.Context, ids []string) (DeviceStatus, error) {
	if len(ids) < 1 {
		return pd.getStatusLocked(), nil
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
	pd.PlaybackQueue.Add(items)

	return pd.getStatusLocked(), nil
}

func (pd *playbackDevice) Clear(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Clear action", "device", pd)
	pd.mutex.Lock()
	defer pd.mutex.Unlock()
	pd.clearLocked()
	return pd.getStatusLocked(), nil
}

func (pd *playbackDevice) clearLocked() {
	if pd.ActiveTrack != nil {
		pd.ActiveTrack.Pause()
		pd.ActiveTrack.Close()
		pd.ActiveTrack = nil
	}
	pd.PlaybackQueue.Clear()
}

func (pd *playbackDevice) Remove(ctx context.Context, index int) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Remove action", "index", index, "device", pd)
	pd.mutex.Lock()
	defer pd.mutex.Unlock()

	// pausing if attempting to remove running track
	if pd.isPlayingLocked() && pd.PlaybackQueue.Index == index {
		_, err := pd.stopLocked(ctx)
		if err != nil {
			log.Error(ctx, "error stopping running track")
			return pd.getStatusLocked(), err
		}
	}

	if index > -1 && index < pd.PlaybackQueue.Size() {
		pd.PlaybackQueue.Remove(index)
	} else {
		log.Error(ctx, "Index to remove out of range: "+fmt.Sprint(index))
	}
	return pd.getStatusLocked(), nil
}

func (pd *playbackDevice) Shuffle(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Shuffle action", "device", pd)
	pd.mutex.Lock()
	defer pd.mutex.Unlock()
	if pd.PlaybackQueue.Size() > 1 {
		pd.PlaybackQueue.Shuffle()
	}
	return pd.getStatusLocked(), nil
}

// SetGain is used to control the playback volume. A float value between 0.0 and 1.0.
func (pd *playbackDevice) SetGain(ctx context.Context, gain float32) (DeviceStatus, error) {
	log.Debug(ctx, "Processing SetGain action", "newGain", gain, "device", pd)

	pd.mutex.Lock()
	defer pd.mutex.Unlock()

	if pd.ActiveTrack != nil {
		pd.ActiveTrack.SetVolume(gain)
	}
	pd.Gain = gain

	return pd.getStatusLocked(), nil
}

// isPlayingLocked must be called with pd.mutex held.
func (pd *playbackDevice) isPlayingLocked() bool {
	return pd.ActiveTrack != nil && pd.ActiveTrack.IsPlaying()
}

func (pd *playbackDevice) trackSwitcherGoroutine() {
	log.Debug("Started trackSwitcher goroutine", "device", pd)
	for {
		select {
		case <-pd.PlaybackDone:
			log.Debug("Track switching detected")
			pd.mutex.Lock()
			if pd.ActiveTrack != nil {
				pd.ActiveTrack.Close()
				pd.ActiveTrack = nil
			}

			if !pd.PlaybackQueue.IsAtLastElement() {
				pd.PlaybackQueue.IncreaseIndex()
				log.Debug("Switching to next song", "queue", pd.PlaybackQueue.String())
				err := pd.switchActiveTrackByIndexLocked(pd.PlaybackQueue.Index)
				if err != nil {
					log.Error("Error switching track", err)
				}
				if pd.ActiveTrack != nil {
					pd.ActiveTrack.Unpause()
				}
			} else {
				log.Debug("There is no song left in the playlist. Finish.")
			}
			pd.mutex.Unlock()
		case <-pd.serviceCtx.Done():
			log.Debug("Stopping trackSwitcher goroutine", "device", pd.Name)
			return
		}
	}
}

// switchActiveTrackByIndexLocked must be called with pd.mutex held.
func (pd *playbackDevice) switchActiveTrackByIndexLocked(index int) error {
	pd.PlaybackQueue.SetIndex(index)
	currentTrack := pd.PlaybackQueue.Current()
	if currentTrack == nil {
		return errors.New("could not get current track")
	}

	track, err := mpv.NewTrack(pd.serviceCtx, pd.PlaybackDone, pd.DeviceName, *currentTrack)
	if err != nil {
		return err
	}
	pd.ActiveTrack = track
	pd.ActiveTrack.SetVolume(pd.Gain)
	return nil
}
