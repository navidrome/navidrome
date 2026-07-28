package plugins

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

// CapabilityPodcastScrobbler indicates the plugin can receive podcast play events.
// Detected when the plugin exports the nd_podcast_scrobbler_on_played function.
const CapabilityPodcastScrobbler Capability = "PodcastScrobbler"

const FuncPodcastScrobblerOnPlayed = "nd_podcast_scrobbler_on_played"

func init() {
	registerCapability(CapabilityPodcastScrobbler, FuncPodcastScrobblerOnPlayed)
}

func newPodcastScrobblerPlugin(p *plugin) *PodcastScrobblerPlugin {
	return &PodcastScrobblerPlugin{plugin: p}
}

// PodcastScrobblerPlugin wraps a plugin and routes podcast play events to it.
type PodcastScrobblerPlugin struct {
	plugin *plugin
}

func (ps *PodcastScrobblerPlugin) OnPodcastPlayed(ctx context.Context, username string, episode *model.PodcastEpisode, channelTitle string) error {
	input := capabilities.PodcastPlayedRequest{
		Username: username,
		Episode: capabilities.EpisodeInfo{
			ID:           episode.ID,
			Title:        episode.Title,
			ChannelID:    episode.ChannelID,
			ChannelTitle: channelTitle,
			Duration:     episode.Duration,
			Timestamp:    time.Now().Unix(),
		},
	}
	return callPluginFunctionNoOutput(ctx, ps.plugin, FuncPodcastScrobblerOnPlayed, input)
}

// LoadPodcastScrobbler returns a PodcastScrobblerPlugin for the named plugin if it
// implements the PodcastScrobbler capability.
func (m *Manager) LoadPodcastScrobbler(name string) (*PodcastScrobblerPlugin, bool) {
	return loadPlugin(m, name, CapabilityPodcastScrobbler, newPodcastScrobblerPlugin)
}

// DispatchPodcastPlayed calls OnPodcastPlayed on all loaded PodcastScrobbler plugins.
// Errors from individual plugins are logged but do not stop dispatch to remaining plugins.
func (m *Manager) DispatchPodcastPlayed(ctx context.Context, username string, episode *model.PodcastEpisode, channelTitle string) {
	for _, name := range m.PluginNames(string(CapabilityPodcastScrobbler)) {
		p, ok := m.LoadPodcastScrobbler(name)
		if !ok {
			continue
		}
		if err := p.OnPodcastPlayed(ctx, username, episode, channelTitle); err != nil {
			log.Warn(ctx, "Plugin podcast scrobbler error", "plugin", name, err)
		}
	}
}
