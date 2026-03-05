package plugins

import (
	"context"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

const (
	CapabilityPlaylistProvider Capability = "PlaylistProvider"

	FuncPlaylistProviderGetAvailablePlaylists = "nd_playlist_provider_get_available_playlists"
	FuncPlaylistProviderGetPlaylist           = "nd_playlist_provider_get_playlist"

	// workChCapacity is the buffer size for the work channel.
	workChCapacity = 64

	// discoveryRetryDelay is how long to wait before retrying a failed GetAvailablePlaylists call.
	discoveryRetryDelay = 5 * time.Minute
)

func init() {
	registerCapability(
		CapabilityPlaylistProvider,
		FuncPlaylistProviderGetAvailablePlaylists,
		FuncPlaylistProviderGetPlaylist,
	)
}

type workType int

const (
	workDiscover workType = iota // run discoverAndSync
	workSync                     // run syncPlaylist for a single playlist
)

type workItem struct {
	typ     workType
	info    capabilities.PlaylistInfo // only for workSync
	dbID    string                    // only for workSync
	ownerID string                    // only for workSync
}

// playlistSyncer manages playlist synchronization for a single plugin.
// All mutable state (refreshTimers, discoveryTimer) is owned exclusively by the
// worker goroutine — no synchronization needed. The retryInterval and
// refreshTimerCount fields use atomics so tests can observe them race-free.
type playlistSyncer struct {
	pluginName        string
	plugin            *plugin
	ds                model.DataStore
	matcher           *matcher.Matcher
	ctx               context.Context
	cancel            context.CancelFunc
	workCh            chan workItem          // serialized work queue
	refreshTimers     map[string]*time.Timer // keyed by playlist DB ID — worker-only
	discoveryTimer    *time.Timer            // worker-only
	retryInterval     atomic.Int64           // nanoseconds; from last GetAvailablePlaylists response
	refreshTimerCount atomic.Int32           // number of active refresh timers
	done              chan struct{}          // closed when worker exits
}

func newPlaylistSyncer(parentCtx context.Context, pluginName string, p *plugin, ds model.DataStore, m *matcher.Matcher) *playlistSyncer {
	ctx, cancel := context.WithCancel(parentCtx)
	return &playlistSyncer{
		pluginName:    pluginName,
		plugin:        p,
		ds:            ds,
		matcher:       m,
		ctx:           ctx,
		cancel:        cancel,
		workCh:        make(chan workItem, workChCapacity),
		refreshTimers: make(map[string]*time.Timer),
		done:          make(chan struct{}),
	}
}

// run is the single worker goroutine that processes all work items sequentially.
// It performs an initial discovery before entering the main loop.
func (p *playlistSyncer) run() {
	defer close(p.done)

	// Run initial discovery before entering the loop
	p.discoverAndSync()

	for {
		select {
		case <-p.ctx.Done():
			p.stopAllTimers()
			return
		case item := <-p.workCh:
			switch item.typ {
			case workDiscover:
				p.discoverAndSync()
			case workSync:
				p.syncPlaylist(item.info, item.dbID, item.ownerID)
			}
		}
	}
}

// discoverAndSync calls GetAvailablePlaylists, then GetPlaylist for each, matches tracks, and upserts.
func (p *playlistSyncer) discoverAndSync() {
	ctx := p.ctx
	resp, err := callPluginFunction[capabilities.GetAvailablePlaylistsRequest, capabilities.GetAvailablePlaylistsResponse](
		ctx, p.plugin, FuncPlaylistProviderGetAvailablePlaylists, capabilities.GetAvailablePlaylistsRequest{},
	)
	if err != nil {
		log.Error(ctx, "Failed to call GetAvailablePlaylists, retrying later", "plugin", p.pluginName, err)
		p.scheduleDiscovery(discoveryRetryDelay)
		return
	}

	// Store retry interval from response
	if resp.RetryInterval > 0 {
		p.retryInterval.Store(int64(time.Duration(resp.RetryInterval) * time.Second))
	}

	resolvedUsers := map[string]string{} // username -> userID cache
	for _, info := range resp.Playlists {
		// Resolve username to user ID (cached)
		ownerID, ok := resolvedUsers[info.OwnerUsername]
		if !ok {
			user, err := p.ds.User(adminContext(ctx)).FindByUsername(info.OwnerUsername)
			if err != nil {
				log.Error(ctx, "Failed to resolve playlist owner", "plugin", p.pluginName,
					"playlistID", info.ID, "username", info.OwnerUsername, err)
				continue
			}
			ownerID = user.ID
			resolvedUsers[info.OwnerUsername] = ownerID
		}

		// Validate that the plugin is permitted to create playlists for this user
		if !p.plugin.allUsers && !slices.Contains(p.plugin.allowedUserIDs, ownerID) {
			log.Error(ctx, "Plugin not permitted to create playlists for user", "plugin", p.pluginName,
				"playlistID", info.ID, "username", info.OwnerUsername)
			continue
		}

		dbID := id.NewHash(p.pluginName, info.ID, ownerID)
		p.syncPlaylist(info, dbID, ownerID)
	}

	// Schedule re-discovery if RefreshInterval > 0
	if resp.RefreshInterval > 0 {
		p.scheduleDiscovery(time.Duration(resp.RefreshInterval) * time.Second)
	}
}

// syncPlaylist calls GetPlaylist, matches tracks, and upserts the playlist in the DB.
func (p *playlistSyncer) syncPlaylist(info capabilities.PlaylistInfo, dbID string, ownerID string) {
	ctx := p.ctx
	resp, err := callPluginFunction[capabilities.GetPlaylistRequest, capabilities.GetPlaylistResponse](
		ctx, p.plugin, FuncPlaylistProviderGetPlaylist, capabilities.GetPlaylistRequest{ID: info.ID},
	)
	if err != nil {
		if isPlaylistNotFoundError(err) {
			log.Info(ctx, "Playlist not found, skipping", "plugin", p.pluginName, "playlistID", info.ID)
			// Stop any existing refresh timer for this playlist
			if timer, ok := p.refreshTimers[dbID]; ok {
				timer.Stop()
				delete(p.refreshTimers, dbID)
				p.refreshTimerCount.Store(int32(len(p.refreshTimers)))
			}
			return
		}
		log.Warn(ctx, "Failed to call GetPlaylist", "plugin", p.pluginName, "playlistID", info.ID, err)
		// Schedule retry for transient errors if retryInterval is configured
		if ri := time.Duration(p.retryInterval.Load()); ri > 0 {
			p.schedulePlaylistRefresh(info, dbID, ownerID, ri)
		}
		return
	}

	// Convert SongRef → agents.Song and match against library
	songs := songRefsToAgentSongs(resp.Tracks)
	matched, err := p.matcher.MatchSongsToLibrary(ctx, songs, len(songs))
	if err != nil {
		log.Error(ctx, "Failed to match songs to library", "plugin", p.pluginName, "playlistID", info.ID, err)
		return
	}

	// Build playlist model
	pls := &model.Playlist{
		ID:               dbID,
		Name:             resp.Name,
		Comment:          resp.Description,
		OwnerID:          ownerID,
		Public:           false,
		ExternalImageURL: resp.CoverArtURL,
		PluginID:         p.pluginName,
		PluginPlaylistID: info.ID,
	}

	// Set tracks from matched media files
	pls.AddMediaFiles(matched)

	// Upsert via repository
	plsRepo := p.ds.Playlist(ctx)
	if err := plsRepo.Put(pls); err != nil {
		log.Error(ctx, "Failed to upsert plugin playlist", "plugin", p.pluginName, "playlistID", info.ID, err)
		return
	}

	log.Info(ctx, "Synced plugin playlist", "plugin", p.pluginName, "playlistID", info.ID,
		"name", resp.Name, "tracks", len(matched), "owner", ownerID)

	// Schedule refresh if ValidUntil > 0
	if resp.ValidUntil > 0 {
		validUntil := time.Unix(resp.ValidUntil, 0)
		delay := time.Until(validUntil)
		if delay <= 0 {
			delay = 1 * time.Second // Already expired, refresh soon
		}
		p.schedulePlaylistRefresh(info, dbID, ownerID, delay)
	}
}

func (p *playlistSyncer) schedulePlaylistRefresh(info capabilities.PlaylistInfo, dbID string, ownerID string, delay time.Duration) {
	// Cancel existing timer if any
	if timer, ok := p.refreshTimers[dbID]; ok {
		timer.Stop()
	}
	p.refreshTimers[dbID] = time.AfterFunc(delay, func() {
		select {
		case p.workCh <- workItem{typ: workSync, info: info, dbID: dbID, ownerID: ownerID}:
		case <-p.ctx.Done():
		}
	})
	p.refreshTimerCount.Store(int32(len(p.refreshTimers)))
}

func (p *playlistSyncer) scheduleDiscovery(delay time.Duration) {
	if p.discoveryTimer != nil {
		p.discoveryTimer.Stop()
	}
	p.discoveryTimer = time.AfterFunc(delay, func() {
		select {
		case p.workCh <- workItem{typ: workDiscover}:
		case <-p.ctx.Done():
		}
	})
}

// isPlaylistNotFoundError checks if the error contains a NotFound sentinel from the plugin.
func isPlaylistNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), capabilities.PlaylistProviderErrorNotFound.Error())
}

// stopAllTimers stops the discovery timer and all refresh timers.
func (p *playlistSyncer) stopAllTimers() {
	if p.discoveryTimer != nil {
		p.discoveryTimer.Stop()
	}
	for _, timer := range p.refreshTimers {
		timer.Stop()
	}
}

// Close cancels the context and waits for the worker goroutine to finish.
func (p *playlistSyncer) Close() error {
	p.cancel()
	<-p.done
	return nil
}
