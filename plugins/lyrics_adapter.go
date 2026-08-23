package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

// lyricsPluginCallTimeout bounds work detached from one caller's request so a
// disconnected client cannot leave a shared plugin lookup running forever.
const lyricsPluginCallTimeout = time.Minute

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

// GetLyrics coalesces concurrent lookups for the same track. The shared call is
// detached from any one request so one disconnected client does not cancel it
// for the remaining callers.
func (l *LyricsPlugin) GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	req := capabilities.GetLyricsRequest{
		Track: mediaFileToTrackInfo(l.plugin, mf),
	}
	key, err := lyricsPluginCallKey(req)
	if err != nil {
		return nil, err
	}

	result := l.plugin.lyricsCalls.DoChan(key, func() (any, error) {
		callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), lyricsPluginCallTimeout)
		defer cancel()
		return l.getLyrics(callCtx, mf, req)
	})

	select {
	case call := <-result:
		if call.Err != nil {
			return nil, call.Err
		}
		lyricsList, ok := call.Val.(model.LyricList)
		if !ok {
			return nil, fmt.Errorf("unexpected lyrics plugin result type %T", call.Val)
		}
		return lyricsList, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func lyricsPluginCallKey(req capabilities.GetLyricsRequest) (string, error) {
	value, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("encode lyrics plugin request key: %w", err)
	}
	return string(value), nil
}

// getLyrics calls the plugin, then content-sniffs each response via
// model.ParseLyrics (TTML/SRT/YAML/LRC/plain).
func (l *LyricsPlugin) getLyrics(ctx context.Context, mf *model.MediaFile, req capabilities.GetLyricsRequest) (model.LyricList, error) {
	select {
	case l.plugin.lyricsSem <- struct{}{}:
		defer func() { <-l.plugin.lyricsSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
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
