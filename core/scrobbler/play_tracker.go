package scrobbler

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/singleton"
)

const (
	StateStarting = "starting"
	StatePlaying  = "playing"
	StatePaused   = "paused"
	StateStopped  = "stopped"
)

var ValidStates = map[string]bool{
	StateStarting: true,
	StatePlaying:  true,
	StatePaused:   true,
	StateStopped:  true,
}

type NowPlayingInfo struct {
	MediaFile    model.MediaFile
	Start        time.Time
	Username     string
	PlayerId     string
	PlayerName   string
	State        string
	PositionMs   int64
	PlaybackRate float64
	LastReport   time.Time
}

type Submission struct {
	TrackID   string
	Timestamp time.Time
}

type ReportPlaybackParams struct {
	MediaId        string
	PositionMs     int64
	State          string
	PlaybackRate   float64
	IgnoreScrobble bool
	ClientId       string
	ClientName     string
}

type nowPlayingEntry struct {
	ctx      context.Context
	userId   string
	track    *model.MediaFile
	position int
}

type PlayTracker interface {
	GetNowPlaying(ctx context.Context) ([]NowPlayingInfo, error)
	Submit(ctx context.Context, submissions []Submission) error
	ReportPlayback(ctx context.Context, params ReportPlaybackParams) error
}

// PluginLoader is a minimal interface for plugin manager usage in PlayTracker
// (avoids import cycles)
type PluginLoader interface {
	PluginNames(capability string) []string
	LoadScrobbler(name string) (Scrobbler, bool)
}

type playTracker struct {
	ds                model.DataStore
	broker            events.Broker
	playMap           cache.SimpleCache[string, NowPlayingInfo]
	builtinScrobblers map[string]Scrobbler
	pluginScrobblers  map[string]Scrobbler
	pluginLoader      PluginLoader
	mu                sync.RWMutex
	npQueue           map[string]nowPlayingEntry
	npMu              sync.Mutex
	npSignal          chan struct{}
	shutdown          chan struct{}
	workerDone        chan struct{}
}

func GetPlayTracker(ds model.DataStore, broker events.Broker, pluginManager PluginLoader) PlayTracker {
	return singleton.GetInstance(func() *playTracker {
		return newPlayTracker(ds, broker, pluginManager)
	})
}

// NewPlayTracker creates a new PlayTracker instance. For normal usage, the PlayTracker has to be a singleton,
// returned by the GetPlayTracker function above. This constructor is exported for testing.
func NewPlayTracker(ds model.DataStore, broker events.Broker, pluginManager PluginLoader) PlayTracker {
	return newPlayTracker(ds, broker, pluginManager)
}

func newPlayTracker(ds model.DataStore, broker events.Broker, pluginManager PluginLoader) *playTracker {
	m := cache.NewSimpleCache[string, NowPlayingInfo]()
	p := &playTracker{
		ds:                ds,
		playMap:           m,
		broker:            broker,
		builtinScrobblers: make(map[string]Scrobbler),
		pluginScrobblers:  make(map[string]Scrobbler),
		pluginLoader:      pluginManager,
		npQueue:           make(map[string]nowPlayingEntry),
		npSignal:          make(chan struct{}, 1),
		shutdown:          make(chan struct{}),
		workerDone:        make(chan struct{}),
	}
	if conf.Server.EnableNowPlaying {
		m.OnExpiration(func(_ string, _ NowPlayingInfo) {
			broker.SendBroadcastMessage(context.Background(), &events.NowPlayingCount{Count: m.Len()})
		})
	}

	var enabled []string
	for name, constructor := range constructors {
		s := constructor(ds)
		if s == nil {
			log.Debug("Scrobbler not available. Missing configuration?", "name", name)
			continue
		}
		enabled = append(enabled, name)
		s = newBufferedScrobbler(ds, s, name)
		p.builtinScrobblers[name] = s
	}
	log.Debug("List of builtin scrobblers enabled", "names", enabled)
	go p.nowPlayingWorker()
	return p
}

// stopNowPlayingWorker stops the background worker. This is primarily for testing.
func (p *playTracker) stopNowPlayingWorker() {
	close(p.shutdown)
	<-p.workerDone // Wait for worker to finish
}

