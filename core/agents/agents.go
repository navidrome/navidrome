package agents

import (
	"context"
	"sort"
	"strings"
	"sync"
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
	PluginNames(serviceName string) []string
	// LoadMediaAgent loads and returns a media agent plugin
	LoadMediaAgent(name string) (Interface, bool)
}

type cachedAgent struct {
	agent      Interface
	expiration time.Time
}

type Agents struct {
	ds           model.DataStore
	pluginLoader PluginLoader
	cachedAgents map[string]cachedAgent // cached agent instances with expiration
	mu           sync.Mutex             // protects cachedAgents map
}

// TTL for cached agents
const agentCacheTTL = 5 * time.Minute

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
		cachedAgents: make(map[string]cachedAgent),
	}
}

// getEnabledAgentNames returns the current list of enabled agent names, including:
// 1. Built-in agents and plugins from config (in the specified order)
// 2. Always include LocalAgentName
// 3. If config is empty, include all available agents and plugins in a default order
func (a *Agents) getEnabledAgentNames() []string {
	// Get all available plugin names
	var availablePlugins []string
	if a.pluginLoader != nil {
		availablePlugins = a.pluginLoader.PluginNames("MediaMetadataService")
	}

	// If agents are explicitly configured, use that order
	if conf.Server.Agents != "" {
		configuredAgents := strings.Split(conf.Server.Agents, ",")

		// Always add LocalAgentName if not already included
		hasLocalAgent := false
		for _, name := range configuredAgents {
			if name == LocalAgentName {
				hasLocalAgent = true
				break
			}
		}
		if !hasLocalAgent {
			configuredAgents = append(configuredAgents, LocalAgentName)
		}

		// Filter to only include valid agents (built-in or plugins)
		var validNames []string
		for _, name := range configuredAgents {
			isBuiltIn := false
			isPlugin := false

			// Check if it's a built-in agent
			if _, ok := Map[name]; ok {
				isBuiltIn = true
			}

			// Check if it's a plugin
			for _, pluginName := range availablePlugins {
				if pluginName == name {
					isPlugin = true
					break
				}
			}

			if isBuiltIn || isPlugin {
				validNames = append(validNames, name)
			} else {
				log.Warn("Unknown agent ignored", "name", name)
			}
		}
		return validNames
	}

	// If no agents configured, use all available built-in agents and plugins
	var allAgents []string

	// Add all built-in agents except local (which will be added at the end)
	for name := range Map {
		if name != LocalAgentName {
			allAgents = append(allAgents, name)
		}
	}

	// Add all plugins
	allAgents = append(allAgents, availablePlugins...)

	// Sort for consistent ordering
	sort.Strings(allAgents)

	// Always add LocalAgentName last
	allAgents = append(allAgents, LocalAgentName)

	return allAgents
}

func (a *Agents) getAgent(name string) Interface {
	now := time.Now()

	// Check cache first
	a.mu.Lock()
	cached, ok := a.cachedAgents[name]
	if ok && cached.expiration.After(now) {
		a.mu.Unlock()
		return cached.agent
	}
	a.mu.Unlock()

	// Try to get built-in agent
	constructor, ok := Map[name]
	if ok {
		agent := constructor(a.ds)
		if agent != nil {
			// Cache the agent with expiration
			a.mu.Lock()
			a.cachedAgents[name] = cachedAgent{
				agent:      agent,
				expiration: now.Add(agentCacheTTL),
			}
			a.mu.Unlock()
			return agent
		}
		log.Debug("Built-in agent not available. Missing configuration?", "name", name)
	}

	// Try to load WASM plugin agent (if plugin loader is available)
	if a.pluginLoader != nil {
		agent, ok := a.pluginLoader.LoadMediaAgent(name)
		if ok && agent != nil {
			// Cache the plugin agent with expiration
			a.mu.Lock()
			a.cachedAgents[name] = cachedAgent{
				agent:      agent,
				expiration: now.Add(agentCacheTTL),
			}
			a.mu.Unlock()
			return agent
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
	for _, agentName := range a.getEnabledAgentNames() {
		ag := a.getAgent(agentName)
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
	for _, agentName := range a.getEnabledAgentNames() {
		ag := a.getAgent(agentName)
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
	for _, agentName := range a.getEnabledAgentNames() {
		ag := a.getAgent(agentName)
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

func (a *Agents) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]Artist, error) {
	switch id {
	case consts.UnknownArtistID:
		return nil, ErrNotFound
	case consts.VariousArtistsID:
		return nil, nil
	}
	start := time.Now()
	for _, agentName := range a.getEnabledAgentNames() {
		ag := a.getAgent(agentName)
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
		similar, err := retriever.GetSimilarArtists(ctx, id, name, mbid, limit)
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
	for _, agentName := range a.getEnabledAgentNames() {
		ag := a.getAgent(agentName)
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

func (a *Agents) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]Song, error) {
	switch id {
	case consts.UnknownArtistID:
		return nil, ErrNotFound
	case consts.VariousArtistsID:
		return nil, nil
	}
	start := time.Now()
	for _, agentName := range a.getEnabledAgentNames() {
		ag := a.getAgent(agentName)
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
		songs, err := retriever.GetArtistTopSongs(ctx, id, artistName, mbid, count)
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
	for _, agentName := range a.getEnabledAgentNames() {
		ag := a.getAgent(agentName)
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
	for _, agentName := range a.getEnabledAgentNames() {
		ag := a.getAgent(agentName)
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
