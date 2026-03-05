package plugins

import (
	"context"
	"fmt"
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
	refreshTimers  map[string]*time.Timer // keyed by playlist DB ID
	discoveryTimer *time.Timer
}

func newPlaylistGeneratorOrchestrator(pluginName string, p *plugin, ds model.DataStore) *playlistGeneratorOrchestrator {
	return &playlistGeneratorOrchestrator{
		pluginName:    pluginName,
		plugin:        p,
		ds:            ds,
		matcher:       matcher.New(ds),
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
		dbID := id.NewHash(o.pluginName, info.ID, info.OwnerUserID)
		o.syncPlaylist(ctx, info, dbID)
	}

	// Schedule re-discovery if RefreshInterval > 0
	if resp.RefreshInterval > 0 {
		o.scheduleDiscovery(ctx, time.Duration(resp.RefreshInterval)*time.Second)
	}
}

// syncPlaylist calls GetPlaylist, matches tracks, and upserts the playlist in the DB.
func (o *playlistGeneratorOrchestrator) syncPlaylist(ctx context.Context, info capabilities.PlaylistInfo, dbID string) {
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
		OwnerID:          info.OwnerUserID,
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
	pls.Tracks = tracks

	// Upsert via repository
	plsRepo := o.ds.Playlist(ctx)
	if err := plsRepo.Put(pls); err != nil {
		log.Error(ctx, "Failed to upsert plugin playlist", "plugin", o.pluginName, "playlistID", info.ID, err)
		return
	}

	log.Info(ctx, "Synced plugin playlist", "plugin", o.pluginName, "playlistID", info.ID,
		"name", resp.Name, "tracks", len(matched), "owner", info.OwnerUserID)

	// Schedule refresh if ValidUntil > 0
	if resp.ValidUntil > 0 {
		validUntil := time.Unix(resp.ValidUntil, 0)
		delay := time.Until(validUntil)
		if delay <= 0 {
			delay = 1 * time.Second // Already expired, refresh soon
		}
		o.schedulePlaylistRefresh(ctx, info, dbID, delay)
	}
}

func (o *playlistGeneratorOrchestrator) schedulePlaylistRefresh(_ context.Context, info capabilities.PlaylistInfo, dbID string, delay time.Duration) {
	// Cancel existing timer if any
	if timer, ok := o.refreshTimers[dbID]; ok {
		timer.Stop()
	}
	o.refreshTimers[dbID] = time.AfterFunc(delay, func() {
		o.syncPlaylist(context.Background(), info, dbID)
	})
}

func (o *playlistGeneratorOrchestrator) scheduleDiscovery(_ context.Context, delay time.Duration) {
	if o.discoveryTimer != nil {
		o.discoveryTimer.Stop()
	}
	o.discoveryTimer = time.AfterFunc(delay, func() {
		o.discoverAndSync(context.Background())
	})
}

// stop cancels all timers.
func (o *playlistGeneratorOrchestrator) stop() {
	if o.discoveryTimer != nil {
		o.discoveryTimer.Stop()
	}
	for _, timer := range o.refreshTimers {
		timer.Stop()
	}
}
