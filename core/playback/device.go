package playback

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/navidrome/navidrome/core/playback/mpv"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
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
	playTracker          scrobbler.PlayTracker
	scrobbleMu           sync.RWMutex
	// scrobbleCtx holds the most recently seen request context (merged onto
	// serviceCtx so it survives past the originating HTTP request), used to
	// report playback state from the async trackSwitcherGoroutine, which has
	// no request of its own. Written from request-handling goroutines, read
	// from trackSwitcherGoroutine, so access is guarded by scrobbleMu.
	scrobbleCtx context.Context
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
func NewPlaybackDevice(ctx context.Context, playbackServer PlaybackServer, name string, deviceName string, playTracker scrobbler.PlayTracker) *playbackDevice {
	return &playbackDevice{
		serviceCtx:           ctx,
		ParentPlaybackServer: playbackServer,
		User:                 "",
		Name:                 name,
		DeviceName:           deviceName,
		Gain:                 DefaultGain,
		PlaybackQueue:        NewQueue(),
		PlaybackDone:         make(chan bool),
		playTracker:          playTracker,
		scrobbleCtx:          ctx,
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
	ctx = pd.rememberScrobbleCtx(ctx)

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
			pd.reportActiveTrackStarted(ctx)
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
	ctx = pd.rememberScrobbleCtx(ctx)

	wasPlaying := pd.isPlaying()

	if pd.ActiveTrack != nil && wasPlaying {
		pd.ActiveTrack.Pause()
	}

	if index != pd.PlaybackQueue.Index && pd.ActiveTrack != nil {
		pd.reportActiveTrackStopped(ctx)
		pd.ActiveTrack.Close()
		pd.ActiveTrack = nil
	}

	if pd.ActiveTrack == nil {
		err := pd.switchActiveTrackByIndex(index)
		if err != nil {
			return pd.getStatus(), err
		}
		pd.reportActiveTrackStarted(ctx)
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
		pd.reportActiveTrackStopped(pd.rememberScrobbleCtx(ctx))
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

// rememberScrobbleCtx merges request-scoped values (user, player, client)
// from ctx onto the device's long-lived service context, and remembers the
// result. This lets the async trackSwitcherGoroutine - which has no HTTP
// request of its own - still report playback state once the originating
// request has completed.
func (pd *playbackDevice) rememberScrobbleCtx(ctx context.Context) context.Context {
	merged := request.AddValues(pd.serviceCtx, ctx)
	pd.scrobbleMu.Lock()
	pd.scrobbleCtx = merged
	pd.scrobbleMu.Unlock()
	return merged
}

// getScrobbleCtx returns the most recently remembered scrobble context,
// safe to call concurrently with rememberScrobbleCtx.
func (pd *playbackDevice) getScrobbleCtx() context.Context {
	pd.scrobbleMu.RLock()
	defer pd.scrobbleMu.RUnlock()
	return pd.scrobbleCtx
}

// reportActiveTrackStarted reports to the scrobbler that the currently active
// track started playing. Must be called after switchActiveTrackByIndex.
func (pd *playbackDevice) reportActiveTrackStarted(ctx context.Context) {
	if pd.ActiveTrack == nil {
		return
	}
	mf := pd.PlaybackQueue.Current()
	if mf == nil {
		return
	}
	pd.reportPlayback(ctx, *mf, scrobbler.StateStarting, 0)
}

// reportActiveTrackStopped reports to the scrobbler that the currently active
// track stopped at its current (live) position. Must be called before
// ActiveTrack is closed/replaced and before PlaybackQueue's index is advanced.
func (pd *playbackDevice) reportActiveTrackStopped(ctx context.Context) {
	if pd.ActiveTrack == nil {
		return
	}
	mf := pd.PlaybackQueue.Current()
	if mf == nil {
		return
	}
	pd.reportPlayback(ctx, *mf, scrobbler.StateStopped, int64(pd.ActiveTrack.Position())*1000)
}

// reportActiveTrackFinished reports a track that played through to its
// natural end (mpv reached end-of-stream). By the time this fires, mpv has
// already exited, so a live position can't be queried; the full track
// duration is reported instead. Must be called before ActiveTrack is closed
// and before PlaybackQueue's index is advanced.
func (pd *playbackDevice) reportActiveTrackFinished(ctx context.Context) {
	if pd.ActiveTrack == nil {
		return
	}
	mf := pd.PlaybackQueue.Current()
	if mf == nil {
		return
	}
	pd.reportPlayback(ctx, *mf, scrobbler.StateStopped, int64(mf.Duration*1000))
}

// reportPlayback forwards a jukebox playback state transition to the
// scrobbler, the same way client-driven playback does via the reportPlayback
// Subsonic endpoint. Jukebox playback happens entirely server-side, so
// without this no play count or scrobble is ever recorded for it (#5693).
func (pd *playbackDevice) reportPlayback(ctx context.Context, mf model.MediaFile, state string, positionMs int64) {
	if pd.playTracker == nil {
		return
	}
	if ctx == nil {
		ctx = pd.serviceCtx
	}
	player, _ := request.PlayerFrom(ctx)
	client, _ := request.ClientFrom(ctx)
	clientId, ok := request.ClientUniqueIdFrom(ctx)
	if !ok {
		clientId = player.ID
	}
	// Fall back to device-scoped identifiers so an empty clientId (e.g. no
	// jukebox request has run yet) can't collide with another player's
	// session key in the scrobbler's playMap cache.
	if clientId == "" {
		clientId = "jukebox-" + pd.Name
	}
	if client == "" {
		client = "Jukebox"
	}
	err := pd.playTracker.ReportPlayback(ctx, scrobbler.ReportPlaybackParams{
		MediaId:      mf.ID,
		PositionMs:   positionMs,
		State:        state,
		PlaybackRate: 1.0,
		ClientId:     clientId,
		ClientName:   client,
	})
	if err != nil {
		log.Warn(ctx, "Error reporting jukebox playback", "mediaId", mf.ID, "state", state, err)
	}
}

func (pd *playbackDevice) trackSwitcherGoroutine() {
	log.Debug("Started trackSwitcher goroutine", "device", pd)
	for {
		select {
		case <-pd.PlaybackDone:
			log.Debug("Track switching detected")
			if pd.ActiveTrack != nil {
				pd.reportActiveTrackFinished(pd.getScrobbleCtx())
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
					pd.reportActiveTrackStarted(pd.getScrobbleCtx())
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
