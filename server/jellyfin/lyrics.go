package jellyfin

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

// lyricsFor resolves lyrics through the full source pipeline (embedded, sidecar, plugins).
func (api *Router) lyricsFor(ctx context.Context, mf *model.MediaFile) model.LyricList {
	list, err := api.lyrics.GetLyrics(ctx, mf)
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
	main, found := servableLyric(api.lyricsFor(r.Context(), mf))
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