// pluginNamesMatchScrobblers returns true if the set of pluginNames matches the keys in pluginScrobblers.
func pluginNamesMatchScrobblers(pluginNames []string, scrobblers map[string]Scrobbler) bool {
	if len(pluginNames) != len(scrobblers) {
		return false
	}
	for _, name := range pluginNames {
		if _, ok := scrobblers[name]; !ok {
			return false
		}
	}
	return true
}

// refreshPluginScrobblers updates the pluginScrobblers map to match the current set of plugin scrobblers.
// The buffered scrobblers use a loader function to dynamically get the current plugin instance,
// so we only need to add/remove scrobblers when plugins are added/removed (not when reloaded).
func (p *playTracker) refreshPluginScrobblers() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pluginLoader == nil {
		return
	}

	// Get the list of available plugin names
	pluginNames := p.pluginLoader.PluginNames("Scrobbler")

	// Early return if plugin names match existing scrobblers (no change)
	if pluginNamesMatchScrobblers(pluginNames, p.pluginScrobblers) {
		return
	}

	// Build a set of current plugins for faster lookups
	current := make(map[string]struct{}, len(pluginNames))

	// Process additions - add new plugins with a loader that dynamically fetches the current instance
	for _, name := range pluginNames {
		current[name] = struct{}{}
		if _, exists := p.pluginScrobblers[name]; !exists {
			// Capture the name for the closure
			pluginName := name
			loader := p.pluginLoader
			p.pluginScrobblers[name] = newBufferedScrobblerWithLoader(p.ds, name, func() (Scrobbler, bool) {
				return loader.LoadScrobbler(pluginName)
			})
		}
	}

	type stoppableScrobbler interface {
		Scrobbler
		Stop()
	}

	// Process removals - remove plugins that no longer exist
	for name, scrobbler := range p.pluginScrobblers {
		if _, exists := current[name]; !exists {
			// If the scrobbler implements stoppableScrobbler, call Stop() before removing it
			if stoppable, ok := scrobbler.(stoppableScrobbler); ok {
				log.Debug("Stopping scrobbler", "name", name)
				stoppable.Stop()
			}
			delete(p.pluginScrobblers, name)
		}
	}
}

// getActiveScrobblers refreshes plugin scrobblers, acquires a read lock,
// combines builtin and plugin scrobblers into a new map, releases the lock,
// and returns the combined map.
func (p *playTracker) getActiveScrobblers() map[string]Scrobbler {
	p.refreshPluginScrobblers()
	p.mu.RLock()
	defer p.mu.RUnlock()
	combined := maps.Clone(p.builtinScrobblers)
	maps.Copy(combined, p.pluginScrobblers)
	return combined
}

func remainingTTL(durationSec float32, positionMs int64, rate float64) time.Duration {
	if rate <= 0 {
		rate = 1.0
	}
	remainingMs := float64(int64(durationSec*1000)-positionMs) / rate
	remainingSec := max(int(remainingMs/1000), 0)
	return time.Duration(remainingSec+5) * time.Second
}

