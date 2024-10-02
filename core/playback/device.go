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

func (pd *playbackDevice) getStatus() DeviceStatus {
	pos := 0
	if pd.ActiveTrack != nil {
		pos = pd.ActiveTrack.Position()
	}
	return DeviceStatus{
		CurrentIndex: pd.PlaybackQueue.Index,
		Playing:      pd.isPlaying(),
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
	return pd.PlaybackQueue.Get(), pd.getStatus(), nil
}

func (pd *playbackDevice) Status(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, fmt.Sprintf("processing Status action on: %s, queue: %s", pd, pd.PlaybackQueue))
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

	if pd.ActiveTrack != nil {
		if pd.isPlaying() {
			log.Debug("trying to start an already playing track")
		} else {
			pd.ActiveTrack.Unpause()
		}
	} else {
		if !pd.PlaybackQueue.IsEmpty() {
			err := pd.switchActiveTrackByIndex(pd.PlaybackQueue.Index)
			if err != nil {
				return pd.getStatus(), err
			}
			pd.ActiveTrack.Unpause()
		}
	}

	return pd.getStatus(), nil
}

func (pd *playbackDevice) Stop(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Stop action", "device", pd)
	if pd.ActiveTrack != nil {
		pd.ActiveTrack.Pause()
	}
	return pd.getStatus(), nil
}

func (pd *playbackDevice) Skip(ctx context.Context, index int, offset int) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Skip action", "index", index, "offset", offset, "device", pd)

	wasPlaying := pd.isPlaying()

	if pd.ActiveTrack != nil && wasPlaying {
		pd.ActiveTrack.Pause()
	}

	if index != pd.PlaybackQueue.Index && pd.ActiveTrack != nil {
		pd.ActiveTrack.Close()
		pd.ActiveTrack = nil
	}

	if pd.ActiveTrack == nil {
		err := pd.switchActiveTrackByIndex(index)
		if err != nil {
			return pd.getStatus(), err
		}
	}

	err := pd.ActiveTrack.SetPosition(offset)
	if err != nil {
		log.Error(ctx, "error setting position", err)
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
	pd.PlaybackQueue.Add(items)

	return pd.getStatus(), nil
}

func (pd *playbackDevice) Clear(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Clear action", "device", pd)
	if pd.ActiveTrack != nil {
		pd.ActiveTrack.Pause()
		pd.ActiveTrack.Close()
		pd.ActiveTrack = nil
	}
	pd.PlaybackQueue.Clear()
	return pd.getStatus(), nil
}

func (pd *playbackDevice) Remove(ctx context.Context, index int) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Remove action", "index", index, "device", pd)
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

func (pd *playbackDevice) Shuffle(ctx context.Context) (DeviceStatus, error) {
	log.Debug(ctx, "Processing Shuffle action", "device", pd)
	if pd.PlaybackQueue.Size() > 1 {
		pd.PlaybackQueue.Shuffle()
	}
	return pd.getStatus(), nil
}

// SetGain is used to control the playback volume. A float value between 0.0 and 1.0.
func (pd *playbackDevice) SetGain(ctx context.Context, gain float32) (DeviceStatus, error) {
	log.Debug(ctx, "Processing SetGain action", "newGain", gain, "device", pd)

	if pd.ActiveTrack != nil {
		pd.ActiveTrack.SetVolume(gain)
	}
	pd.Gain = gain

	return pd.getStatus(), nil
}

func (pd *playbackDevice) isPlaying() bool {
	return pd.ActiveTrack != nil && pd.ActiveTrack.IsPlaying()
}

func (pd *playbackDevice) trackSwitcherGoroutine() {
	log.Debug("Started trackSwitcher goroutine", "device", pd)
	for {
		select {
		case <-pd.PlaybackDone:
			log.Debug("Track switching detected")
			if pd.ActiveTrack != nil {
				pd.ActiveTrack.Close()
				pd.ActiveTrack = nil
			}

			if !pd.PlaybackQueue.IsAtLastElement() {
				pd.PlaybackQueue.IncreaseIndex()
				log.Debug("Switching to next song", "queue", pd.PlaybackQueue.String())
				err := pd.switchActiveTrackByIndex(pd.PlaybackQueue.Index)
				if err != nil {
					log.Error("Error switching track", err)
				}
				if pd.ActiveTrack != nil {
					pd.ActiveTrack.Unpause()
				}
			} else {
				log.Debug("There is no song left in the playlist. Finish.")
			}
		case <-pd.serviceCtx.Done():
			log.Debug("Stopping trackSwitcher goroutine", "device", pd.Name)
			return
		}
	}
}

func (pd *playbackDevice) switchActiveTrackByIndex(index int) error {
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
