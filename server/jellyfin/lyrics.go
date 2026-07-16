package jellyfin

import (
	"context"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

// cachedLyrics resolves lyrics through the full source pipeline (embedded, sidecar, plugins),
// caching results — including empty: clients poll per played track, so misses are the hot path.
func (api *Router) cachedLyrics(ctx context.Context, mf *model.MediaFile) model.LyricList {
	list, err := api.lyricsCache.GetWithLoader(mf.ID, func(string) (model.LyricList, time.Duration, error) {
		l, err := api.lyrics.GetLyrics(ctx, mf)
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
	main, found := api.cachedLyrics(r.Context(), mf).Main()
	if !found || main.IsEmpty() {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	api.ok(w, r, dto.LyricDtoFromLyrics(*mf, main))
}