func (p *playTracker) ReportPlayback(ctx context.Context, params ReportPlaybackParams) error {
	player, _ := request.PlayerFrom(ctx)
	user, _ := request.UserFrom(ctx)
	clientId := params.ClientId
	client := params.ClientName

	now := time.Now()

	switch params.State {
	case StateStarting:
		mf, err := p.ds.MediaFile(ctx).GetWithParticipants(params.MediaId)
		if err != nil {
			return err
		}
		info := NowPlayingInfo{
			MediaFile:    *mf,
			Start:        now,
			Username:     user.UserName,
			PlayerId:     clientId,
			PlayerName:   client,
			State:        params.State,
			PositionMs:   params.PositionMs,
			PlaybackRate: params.PlaybackRate,
			LastReport:   now,
		}
		err = p.playMap.AddWithTTL(clientId, info, remainingTTL(mf.Duration, params.PositionMs, params.PlaybackRate))
		if err != nil {
			log.Warn(ctx, "Error adding NowPlayingInfo to cache", "clientId", clientId, "mediaId", params.MediaId, "state", params.State, err)
		}

	case StatePlaying, StatePaused:
		info, getErr := p.playMap.Get(clientId)
		if getErr != nil || info.MediaFile.ID != params.MediaId {
			mf, err := p.ds.MediaFile(ctx).GetWithParticipants(params.MediaId)
			if err != nil {
				return err
			}
			info = NowPlayingInfo{
				MediaFile:  *mf,
				Start:      now.Add(-time.Duration(params.PositionMs) * time.Millisecond),
				Username:   user.UserName,
				PlayerId:   clientId,
				PlayerName: client,
			}
		}
		info.State = params.State
		info.PositionMs = params.PositionMs
		info.PlaybackRate = params.PlaybackRate
		info.LastReport = now
		ttl := 30 * time.Minute
		if params.State == StatePlaying {
			ttl = remainingTTL(info.MediaFile.Duration, params.PositionMs, params.PlaybackRate)
		}
		err := p.playMap.AddWithTTL(clientId, info, ttl)
		if err != nil {
			log.Warn(ctx, "Error updating NowPlayingInfo in cache", "clientId", clientId, "mediaId", params.MediaId, "state", params.State, err)
		}

	case StateStopped:
		if !params.IgnoreScrobble && player.ScrobbleEnabled {
			mf, err := p.ds.MediaFile(ctx).GetWithParticipants(params.MediaId)
			if err != nil {
				return err
			}
			trackDurationMs := int64(mf.Duration * 1000)
			threshold := min(trackDurationMs*50/100, 240_000)
			if params.PositionMs >= threshold {
				err = p.incPlay(ctx, mf, now)
				if err != nil {
					log.Warn(ctx, "Error updating play counts", "id", mf.ID, "track", mf.Title, "user", user.UserName, err)
				}
				p.dispatchScrobble(ctx, mf, now)
			}
		}
		p.playMap.Remove(clientId)
	}

	if conf.Server.EnableNowPlaying {
		p.broker.SendBroadcastMessage(ctx, &events.NowPlayingCount{Count: p.playMap.Len()})
	}

	if !params.IgnoreScrobble && player.ScrobbleEnabled &&
		(params.State == StateStarting || params.State == StatePlaying) {
		if info, err := p.playMap.Get(clientId); err == nil {
			p.enqueueNowPlaying(ctx, clientId, user.ID, &info.MediaFile, int(params.PositionMs/1000))
		}
	}

	return nil
}

func (p *playTracker) enqueueNowPlaying(ctx context.Context, playerId string, userId string, track *model.MediaFile, position int) {
	p.npMu.Lock()
	defer p.npMu.Unlock()
	ctx = context.WithoutCancel(ctx) // Prevent cancellation from affecting background processing
	p.npQueue[playerId] = nowPlayingEntry{
		ctx:      ctx,
		userId:   userId,
		track:    track,
		position: position,
	}
	p.sendNowPlayingSignal()
}

func (p *playTracker) sendNowPlayingSignal() {
	// Don't block if the previous signal was not read yet
	select {
	case p.npSignal <- struct{}{}:
	default:
	}
}

func (p *playTracker) nowPlayingWorker() {
	defer close(p.workerDone)
	for {
		select {
		case <-p.shutdown:
			return
		case <-time.After(time.Second):
		case <-p.npSignal:
		}

		p.npMu.Lock()
		if len(p.npQueue) == 0 {
			p.npMu.Unlock()
			continue
		}

		// Keep a copy of the entries to process and clear the queue
		entries := p.npQueue
		p.npQueue = make(map[string]nowPlayingEntry)
		p.npMu.Unlock()

		// Process entries without holding lock
		for _, entry := range entries {
			p.dispatchNowPlaying(entry.ctx, entry.userId, entry.track, entry.position)
		}
	}
}

func (p *playTracker) dispatchNowPlaying(ctx context.Context, userId string, t *model.MediaFile, position int) {
	if t.Artist == consts.UnknownArtist {
		log.Debug(ctx, "Ignoring external NowPlaying update for track with unknown artist", "track", t.Title, "artist", t.Artist)
		return
	}
	allScrobblers := p.getActiveScrobblers()
	for name, s := range allScrobblers {
		if !s.IsAuthorized(ctx, userId) {
			continue
		}
		log.Debug(ctx, "Sending NowPlaying update", "scrobbler", name, "track", t.Title, "artist", t.Artist, "position", position)
		err := s.NowPlaying(ctx, userId, t, position)
		if err != nil {
			log.Error(ctx, "Error sending NowPlayingInfo", "scrobbler", name, "track", t.Title, "artist", t.Artist, err)
			continue
		}
	}
}

