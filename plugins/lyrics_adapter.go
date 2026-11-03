package plugins

import (
	"context"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

const CapabilityLyrics Capability = "Lyrics"

const (
	FuncLyricsGetLyrics = "nd_lyrics_get_lyrics"
)

// maxConcurrentLyricsCalls caps in-flight lyrics calls per plugin: clients prefetch
// lyrics for whole queues, and the resulting burst can rate-limit upstream providers.
const maxConcurrentLyricsCalls = 2

func init() {
	registerCapability(
		CapabilityLyrics,
		FuncLyricsGetLyrics,
	)
}

func newLyricsPlugin(p *plugin) *LyricsPlugin {
	displayName := p.name
	if p.manifest != nil && strings.TrimSpace(p.manifest.Name) != "" {
		displayName = strings.TrimSpace(p.manifest.Name)
	}
	return &LyricsPlugin{name: p.name, displayName: displayName, plugin: p}
}

// LyricsPlugin adapts a WASM plugin with the Lyrics capability.
type LyricsPlugin struct {
	name        string
	displayName string
	plugin      *plugin
}

// GetLyrics calls the plugin to fetch lyrics, then content-sniffs each response
// via model.ParseLyrics (TTML/SRT/YAML/LRC/plain).
func (l *LyricsPlugin) GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	select {
	case l.plugin.lyricsSem <- struct{}{}:
		defer func() { <-l.plugin.lyricsSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	req := capabilities.GetLyricsRequest{
		Track: mediaFileToTrackInfo(l.plugin, mf),
	}
	resp, err := callPluginFunction[capabilities.GetLyricsRequest, capabilities.GetLyricsResponse](
		ctx, l.plugin, FuncLyricsGetLyrics, req,
	)
	if err != nil {
		return nil, err
	}

	// The lyric text comes from the plugin, not the media file's own tags, so
	// attribute logs to both the plugin and the track it was fetched for.
	ctx = log.NewContext(ctx, "plugin", l.name, "file", mf.Path)
	var defaults *capabilities.LyricsSource
	if resp.Source != nil {
		defaults = &capabilities.LyricsSource{
			Provider: strings.TrimSpace(resp.Source.Provider),
			Format:   strings.ToLower(strings.TrimSpace(resp.Source.Format)),
		}
	}

	var result model.LyricList
	for _, lt := range resp.Lyrics {
		lang := lt.Lang
		if lang == "" {
			lang = "xxx"
		}
		parsed, err := model.ParseLyrics(ctx, "", lang, []byte(lt.Text))
		if err != nil {
			log.Warn(ctx, "Error parsing plugin lyrics", err)
			continue
		}
		source := lyricsSourceFor(lt, defaults, l.displayName)
		for _, lyric := range parsed {
			if !lyric.IsEmpty() {
				lyric.Source = source
				result = append(result, lyric)
			}
		}
	}
	return result, nil
}

// lyricsSourceFor resolves one entry's provenance: the entry's own source wins
// field by field, the response-level source is the default for entries without
// one, and Navidrome always supplies the plugin identity itself.
func lyricsSourceFor(entry capabilities.LyricsText, defaults *capabilities.LyricsSource, pluginName string) *model.LyricsSource {
	source := &model.LyricsSource{
		Type: model.LyricsSourcePlugin,
		Name: pluginName,
	}
	if defaults != nil {
		source.Provider = defaults.Provider
		source.Format = defaults.Format
	}
	if entry.Source != nil {
		if provider := strings.TrimSpace(entry.Source.Provider); provider != "" {
			source.Provider = provider
		}
		if format := strings.ToLower(strings.TrimSpace(entry.Source.Format)); format != "" {
			source.Format = format
		}
	}
	return source
}
