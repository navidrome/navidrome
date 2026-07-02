package plugins

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

const CapabilityLyrics Capability = "Lyrics"

const (
	FuncLyricsGetLyrics = "nd_lyrics_get_lyrics"
)

func init() {
	registerCapability(
		CapabilityLyrics,
		FuncLyricsGetLyrics,
	)
}

func newLyricsPlugin(p *plugin) *LyricsPlugin {
	return &LyricsPlugin{name: p.name, plugin: p}
}

// LyricsPlugin adapts a WASM plugin with the Lyrics capability.
type LyricsPlugin struct {
	name   string
	plugin *plugin
}

// GetLyrics calls the plugin to fetch lyrics, then content-sniffs each response
// via model.ParseLyrics (TTML/SRT/YAML/LRC/plain).
func (l *LyricsPlugin) GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
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
		for _, lyric := range parsed {
			if !lyric.IsEmpty() {
				result = append(result, lyric)
			}
		}
	}
	return result, nil
}
