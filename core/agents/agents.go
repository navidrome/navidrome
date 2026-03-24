package agents

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/singleton"
)

// PluginLoader defines an interface for loading plugins
type PluginLoader interface {
	// PluginNames returns the names of all plugins that implement a particular service
	PluginNames(capability string) []string
	// LoadMediaAgent loads and returns a media agent plugin
	LoadMediaAgent(name string) (Interface, bool)
}

// Agents is a meta-agent that aggregates multiple built-in and plugin agents. It tries each enabled agent in order
// until one returns valid data.
type Agents struct {
	ds           model.DataStore
	pluginLoader PluginLoader
}

// GetAgents returns the singleton instance of Agents
func GetAgents(ds model.DataStore, pluginLoader PluginLoader) *Agents {
	return singleton.GetInstance(func() *Agents {
		return createAgents(ds, pluginLoader)
	})
}

// createAgents creates a new Agents instance. Used in tests
func createAgents(ds model.DataStore, pluginLoader PluginLoader) *Agents {
	return &Agents{
		ds:           ds,
		pluginLoader: pluginLoader,
	}
}

// enabledAgent represents an enabled agent with its type information
type enabledAgent struct {
	name     string
	isPlugin bool
}

// getEnabledAgentNames returns the current list of enabled agents, including:
// 1. Built-in agents and plugins from config (in the specified order)
// 2. Always include LocalAgentName
// 3. If config is empty, include ONLY LocalAgentName
// Each enabledAgent contains the name and whether it's a plugin (true) or built-in (false)
func (a *Agents) getEnabledAgentNames() []enabledAgent {
	// If no agents configured, ONLY use the local agent
	if conf.Server.Agents == "" {
		return []enabledAgent{{name: LocalAgentName, isPlugin: false}}
	}

	// Get all available plugin names
	var availablePlugins []string
	if a.pluginLoader != nil {
		availablePlugins = a.pluginLoader.PluginNames("MetadataAgent")
	}
	log.Trace("Available MetadataAgent plugins", "plugins", availablePlugins)

	configuredAgents := strings.Split(conf.Server.Agents, ",")

	// Always add LocalAgentName if not already included
	hasLocalAgent := slices.Contains(configuredAgents, LocalAgentName)
	if !hasLocalAgent {
		configuredAgents = append(configuredAgents, LocalAgentName)
	}

	// Filter to only include valid agents (built-in or plugins)
	var validAgents []enabledAgent
	for _, name := range configuredAgents {
		// Check if it's a built-in agent
		isBuiltIn := Map[name] != nil

		// Check if it's a plugin
		isPlugin := slices.Contains(availablePlugins, name)

		if isBuiltIn {
			validAgents = append(validAgents, enabledAgent{name: name, isPlugin: false})
		} else if isPlugin {
			validAgents = append(validAgents, enabledAgent{name: name, isPlugin: true})
		} else {
			log.Debug("Unknown agent ignored", "name", name)
		}
	}
	return validAgents
}

func (a *Agents) getAgent(ea enabledAgent) Interface {
	if ea.isPlugin {
		// Try to load WASM plugin agent (if plugin loader is available)
		if a.pluginLoader != nil {
			agent, ok := a.pluginLoader.LoadMediaAgent(ea.name)
			if ok && agent != nil {
				return agent
			}
		}
	} else {
		// Try to get built-in agent
		constructor, ok := Map[ea.name]
		if ok {
			agent := constructor(a.ds)
			if agent != nil {
				return agent
			}
			log.Debug("Built-in agent not available. Missing configuration?", "name", ea.name)
		}
	}

	return nil
}

func (a *Agents) AgentName() string {
	return "agents"
}

func (a *Agents) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	switch id {
	case consts.UnknownArtistID:
		return "", ErrNotFound
	case consts.VariousArtistsID:
		return "", nil
	}

	return callAgentMethod(ctx, a, "GetArtistMBID", func(ag Interface) (string, error) {
		retriever, ok := ag.(ArtistMBIDRetriever)
		if !ok {
			return "", ErrNotFound
		}
		return retriever.GetArtistMBID(ctx, id, name)
	})
}

func (a *Agents) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	switch id {
	case consts.UnknownArtistID:
		return "", ErrNotFound
	case consts.VariousArtistsID:
		return "", nil
	}

	return callAgentMethod(ctx, a, "GetArtistURL", func(ag Interface) (string, error) {
		retriever, ok := ag.(ArtistURLRetriever)
		if !ok {
			return "", ErrNotFound
		}
		return retriever.GetArtistURL(ctx, id, name, mbid)
	})
}