func (p *playTracker) GetNowPlaying(_ context.Context) ([]NowPlayingInfo, error) {
	res := p.playMap.Values()
	slices.SortFunc(res, func(a, b NowPlayingInfo) int {
		return b.Start.Compare(a.Start)
	})
	for i := range res {
		if res[i].State == StatePlaying {
			elapsed := time.Since(res[i].LastReport).Milliseconds()
			estimated := res[i].PositionMs + int64(float64(elapsed)*res[i].PlaybackRate)
			trackDurationMs := int64(res[i].MediaFile.Duration * 1000)
			res[i].PositionMs = min(estimated, trackDurationMs)
		}
	}
	return res, nil
}

func (p *playTracker) Submit(ctx context.Context, submissions []Submission) error {
	username, _ := request.UsernameFrom(ctx)
	player, _ := request.PlayerFrom(ctx)
	if !player.ScrobbleEnabled {
		log.Debug(ctx, "External scrobbling disabled for this player", "player", player.Name, "ip", player.IP, "user", username)
	}
	event := &events.RefreshResource{}
	success := 0

	for _, s := range submissions {
		mf, err := p.ds.MediaFile(ctx).GetWithParticipants(s.TrackID)
		if err != nil {
			log.Error(ctx, "Cannot find track for scrobbling", "id", s.TrackID, "user", username, err)
			continue
		}
		err = p.incPlay(ctx, mf, s.Timestamp)
		if err != nil {
			log.Error(ctx, "Error updating play counts", "id", mf.ID, "track", mf.Title, "user", username, err)
		} else {
			success++
			event.With("song", mf.ID).With("album", mf.AlbumID).With("artist", mf.AlbumArtistID)
			log.Info(ctx, "Scrobbled", "title", mf.Title, "artist", mf.Artist, "user", username, "timestamp", s.Timestamp)
			if player.ScrobbleEnabled {
				p.dispatchScrobble(ctx, mf, s.Timestamp)
			}
		}
	}

	if success > 0 {
		p.broker.SendMessage(ctx, event)
	}
	return nil
}

func (p *playTracker) incPlay(ctx context.Context, track *model.MediaFile, timestamp time.Time) error {
	return p.ds.WithTx(func(tx model.DataStore) error {
		err := tx.MediaFile(ctx).IncPlayCount(track.ID, timestamp)
		if err != nil {
			return err
		}
		err = tx.Album(ctx).IncPlayCount(track.AlbumID, timestamp)
		if err != nil {
			return err
		}
		for _, artist := range track.Participants[model.RoleArtist] {
			err = tx.Artist(ctx).IncPlayCount(artist.ID, timestamp)
			if err != nil {
				return err
			}
		}
		if conf.Server.EnableScrobbleHistory {
			return tx.Scrobble(ctx).RecordScrobble(track.ID, timestamp)
		}
		return nil
	})
}

func (p *playTracker) dispatchScrobble(ctx context.Context, t *model.MediaFile, playTime time.Time) {
	if t.Artist == consts.UnknownArtist {
		log.Debug(ctx, "Ignoring external Scrobble for track with unknown artist", "track", t.Title, "artist", t.Artist)
		return
	}

	allScrobblers := p.getActiveScrobblers()
	u, _ := request.UserFrom(ctx)
	scrobble := Scrobble{MediaFile: *t, TimeStamp: playTime}
	for name, s := range allScrobblers {
		if !s.IsAuthorized(ctx, u.ID) {
			continue
		}
		log.Debug(ctx, "Buffering Scrobble", "scrobbler", name, "track", t.Title, "artist", t.Artist)
		err := s.Scrobble(ctx, u.ID, scrobble)
		if err != nil {
			log.Error(ctx, "Error sending Scrobble", "scrobbler", name, "track", t.Title, "artist", t.Artist, err)
			continue
		}
	}
}

var constructors map[string]Constructor

func Register(name string, init Constructor) {
	if constructors == nil {
		constructors = make(map[string]Constructor)
	}
	constructors[name] = init
}
