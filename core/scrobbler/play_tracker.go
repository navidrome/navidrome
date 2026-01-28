package scrobbler

import (
	"context"
	"maps"
	"sort"
	"sync"
	"sync/atomic"
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

type NowPlayingInfo struct {
	MediaFile  model.MediaFile
	Start      time.Time
	Position   int
	Username   string
	PlayerId   string
	PlayerName string
}

type Submission struct {
	TrackID   string
	Timestamp time.Time
}

type nowPlayingEntry struct {
	ctx      context.Context
	userId   string
	track    *model.MediaFile
	position int
}

// playSession tracks an active play session for duration calculation.
// Keyed by userID:playerID to track what each player is currently playing.
type playSession struct {
	TrackID    string    // The track being played
	ScrobbleID string    // The scrobble ID (set when Submit is called), empty if not yet scrobbled
	Start      time.Time // When playback started (wall clock time)
	Position   int       // Position in seconds when playback started
	UserID     string    // User ID for DB operations
}

type PlayTracker interface {
	NowPlaying(ctx context.Context, playerId string, playerName string, trackId string, position int) error
	GetNowPlaying(ctx context.Context) ([]NowPlayingInfo, error)
	Submit(ctx context.Context, submissions []Submission) error
	// StopPlayback finalizes the duration for an active session when playback stops.
	StopPlayback(ctx context.Context, trackId string, positionInSeconds int)
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
	playSessionMap    map[string]*playSession // key: userID:playerID
	playSessionMu     sync.Mutex
	enableNowPlaying  bool // Captured at creation time to avoid races with config changes
	builtinScrobblers map[string]Scrobbler
	pluginScrobblers  map[string]Scrobbler
	pluginLoader      PluginLoader
	mu                sync.RWMutex
	npQueue           map[string]nowPlayingEntry
	npMu              sync.Mutex
	npSignal          chan struct{}
	shutdown          chan struct{}
	workerDone        chan struct{}
	stopped           atomic.Bool // Set to true when stopNowPlayingWorker is called
}

func GetPlayTracker(ds model.DataStore, broker events.Broker, pluginManager PluginLoader) PlayTracker {
	return singleton.GetInstance(func() *playTracker {
		return newPlayTracker(ds, broker, pluginManager)
	})
}

// This constructor only exists for testing. For normal usage, the PlayTracker has to be a singleton, returned by
// the GetPlayTracker function above
func newPlayTracker(ds model.DataStore, broker events.Broker, pluginManager PluginLoader) *playTracker {
	m := cache.NewSimpleCache[string, NowPlayingInfo]()
	// Capture config value at creation time to avoid races with config changes in tests
	enableNowPlaying := conf.Server.EnableNowPlaying
	p := &playTracker{
		ds:                ds,
		playMap:           m,
		playSessionMap:    make(map[string]*playSession),
		enableNowPlaying:  enableNowPlaying,
		broker:            broker,
		builtinScrobblers: make(map[string]Scrobbler),
		pluginScrobblers:  make(map[string]Scrobbler),
		pluginLoader:      pluginManager,
		npQueue:           make(map[string]nowPlayingEntry),
		npSignal:          make(chan struct{}, 1),
		shutdown:          make(chan struct{}),
		workerDone:        make(chan struct{}),
	}

	// Set up expiration callback for NowPlaying entries
	// When a NowPlaying entry expires (track finished), finalize the session duration
	m.OnExpiration(func(playerId string, info NowPlayingInfo) {
		// Skip if the tracker has been stopped (prevents races during test cleanup)
		if p.stopped.Load() {
			return
		}
		if p.enableNowPlaying {
			broker.SendBroadcastMessage(context.Background(), &events.NowPlayingCount{Count: m.Len()})
		}
		// Finalize the session when NowPlaying expires (track finished naturally)
		p.finalizeSessionOnExpiration(playerId, info)
	})

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
	p.stopped.Store(true) // Prevent expiration callbacks from running
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

// sessionKey generates a unique key for tracking play sessions per user/player combination.
func sessionKey(userID, playerID string) string {
	return userID + ":" + playerID
}

// finalizeSession calculates and updates the duration for a session's scrobble.
// Called when a track changes or when NowPlaying expires.
// NOTE: This should be called WITHOUT holding the playSessionMu lock since it makes DB calls.
func (p *playTracker) finalizeSession(session *playSession) {
	if session == nil || session.ScrobbleID == "" {
		// No scrobble recorded yet, nothing to update
		return
	}

	duration := session.Position

	// Update the scrobble in the database
	ctx := context.Background()
	err := p.ds.Scrobble(ctx).UpdateDuration(session.ScrobbleID, duration)
	if err != nil {
		log.Error(ctx, "Error updating scrobble duration", "scrobbleID", session.ScrobbleID, "duration", duration, err)
	} else {
		log.Debug(ctx, "Updated scrobble duration", "scrobbleID", session.ScrobbleID, "duration", duration, "trackID", session.TrackID)
	}
}

// finalizeSessionOnExpiration is called when a NowPlaying entry expires.
// At TTL expiration, we can't know if the user kept listening or stopped earlier.
// We use the last known Position as the duration since it's the most reliable data.
func (p *playTracker) finalizeSessionOnExpiration(playerID string, info NowPlayingInfo) {
	// Skip if the tracker has been stopped
	if p.stopped.Load() {
		return
	}

	p.playSessionMu.Lock()
	var sessionToFinalize *playSession
	// Find session by iterating (we need to match by playerID which is part of the key)
	for key, session := range p.playSessionMap {
		// Check if this session matches the expired NowPlaying entry
		if session.TrackID == info.MediaFile.ID && key == sessionKey(session.UserID, playerID) {
			sessionToFinalize = session
			delete(p.playSessionMap, key)
			break
		}
	}
	p.playSessionMu.Unlock()

	// Finalize outside the lock. This will calculate the duration up to the point of expiration.
	if sessionToFinalize != nil {
		p.finalizeSession(sessionToFinalize)
	}
}

// getOrCreateSession gets the current session for a user/player, finalizing any previous one if track changed.
// Returns the session for the current track.
// NOTE: For the same track, this updates Start and Position on every call to improve duration accuracy.
func (p *playTracker) getOrCreateSession(userID, playerID, trackID string, start time.Time, position int) *playSession {
	var sessionToFinalize *playSession

	p.playSessionMu.Lock()
	key := sessionKey(userID, playerID)
	existing := p.playSessionMap[key]

	// If there's an existing session for a different track, mark it for finalization
	if existing != nil && existing.TrackID != trackID {
		sessionToFinalize = existing
		existing = nil
	}

	// If no session or session was for different track, create new one
	if existing == nil {
		existing = &playSession{
			ScrobbleID: "", // Will be set when Submit is called
			TrackID:    trackID,
			Start:      start,
			Position:   position,
			UserID:     userID,
		}
		p.playSessionMap[key] = existing
	} else {
		existing.Start = start
		existing.Position = position
	}
	result := existing
	p.playSessionMu.Unlock()

	// Finalize outside the lock to avoid holding it during DB operations
	if sessionToFinalize != nil {
		p.finalizeSession(sessionToFinalize)
	}

	return result
}

// setSessionScrobbleID sets the scrobble ID for an existing session.
// Called when Submit creates a scrobble record.
func (p *playTracker) setSessionScrobbleID(userID, playerID, trackID, scrobbleID string) {
	p.playSessionMu.Lock()
	defer p.playSessionMu.Unlock()

	key := sessionKey(userID, playerID)
	session := p.playSessionMap[key]
	if session != nil && session.TrackID == trackID {
		session.ScrobbleID = scrobbleID
	}
}

// getSessionDuration returns the current duration (in seconds) for an active session.
// Returns 0 if no session exists or if trackID doesn't match.
func (p *playTracker) getSessionDuration(userID, playerID, trackID string) int {
	p.playSessionMu.Lock()
	defer p.playSessionMu.Unlock()

	key := sessionKey(userID, playerID)
	session := p.playSessionMap[key]
	if session == nil || session.TrackID != trackID {
		return 0
	}

	return session.Position
}

func (p *playTracker) NowPlaying(ctx context.Context, playerId string, playerName string, trackId string, position int) error {
	mf, err := p.ds.MediaFile(ctx).GetWithParticipants(trackId)
	if err != nil {
		log.Error(ctx, "Error retrieving mediaFile", "id", trackId, err)
		return err
	}

	user, _ := request.UserFrom(ctx)
	now := time.Now()
	info := NowPlayingInfo{
		MediaFile:  *mf,
		Start:      now,
		Position:   position,
		Username:   user.UserName,
		PlayerId:   playerId,
		PlayerName: playerName,
	}

	// Calculate TTL based on remaining track duration. If position exceeds track duration,
	// remaining is set to 0 to avoid negative TTL.
	remaining := int(mf.Duration) - position
	if remaining < 0 {
		remaining = 0
	}
	// Add 5 seconds buffer to ensure the NowPlaying info is available slightly longer than the track duration.
	ttl := time.Duration(remaining+5) * time.Second
	_ = p.playMap.AddWithTTL(playerId, info, ttl)
	if conf.Server.EnableNowPlaying {
		p.broker.SendBroadcastMessage(ctx, &events.NowPlayingCount{Count: p.playMap.Len()})
	}

	// Get or create play session for duration tracking.
	// This will finalize any previous session for a different track on this player.
	_ = p.getOrCreateSession(user.ID, playerId, trackId, now, position)

	player, _ := request.PlayerFrom(ctx)
	if player.ScrobbleEnabled {
		p.enqueueNowPlaying(ctx, playerId, user.ID, mf, position)
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
	sort.Slice(res, func(i, j int) bool {
		return res[i].Start.After(res[j].Start)
	})
	return res, nil
}

func (p *playTracker) Submit(ctx context.Context, submissions []Submission) error {
	username, _ := request.UsernameFrom(ctx)
	user, _ := request.UserFrom(ctx)
	player, _ := request.PlayerFrom(ctx)

	// Get player ID for session lookup
	playerID, ok := request.ClientUniqueIdFrom(ctx)
	if !ok {
		playerID = player.ID
	}

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

		// Get initial duration from the active session (if any).
		// This ensures we have a duration value even if the server restarts before TTL expires.
		initialDuration := p.getSessionDuration(user.ID, playerID, s.TrackID)
		var durationPtr *int
		if initialDuration > 0 {
			durationPtr = &initialDuration
		}

		// Create scrobble with initial duration from session
		scrobbleID, err := p.incPlay(ctx, mf, s.Timestamp, durationPtr)
		if err != nil {
			log.Error(ctx, "Error updating play counts", "id", mf.ID, "track", mf.Title, "user", username, err)
		} else {
			success++
			event.With("song", mf.ID).With("album", mf.AlbumID).With("artist", mf.AlbumArtistID)
			log.Info(ctx, "Scrobbled", "title", mf.Title, "artist", mf.Artist, "user", username, "timestamp", s.Timestamp, "scrobbleID", scrobbleID, "initialDuration", initialDuration)

			// Store the scrobble ID in the session so duration can be updated later when playback ends
			if scrobbleID != "" {
				p.setSessionScrobbleID(user.ID, playerID, s.TrackID, scrobbleID)
			}

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

func (p *playTracker) StopPlayback(ctx context.Context, trackId string, position int) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return
	}

	// Get player ID - same logic as Submit
	player, _ := request.PlayerFrom(ctx)
	playerID, ok := request.ClientUniqueIdFrom(ctx)
	if !ok {
		playerID = player.ID
	}

	var sessionToFinalize *playSession
	var scrobbleID string

	p.playSessionMu.Lock()
	key := sessionKey(user.ID, playerID)
	session := p.playSessionMap[key]
	if session != nil && session.TrackID == trackId && session.ScrobbleID != "" {
		scrobbleID = session.ScrobbleID
		sessionToFinalize = session
		delete(p.playSessionMap, key)
	}
	p.playSessionMu.Unlock()

	// Update duration outside the lock
	if sessionToFinalize != nil && scrobbleID != "" {
		duration := position
		if duration < 0 {
			duration = 0
		}

		err := p.ds.Scrobble(ctx).UpdateDuration(scrobbleID, duration)
		if err != nil {
			log.Error(ctx, "Error updating scrobble duration on stop", "scrobbleID", scrobbleID, "duration", duration, err)
		} else {
			log.Debug(ctx, "Updated scrobble duration on stop", "scrobbleID", scrobbleID, "duration", duration, "trackID", trackId)
		}
	}
}

func (p *playTracker) incPlay(ctx context.Context, track *model.MediaFile, timestamp time.Time, duration *int) (string, error) {
	var scrobbleID string
	err := p.ds.WithTx(func(tx model.DataStore) error {
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
			// Create scrobble with initial duration from session.
			// Duration may be updated later when playback ends (track change or TTL expiration).
			scrobbleID, err = tx.Scrobble(ctx).RecordScrobble(track.ID, timestamp, duration)
			return err
		}
		return nil
	})
	return scrobbleID, err
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
