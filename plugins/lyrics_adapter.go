package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
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

// lyricsCallGroup coalesces lookups while keeping their lifetime tied to the
// callers that are still waiting. One caller may leave without interrupting the
// others, but the shared work is cancelled once the last waiter is gone.
type lyricsCallGroup struct {
	mu    sync.Mutex
	calls map[string]*lyricsCall
}

type lyricsCall struct {
	group    *lyricsCallGroup
	key      string
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	lyrics   model.LyricList
	err      error
	waiters  int
	finished bool
}

func (g *lyricsCallGroup) join(
	parent context.Context,
	key string,
	lookup func(context.Context) (model.LyricList, error),
) *lyricsCall {
	g.mu.Lock()
	if call := g.calls[key]; call != nil {
		call.waiters++
		g.mu.Unlock()
		return call
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), lyricsPluginCallTimeout)
	call := &lyricsCall{
		group:   g,
		key:     key,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
		waiters: 1,
	}
	if g.calls == nil {
		g.calls = make(map[string]*lyricsCall)
	}
	g.calls[key] = call
	g.mu.Unlock()

	go call.run(lookup)
	return call
}

func (c *lyricsCall) run(lookup func(context.Context) (model.LyricList, error)) {
	lyrics, err := lookup(c.ctx)

	c.group.mu.Lock()
	c.lyrics = lyrics
	c.err = err
	c.finished = true
	if c.group.calls[c.key] == c {
		delete(c.group.calls, c.key)
	}
	close(c.done)
	c.group.mu.Unlock()
	c.cancel()
}

func (c *lyricsCall) release() {
	c.group.mu.Lock()
	c.waiters--
	shouldCancel := c.waiters == 0 && !c.finished
	if shouldCancel && c.group.calls[c.key] == c {
		delete(c.group.calls, c.key)
	}
	c.group.mu.Unlock()

	if shouldCancel {
		c.cancel()
	}
}

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

// GetLyrics coalesces concurrent lookups for the same track. The shared call
// survives individual disconnections while another caller is still waiting.
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

	call := l.plugin.lyricsCalls.join(ctx, key, func(callCtx context.Context) (model.LyricList, error) {
		return l.getLyrics(callCtx, mf, req)
	})
	defer call.release()

	select {
	case <-call.done:
		return call.lyrics, call.err
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