func (a *Agents) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	switch id {
	case consts.UnknownArtistID:
		return "", ErrNotFound
	case consts.VariousArtistsID:
		return "", nil
	}

	return callAgentMethod(ctx, a, "GetArtistBiography", func(ag Interface) (string, error) {
		retriever, ok := ag.(ArtistBiographyRetriever)
		if !ok {
			return "", ErrNotFound
		}
		return retriever.GetArtistBiography(ctx, id, name, mbid)
	})
}

// GetSimilarArtists returns similar artists by id, name, and/or mbid. Because some artists returned from an enabled
// agent may not exist in the database, return at most limit * conf.Server.DevExternalArtistFetchMultiplier items.
func (a *Agents) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]Artist, error) {
	switch id {
	case consts.UnknownArtistID:
		return nil, ErrNotFound
	case consts.VariousArtistsID:
		return nil, nil
	}

	overLimit := int(float64(limit) * conf.Server.DevExternalArtistFetchMultiplier)

	start := time.Now()
	for _, enabledAgent := range a.getEnabledAgentNames() {
		ag := a.getAgent(enabledAgent)
		if ag == nil {
			continue
		}
		if utils.IsCtxDone(ctx) {
			break
		}
		retriever, ok := ag.(ArtistSimilarRetriever)
		if !ok {
			continue
		}
		similar, err := retriever.GetSimilarArtists(ctx, id, name, mbid, overLimit)
		if len(similar) > 0 && err == nil {
			if log.IsGreaterOrEqualTo(log.LevelTrace) {
				log.Debug(ctx, "Got Similar Artists", "agent", ag.AgentName(), "artist", name, "similar", similar, "elapsed", time.Since(start))
			} else {
				log.Debug(ctx, "Got Similar Artists", "agent", ag.AgentName(), "artist", name, "similarReceived", len(similar), "elapsed", time.Since(start))
			}
			return similar, err
		}
	}
	return nil, ErrNotFound
}

func (a *Agents) GetArtistImages(ctx context.Context, id, name, mbid string) ([]ExternalImage, error) {
	switch id {
	case consts.UnknownArtistID:
		return nil, ErrNotFound
	case consts.VariousArtistsID:
		return nil, nil
	}

	return callAgentSliceMethod(ctx, a, "GetArtistImages", func(ag Interface) ([]ExternalImage, error) {
		retriever, ok := ag.(ArtistImageRetriever)
		if !ok {
			return nil, ErrNotFound
		}
		return retriever.GetArtistImages(ctx, id, name, mbid)
	})
}

// GetArtistTopSongs returns top songs by id, name, and/or mbid. Because some songs returned from an enabled
// agent may not exist in the database, return at most limit * conf.Server.DevExternalArtistFetchMultiplier items.
func (a *Agents) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]Song, error) {
	switch id {
	case consts.UnknownArtistID:
		return nil, ErrNotFound
	case consts.VariousArtistsID:
		return nil, nil
	}

	overLimit := int(float64(count) * conf.Server.DevExternalArtistFetchMultiplier)

	return callAgentSliceMethod(ctx, a, "GetArtistTopSongs", func(ag Interface) ([]Song, error) {
		retriever, ok := ag.(ArtistTopSongsRetriever)
		if !ok {
			return nil, ErrNotFound
		}
		return retriever.GetArtistTopSongs(ctx, id, artistName, mbid, overLimit)
	})
}

func (a *Agents) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*AlbumInfo, error) {
	if name == consts.UnknownAlbum {
		return nil, ErrNotFound
	}

	return callAgentMethod(ctx, a, "GetAlbumInfo", func(ag Interface) (*AlbumInfo, error) {
		retriever, ok := ag.(AlbumInfoRetriever)
		if !ok {
			return nil, ErrNotFound
		}
		return retriever.GetAlbumInfo(ctx, name, artist, mbid)
	})
}

func (a *Agents) GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]ExternalImage, error) {
	if name == consts.UnknownAlbum {
		return nil, ErrNotFound
	}

	return callAgentSliceMethod(ctx, a, "GetAlbumImages", func(ag Interface) ([]ExternalImage, error) {
		retriever, ok := ag.(AlbumImageRetriever)
		if !ok {
			return nil, ErrNotFound
		}
		return retriever.GetAlbumImages(ctx, name, artist, mbid)
	})
}

