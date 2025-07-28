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
			log.Warn("Unknown agent ignored", "name", name)
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
	start := time.Now()
	for _, enabledAgent := range a.getEnabledAgentNames() {
		ag := a.getAgent(enabledAgent)
		if ag == nil {
			continue
		}
		if utils.IsCtxDone(ctx) {
			break
		}
		retriever, ok := ag.(ArtistMBIDRetriever)
		if !ok {
			continue
		}
		mbid, err := retriever.GetArtistMBID(ctx, id, name)
		if mbid != "" && err == nil {
			log.Debug(ctx, "Got MBID", "agent", ag.AgentName(), "artist", name, "mbid", mbid, "elapsed", time.Since(start))
			return mbid, nil
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	switch id {
	case consts.UnknownArtistID:
		return "", ErrNotFound
	case consts.VariousArtistsID:
		return "", nil
	}
	start := time.Now()
	for _, enabledAgent := range a.getEnabledAgentNames() {
		ag := a.getAgent(enabledAgent)
		if ag == nil {
			continue
		}
		if utils.IsCtxDone(ctx) {
			break
		}
		retriever, ok := ag.(ArtistURLRetriever)
		if !ok {
			continue
		}
		url, err := retriever.GetArtistURL(ctx, id, name, mbid)
		if url != "" && err == nil {
			log.Debug(ctx, "Got External Url", "agent", ag.AgentName(), "artist", name, "url", url, "elapsed", time.Since(start))
			return url, nil
		}
	}
	return "", ErrNotFound
}

func (a *Agents) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	switch id {
	case consts.UnknownArtistID:
		return "", ErrNotFound
	case consts.VariousArtistsID:
		return "", nil
	}
	start := time.Now()
	for _, enabledAgent := range a.getEnabledAgentNames() {
		ag := a.getAgent(enabledAgent)
		if ag == nil {
			continue
		}
		if utils.IsCtxDone(ctx) {
			break
		}
		retriever, ok := ag.(ArtistBiographyRetriever)
		if !ok {
			continue
		}
		bio, err := retriever.GetArtistBiography(ctx, id, name, mbid)
		if err == nil {
			log.Debug(ctx, "Got Biography", "agent", ag.AgentName(), "artist", name, "len", len(bio), "elapsed", time.Since(start))
			return bio, nil
		}
	}
	return "", ErrNotFound
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
	start := time.Now()
	for _, enabledAgent := range a.getEnabledAgentNames() {
		ag := a.getAgent(enabledAgent)
		if ag == nil {
			continue
		}
		if utils.IsCtxDone(ctx) {
			break
		}
		retriever, ok := ag.(ArtistImageRetriever)
		if !ok {
			continue
		}
		images, err := retriever.GetArtistImages(ctx, id, name, mbid)
		if len(images) > 0 && err == nil {
			log.Debug(ctx, "Got Images", "agent", ag.AgentName(), "artist", name, "images", images, "elapsed", time.Since(start))
			return images, nil
		}
	}
	return nil, ErrNotFound
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

	start := time.Now()
	for _, enabledAgent := range a.getEnabledAgentNames() {
		ag := a.getAgent(enabledAgent)
		if ag == nil {
			continue
		}
		if utils.IsCtxDone(ctx) {
			break
		}
		retriever, ok := ag.(ArtistTopSongsRetriever)
		if !ok {
			continue
		}
		songs, err := retriever.GetArtistTopSongs(ctx, id, artistName, mbid, overLimit)
		if len(songs) > 0 && err == nil {
			log.Debug(ctx, "Got Top Songs", "agent", ag.AgentName(), "artist", artistName, "songs", songs, "elapsed", time.Since(start))
			return songs, nil
		}
	}
	return nil, ErrNotFound
}

func (a *Agents) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*AlbumInfo, error) {
	if name == consts.UnknownAlbum {
		return nil, ErrNotFound
	}
	start := time.Now()
	for _, enabledAgent := range a.getEnabledAgentNames() {
		ag := a.getAgent(enabledAgent)
		if ag == nil {
			continue
		}
		if utils.IsCtxDone(ctx) {
			break
		}
		retriever, ok := ag.(AlbumInfoRetriever)
		if !ok {
			continue
		}
		album, err := retriever.GetAlbumInfo(ctx, name, artist, mbid)
		if err == nil {
			log.Debug(ctx, "Got Album Info", "agent", ag.AgentName(), "album", name, "artist", artist,
				"mbid", mbid, "elapsed", time.Since(start))
			return album, nil
		}
	}
	return nil, ErrNotFound
}

func (a *Agents) GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]ExternalImage, error) {
	if name == consts.UnknownAlbum {
		return nil, ErrNotFound
	}
	start := time.Now()
	for _, enabledAgent := range a.getEnabledAgentNames() {
		ag := a.getAgent(enabledAgent)
		if ag == nil {
			continue
		}
		if utils.IsCtxDone(ctx) {
			break
		}
		retriever, ok := ag.(AlbumImageRetriever)
		if !ok {
			continue
		}
		images, err := retriever.GetAlbumImages(ctx, name, artist, mbid)
		if len(images) > 0 && err == nil {
			log.Debug(ctx, "Got Album Images", "agent", ag.AgentName(), "album", name, "artist", artist,
				"mbid", mbid, "elapsed", time.Since(start))
			return images, nil
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
