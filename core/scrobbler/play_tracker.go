package scrobbler

import (
	"context"
	"maps"
	"sort"
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

type PlayTracker interface {
	NowPlaying(ctx context.Context, playerId string, playerName string, trackId string, position int) error
	GetNowPlaying(ctx context.Context) ([]NowPlayingInfo, error)
	Submit(ctx context.Context, submissions []Submission) error
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
	p := &playTracker{
		ds:                ds,
		playMap:           m,
		broker:            broker,
		builtinScrobblers: make(map[string]Scrobbler),
		pluginScrobblers:  make(map[string]Scrobbler),
		pluginLoader:      pluginManager,
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
	return p
}

// pluginNamesMatchScrobblers returns true if the set of pluginNames matches the keys in pluginScrobblers
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

// refreshPluginScrobblers updates the pluginScrobblers map to match the current set of plugin scrobblers
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

	// Process additions - add new plugins
	for _, name := range pluginNames {
		current[name] = struct{}{}
		// Only create a new scrobbler if it doesn't exist
		if _, exists := p.pluginScrobblers[name]; !exists {
			s, ok := p.pluginLoader.LoadScrobbler(name)
			if ok && s != nil {
				p.pluginScrobblers[name] = newBufferedScrobbler(p.ds, s, name)
			}
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

func (p *playTracker) NowPlaying(ctx context.Context, playerId string, playerName string, trackId string, position int) error {
	mf, err := p.ds.MediaFile(ctx).GetWithParticipants(trackId)
	if err != nil {
		log.Error(ctx, "Error retrieving mediaFile", "id", trackId, err)
		return err
	}

	user, _ := request.UserFrom(ctx)
	info := NowPlayingInfo{
		MediaFile:  *mf,
		Start:      time.Now(),
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
	player, _ := request.PlayerFrom(ctx)
	if player.ScrobbleEnabled {
		p.dispatchNowPlaying(ctx, user.ID, mf, position)
	}
	return nil
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
		}
		return err
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
