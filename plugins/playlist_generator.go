package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

const (
	CapabilityPlaylistGenerator Capability = "PlaylistGenerator"

	FuncPlaylistGeneratorGetPlaylists = "nd_playlist_generator_get_playlists"
	FuncPlaylistGeneratorGetPlaylist  = "nd_playlist_generator_get_playlist"
)

func init() {
	registerCapability(
		CapabilityPlaylistGenerator,
		FuncPlaylistGeneratorGetPlaylists,
		FuncPlaylistGeneratorGetPlaylist,
	)
}

// playlistGeneratorOrchestrator manages playlist generation for a single plugin.
type playlistGeneratorOrchestrator struct {
	pluginName     string
	plugin         *plugin
	ds             model.DataStore
	matcher        *matcher.Matcher
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	refreshTimers  map[string]*time.Timer // keyed by playlist DB ID
	discoveryTimer *time.Timer
}

func newPlaylistGeneratorOrchestrator(pluginName string, p *plugin, ds model.DataStore, parentCtx context.Context) *playlistGeneratorOrchestrator {
	ctx, cancel := context.WithCancel(parentCtx)
	return &playlistGeneratorOrchestrator{
		pluginName:    pluginName,
		plugin:        p,
		ds:            ds,
		matcher:       matcher.New(ds),
		ctx:           ctx,
		cancel:        cancel,
		refreshTimers: make(map[string]*time.Timer),
	}
}

// discoverAndSync calls GetPlaylists, then GetPlaylist for each, matches tracks, and upserts.
func (o *playlistGeneratorOrchestrator) discoverAndSync(ctx context.Context) {
	resp, err := callPluginFunction[capabilities.GetPlaylistsRequest, capabilities.GetPlaylistsResponse](
		ctx, o.plugin, FuncPlaylistGeneratorGetPlaylists, capabilities.GetPlaylistsRequest{},
	)
	if err != nil {
		log.Error(ctx, "Failed to call GetPlaylists", "plugin", o.pluginName, err)
		return
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
		o.syncPlaylist(ctx, info, dbID, ownerID)
	}

	// Schedule re-discovery if RefreshInterval > 0
	if resp.RefreshInterval > 0 {
		o.scheduleDiscovery(ctx, time.Duration(resp.RefreshInterval)*time.Second)
	}
}

// syncPlaylist calls GetPlaylist, matches tracks, and upserts the playlist in the DB.
func (o *playlistGeneratorOrchestrator) syncPlaylist(ctx context.Context, info capabilities.PlaylistInfo, dbID string, ownerID string) {
	resp, err := callPluginFunction[capabilities.GetPlaylistRequest, capabilities.GetPlaylistResponse](
		ctx, o.plugin, FuncPlaylistGeneratorGetPlaylist, capabilities.GetPlaylistRequest{ID: info.ID},
	)
	if err != nil {
		log.Error(ctx, "Failed to call GetPlaylist", "plugin", o.pluginName, "playlistID", info.ID, err)
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
		o.schedulePlaylistRefresh(ctx, info, dbID, ownerID, delay)
	}
}

func (o *playlistGeneratorOrchestrator) schedulePlaylistRefresh(_ context.Context, info capabilities.PlaylistInfo, dbID string, ownerID string, delay time.Duration) {
	// Cancel existing timer if any
	if timer, ok := o.refreshTimers[dbID]; ok {
		timer.Stop()
	}
	o.refreshTimers[dbID] = time.AfterFunc(delay, func() {
		o.wg.Go(func() { o.syncPlaylist(o.ctx, info, dbID, ownerID) })
	})
}

func (o *playlistGeneratorOrchestrator) scheduleDiscovery(_ context.Context, delay time.Duration) {
	if o.discoveryTimer != nil {
		o.discoveryTimer.Stop()
	}
	o.discoveryTimer = time.AfterFunc(delay, func() {
		o.wg.Go(func() { o.discoverAndSync(o.ctx) })
	})
}

// stop cancels the context, stops all timers, and waits for in-flight goroutines.
func (o *playlistGeneratorOrchestrator) stop() {
	o.cancel()
	if o.discoveryTimer != nil {
		o.discoveryTimer.Stop()
	}
	for _, timer := range o.refreshTimers {
		timer.Stop()
	}
	o.wg.Wait()
}
