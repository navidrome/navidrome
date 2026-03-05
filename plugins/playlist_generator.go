package plugins

import (
	"context"
	"fmt"
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
	CapabilityPlaylistGenerator Capability = "PlaylistGenerator"

	FuncPlaylistGeneratorGetAvailablePlaylists = "nd_playlist_generator_get_available_playlists"
	FuncPlaylistGeneratorGetPlaylist           = "nd_playlist_generator_get_playlist"

	// workChCapacity is the buffer size for the work channel.
	workChCapacity = 16

	// discoveryRetryDelay is how long to wait before retrying a failed GetAvailablePlaylists call.
	discoveryRetryDelay = 5 * time.Minute
)

func init() {
	registerCapability(
		CapabilityPlaylistGenerator,
		FuncPlaylistGeneratorGetAvailablePlaylists,
		FuncPlaylistGeneratorGetPlaylist,
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

// playlistGeneratorOrchestrator manages playlist generation for a single plugin.
// All mutable state (refreshTimers, discoveryTimer) is owned exclusively by the
// worker goroutine — no synchronization needed. The retryInterval and
// refreshTimerCount fields use atomics so tests can observe them race-free.
type playlistGeneratorOrchestrator struct {
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

func newPlaylistGeneratorOrchestrator(parentCtx context.Context, pluginName string, p *plugin, ds model.DataStore) *playlistGeneratorOrchestrator {
	ctx, cancel := context.WithCancel(parentCtx)
	return &playlistGeneratorOrchestrator{
		pluginName:    pluginName,
		plugin:        p,
		ds:            ds,
		matcher:       matcher.New(ds),
		ctx:           ctx,
		cancel:        cancel,
		workCh:        make(chan workItem, workChCapacity),
		refreshTimers: make(map[string]*time.Timer),
		done:          make(chan struct{}),
	}
}

// run is the single worker goroutine that processes all work items sequentially.
// It performs an initial discovery before entering the main loop.
func (o *playlistGeneratorOrchestrator) run() {
	defer close(o.done)

	// Run initial discovery before entering the loop
	o.discoverAndSync()

	for {
		select {
		case <-o.ctx.Done():
			o.stopAllTimers()
			return
		case item := <-o.workCh:
			switch item.typ {
			case workDiscover:
				o.discoverAndSync()
			case workSync:
				o.syncPlaylist(item.info, item.dbID, item.ownerID)
			}
		}
	}
}

// discoverAndSync calls GetAvailablePlaylists, then GetPlaylist for each, matches tracks, and upserts.
func (o *playlistGeneratorOrchestrator) discoverAndSync() {
	ctx := o.ctx
	resp, err := callPluginFunction[capabilities.GetAvailablePlaylistsRequest, capabilities.GetAvailablePlaylistsResponse](
		ctx, o.plugin, FuncPlaylistGeneratorGetAvailablePlaylists, capabilities.GetAvailablePlaylistsRequest{},
	)
	if err != nil {
		log.Error(ctx, "Failed to call GetAvailablePlaylists, retrying later", "plugin", o.pluginName, err)
		o.scheduleDiscovery(discoveryRetryDelay)
		return
	}

	// Store retry interval from response
	if resp.RetryInterval > 0 {
		o.retryInterval.Store(int64(time.Duration(resp.RetryInterval) * time.Second))
	}

	for _, info := range resp.Playlists {
		// Resolve username to user ID
		user, err := o.ds.User(adminContext(ctx)).FindByUsername(info.OwnerUsername)
		if err != nil {
			log.Error(ctx, "Failed to resolve playlist owner", "plugin", o.pluginName,
				"playlistID", info.ID, "username", info.OwnerUsername, err)
			continue
		}
		ownerID := user.ID
		dbID := id.NewHash(o.pluginName, info.ID, ownerID)
		o.syncPlaylist(info, dbID, ownerID)
	}

	// Schedule re-discovery if RefreshInterval > 0
	if resp.RefreshInterval > 0 {
		o.scheduleDiscovery(time.Duration(resp.RefreshInterval) * time.Second)
	}
}

// syncPlaylist calls GetPlaylist, matches tracks, and upserts the playlist in the DB.
func (o *playlistGeneratorOrchestrator) syncPlaylist(info capabilities.PlaylistInfo, dbID string, ownerID string) {
	ctx := o.ctx
	resp, err := callPluginFunction[capabilities.GetPlaylistRequest, capabilities.GetPlaylistResponse](
		ctx, o.plugin, FuncPlaylistGeneratorGetPlaylist, capabilities.GetPlaylistRequest{ID: info.ID},
	)
	if err != nil {
		if isPlaylistNotFoundError(err) {
			log.Info(ctx, "Playlist not found, skipping", "plugin", o.pluginName, "playlistID", info.ID)
			// Stop any existing refresh timer for this playlist
			if timer, ok := o.refreshTimers[dbID]; ok {
				timer.Stop()
				delete(o.refreshTimers, dbID)
				o.refreshTimerCount.Store(int32(len(o.refreshTimers)))
			}
			return
		}
		log.Warn(ctx, "Failed to call GetPlaylist", "plugin", o.pluginName, "playlistID", info.ID, err)
		// Schedule retry for transient errors if retryInterval is configured
		if ri := time.Duration(o.retryInterval.Load()); ri > 0 {
			o.schedulePlaylistRefresh(info, dbID, ownerID, ri)
		}
		return
	}

	// Convert SongRef → agents.Song and match against library
	songs := songRefsToAgentSongs(resp.Tracks)
	matched, err := o.matcher.MatchSongsToLibrary(ctx, songs, len(songs))
	if err != nil {
		log.Error(ctx, "Failed to match songs to library", "plugin", o.pluginName, "playlistID", info.ID, err)
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
		PluginID:         o.pluginName,
		PluginPlaylistID: info.ID,
	}

	// Set tracks from matched media files
	tracks := make(model.PlaylistTracks, len(matched))
	for i, mf := range matched {
		tracks[i] = model.PlaylistTrack{
			ID:          fmt.Sprintf("%d", i+1),
			MediaFileID: mf.ID,
			PlaylistID:  dbID,
			MediaFile:   mf,
		}
	}
	pls.SetTracks(tracks)

	// Upsert via repository
	plsRepo := o.ds.Playlist(ctx)
	if err := plsRepo.Put(pls); err != nil {
		log.Error(ctx, "Failed to upsert plugin playlist", "plugin", o.pluginName, "playlistID", info.ID, err)
		return
	}

	log.Info(ctx, "Synced plugin playlist", "plugin", o.pluginName, "playlistID", info.ID,
		"name", resp.Name, "tracks", len(matched), "owner", ownerID)

	// Schedule refresh if ValidUntil > 0
	if resp.ValidUntil > 0 {
		validUntil := time.Unix(resp.ValidUntil, 0)
		delay := time.Until(validUntil)
		if delay <= 0 {
			delay = 1 * time.Second // Already expired, refresh soon
		}
		o.schedulePlaylistRefresh(info, dbID, ownerID, delay)
	}
}

func (o *playlistGeneratorOrchestrator) schedulePlaylistRefresh(info capabilities.PlaylistInfo, dbID string, ownerID string, delay time.Duration) {
	// Cancel existing timer if any
	if timer, ok := o.refreshTimers[dbID]; ok {
		timer.Stop()
	}
	o.refreshTimers[dbID] = time.AfterFunc(delay, func() {
		select {
		case o.workCh <- workItem{typ: workSync, info: info, dbID: dbID, ownerID: ownerID}:
		case <-o.ctx.Done():
		}
	})
	o.refreshTimerCount.Store(int32(len(o.refreshTimers)))
}

func (o *playlistGeneratorOrchestrator) scheduleDiscovery(delay time.Duration) {
	if o.discoveryTimer != nil {
		o.discoveryTimer.Stop()
	}
	o.discoveryTimer = time.AfterFunc(delay, func() {
		select {
		case o.workCh <- workItem{typ: workDiscover}:
		case <-o.ctx.Done():
		}
	})
}

// isPlaylistNotFoundError checks if the error contains a NotFound sentinel from the plugin.
func isPlaylistNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), capabilities.PlaylistGeneratorErrorNotFound.Error())
}

// stopAllTimers stops the discovery timer and all refresh timers.
func (o *playlistGeneratorOrchestrator) stopAllTimers() {
	if o.discoveryTimer != nil {
		o.discoveryTimer.Stop()
	}
	for _, timer := range o.refreshTimers {
		timer.Stop()
	}
}

// stop cancels the context and waits for the worker goroutine to finish.
func (o *playlistGeneratorOrchestrator) stop() {
	o.cancel()
	<-o.done
}
