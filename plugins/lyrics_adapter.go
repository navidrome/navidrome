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

// LyricsPlugin adapts a WASM plugin with the Lyrics capability.
type LyricsPlugin struct {
	name   string
	plugin *plugin
}

// GetLyrics calls the plugin to fetch lyrics, then parses the raw text responses
// using model.ToLyrics.
func (l *LyricsPlugin) GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	req := capabilities.GetLyricsRequest{
		Track: mediaFileToTrackInfo(mf),
	}
	resp, err := callPluginFunction[capabilities.GetLyricsRequest, capabilities.GetLyricsResponse](
		ctx, l.plugin, FuncLyricsGetLyrics, req,
	)
	if err != nil {
		return nil, err
	}

	var result model.LyricList
	for _, lt := range resp.Lyrics {
		parsed, err := model.ToLyrics(lt.Lang, lt.Text)
		if err != nil {
			log.Warn(ctx, "Error parsing plugin lyrics", "plugin", l.name, err)
			continue
		}
		if parsed != nil && !parsed.IsEmpty() {
			result = append(result, *parsed)
		}
	}
	return result, nil
}
