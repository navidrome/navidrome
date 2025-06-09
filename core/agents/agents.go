package agents

import (
	"context"
	"slices"
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

// Encapsulates agent caching logic
// agentCache is a simple TTL cache for agents
// Not exported, only used by Agents

type agentCache struct {
	mu    sync.Mutex
	items map[string]cachedAgent
	ttl   time.Duration
}

// TTL for cached agents
const agentCacheTTL = 5 * time.Minute

func newAgentCache(ttl time.Duration) *agentCache {
	return &agentCache{
		items: make(map[string]cachedAgent),
		ttl:   ttl,
	}
}

func (c *agentCache) Get(name string) Interface {
	c.mu.Lock()
	defer c.mu.Unlock()
	cached, ok := c.items[name]
	if ok && cached.expiration.After(time.Now()) {
		return cached.agent
	}
	return nil
}

func (c *agentCache) Set(name string, agent Interface) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[name] = cachedAgent{
		agent:      agent,
		expiration: time.Now().Add(c.ttl),
	}
}

type Agents struct {
	ds           model.DataStore
	pluginLoader PluginLoader
	cache        *agentCache
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
		cache:        newAgentCache(agentCacheTTL),
	}
}

// getEnabledAgentNames returns the current list of enabled agent names, including:
// 1. Built-in agents and plugins from config (in the specified order)
// 2. Always include LocalAgentName
// 3. If config is empty, include ONLY LocalAgentName
func (a *Agents) getEnabledAgentNames() []string {
	// If no agents configured, ONLY use the local agent
	if conf.Server.Agents == "" {
		return []string{LocalAgentName}
	}

	// Get all available plugin names
	var availablePlugins []string
	if a.pluginLoader != nil {
		availablePlugins = a.pluginLoader.PluginNames("MetadataAgent")
	}

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
		// Check if it's a built-in agent
		isBuiltIn := Map[name] != nil

		// Check if it's a plugin
		isPlugin := slices.Contains(availablePlugins, name)

		if isBuiltIn || isPlugin {
			validNames = append(validNames, name)
		} else {
			log.Warn("Unknown agent ignored", "name", name)
		}
	}
	return validNames
}

func (a *Agents) getAgent(name string) Interface {
	// Check cache first
	agent := a.cache.Get(name)
	if agent != nil {
		return agent
	}

	// Try to get built-in agent
	constructor, ok := Map[name]
	if ok {
		agent := constructor(a.ds)
		if agent != nil {
			a.cache.Set(name, agent)
			return agent
		}
		log.Debug("Built-in agent not available. Missing configuration?", "name", name)
	}

	// Try to load WASM plugin agent (if plugin loader is available)
	if a.pluginLoader != nil {
		agent, ok := a.pluginLoader.LoadMediaAgent(name)
		if ok && agent != nil {
			a.cache.Set(name, agent)
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