// GetSimilarSongsByTrack returns similar songs for a given track.
func (a *Agents) GetSimilarSongsByTrack(ctx context.Context, id, name, artist, mbid string, count int) ([]Song, error) {
	return callAgentSliceMethod(ctx, a, "GetSimilarSongsByTrack", func(ag Interface) ([]Song, error) {
		retriever, ok := ag.(SimilarSongsByTrackRetriever)
		if !ok {
			return nil, ErrNotFound
		}
		return retriever.GetSimilarSongsByTrack(ctx, id, name, artist, mbid, count)
	})
}

// GetSimilarSongsByAlbum returns similar songs for a given album.
func (a *Agents) GetSimilarSongsByAlbum(ctx context.Context, id, name, artist, mbid string, count int) ([]Song, error) {
	return callAgentSliceMethod(ctx, a, "GetSimilarSongsByAlbum", func(ag Interface) ([]Song, error) {
		retriever, ok := ag.(SimilarSongsByAlbumRetriever)
		if !ok {
			return nil, ErrNotFound
		}
		return retriever.GetSimilarSongsByAlbum(ctx, id, name, artist, mbid, count)
	})
}

// GetSimilarSongsByArtist returns similar songs for a given artist.
func (a *Agents) GetSimilarSongsByArtist(ctx context.Context, id, name, mbid string, count int) ([]Song, error) {
	switch id {
	case consts.UnknownArtistID:
		return nil, ErrNotFound
	case consts.VariousArtistsID:
		return nil, nil
	}

	return callAgentSliceMethod(ctx, a, "GetSimilarSongsByArtist", func(ag Interface) ([]Song, error) {
		retriever, ok := ag.(SimilarSongsByArtistRetriever)
		if !ok {
			return nil, ErrNotFound
		}
		return retriever.GetSimilarSongsByArtist(ctx, id, name, mbid, count)
	})
}

func callAgentMethod[T comparable](ctx context.Context, agents *Agents, methodName string, fn func(Interface) (T, error)) (T, error) {
	var zero T
	start := time.Now()
	for _, enabledAgent := range agents.getEnabledAgentNames() {
		ag := agents.getAgent(enabledAgent)
		if ag == nil {
			continue
		}
		if utils.IsCtxDone(ctx) {
			break
		}
		result, err := fn(ag)
		if err != nil {
			log.Trace(ctx, "Agent method call error", "method", methodName, "agent", ag.AgentName(), "error", err)
			continue
		}

		if result != zero {
			log.Debug(ctx, "Got result", "method", methodName, "agent", ag.AgentName(), "elapsed", time.Since(start))
			return result, nil
		}
	}
	return zero, ErrNotFound
}

func callAgentSliceMethod[T any](ctx context.Context, agents *Agents, methodName string, fn func(Interface) ([]T, error)) ([]T, error) {
	start := time.Now()
	for _, enabledAgent := range agents.getEnabledAgentNames() {
		ag := agents.getAgent(enabledAgent)
		if ag == nil {
			continue
		}
		if utils.IsCtxDone(ctx) {
			break
		}
		results, err := fn(ag)
		if err != nil {
			log.Trace(ctx, "Agent method call error", "method", methodName, "agent", ag.AgentName(), "error", err)
			continue
		}

		if len(results) > 0 {
			log.Debug(ctx, "Got results", "method", methodName, "agent", ag.AgentName(), "count", len(results), "elapsed", time.Since(start))
			return results, nil
		}
	}
	return nil, ErrNotFound
}

var _ Interface = (*Agents)(nil)
var _ ArtistMBIDRetriever = (*Agents)(nil)
var _ ArtistURLRetriever = (*Agents)(nil)
var _ ArtistBiographyRetriever = (*Agents)(nil)
var _ ArtistSimilarRetriever = (*Agents)(nil)
var _ ArtistImageRetriever = (*Agents)(nil)
var _ ArtistTopSongsRetriever = (*Agents)(nil)
var _ AlbumInfoRetriever = (*Agents)(nil)
var _ AlbumImageRetriever = (*Agents)(nil)
var _ SimilarSongsByTrackRetriever = (*Agents)(nil)
var _ SimilarSongsByAlbumRetriever = (*Agents)(nil)
var _ SimilarSongsByArtistRetriever = (*Agents)(nil)
