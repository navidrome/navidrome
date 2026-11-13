package jellyfin

import (
	"context"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

const lyricsLoadTimeout = time.Minute

// playbackInfoLyricsTimeout bounds the time getPlaybackInfo will wait for lyrics before
// serving direct playback info, preventing slow or hanging external plugins from stalling
// track playback start (#6423).
var playbackInfoLyricsTimeout = 100 * time.Millisecond

// playbackLyrics resolves lyrics for getPlaybackInfo without blocking the critical playback
// start path on slow external plugin network timeouts. If already cached, it returns immediately;
// otherwise it bounds the lookup so PlaybackInfo returns promptly.
func (api *Router) playbackLyrics(ctx context.Context, mf *model.MediaFile) model.LyricList {
	if list, err := api.lyricsCache.Get(mf.ID); err == nil {
		return list
	}
	resCh := make(chan model.LyricList, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error(ctx, "Panic recovered in playback lyrics lookup", "id", mf.ID, "err", r)
			}
		}()
		resCh <- api.cachedLyrics(ctx, mf)
	}()
	select {
	case list := <-resCh:
		return list
	case <-ctx.Done():
		return nil
	case <-time.After(playbackInfoLyricsTimeout):
		return nil
	}
}

// cachedLyrics resolves lyrics through the full source pipeline (embedded, sidecar, plugins),
// caching results — including empty: clients poll per played track, so misses are the hot path.
func (api *Router) cachedLyrics(ctx context.Context, mf *model.MediaFile) model.LyricList {
	// The load is shared across requests (singleflight) and cached, so don't let one
	// cancelled request abort it for everybody — detach it from the request's lifetime,
	// keeping a bound so a hung plugin can't pin the fetch (and its plugin slot) forever.
	loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), lyricsLoadTimeout)
	defer cancel()
	list, err := api.lyricsCache.GetWithLoader(mf.ID, func(string) (model.LyricList, time.Duration, error) {
		l, err := api.lyrics.GetLyrics(loadCtx, mf)
		return l, 0, err // 0 → cache DefaultTTL
	})
	if err != nil {
		log.Error(ctx, "Error getting lyrics", "id", mf.ID, "title", mf.Title, err)
		return nil
	}
	return list
}

// getLyrics serves GET /Audio/{itemId}/Lyrics. Jellyfin returns 404 when a track has no lyrics
// (never an empty 200); all surveyed clients treat that gracefully.
func (api *Router) getLyrics(w http.ResponseWriter, r *http.Request) {
	mf, ok := api.mediaFileForRequest(w, r)
	if !ok {
		return
	}
	main, found := servableLyric(api.cachedLyrics(r.Context(), mf))
	if !found {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	api.ok(w, r, dto.LyricDtoFromLyrics(*mf, main))
}

// servableLyric is the single predicate for both serving and advertising, so PlaybackInfo never
// advertises a Lyric stream that this endpoint would 404.
func servableLyric(list model.LyricList) (model.Lyrics, bool) {
	main, found := list.Main()
	return main, found && !main.IsEmpty()
}
