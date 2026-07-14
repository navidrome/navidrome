package jellyfin

import (
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
)

// audioMuseEndpoints is the list reported by /AudioMuseAI/info. It excludes info itself,
// matching the reference plugin, whose reflection-built list drops the info route.
var audioMuseEndpoints = []string{
	"GET /AudioMuseAI/similar_tracks",
	"GET /AudioMuseAI/find_path",
}

type audioMuseInfoResponse struct {
	Version            string   `json:"Version"`
	AvailableEndpoints []string `json:"AvailableEndpoints"`
}

func (api *Router) audioMuseInfo(w http.ResponseWriter, r *http.Request) {
	// Like getOpenSubsonicExtensions: advertise the sonic endpoints only when a provider is loaded.
	endpoints := []string{}
	if api.sonic != nil && api.sonic.HasProvider() {
		endpoints = audioMuseEndpoints
	}
	api.ok(w, r, audioMuseInfoResponse{
		Version:            consts.Version,
		AvailableEndpoints: endpoints,
	})
}

type audioMuseSimilarTrack struct {
	Author   string  `json:"author"`
	Distance float64 `json:"distance"`
	ItemID   string  `json:"item_id"`
	Title    string  `json:"title"`
}

func (api *Router) audioMuseSimilarTracks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Gate the whole endpoint on a sonic provider, like the Subsonic sonicSimilarity handlers.
	if api.sonic == nil || !api.sonic.HasProvider() {
		api.notFound(w, r)
		return
	}
	p := req.Params(r)
	tracks := []audioMuseSimilarTrack{}

	itemID := p.StringOr("item_id", "")
	if itemID == "" {
		api.ok(w, r, tracks)
		return
	}

	id := api.resolveItemID(ctx, dto.DecodeID(itemID))
	n := p.IntOr("n", 10)
	eliminateDuplicates := p.BoolOr("eliminate_duplicates", true)

	matches, err := api.sonic.GetSonicSimilarTracks(ctx, id, n)
	if err != nil {
		api.ok(w, r, tracks)
		return
	}

	u, _ := request.UserFrom(ctx)
	seenArtists := map[string]bool{}
	for _, m := range matches {
		mf := m.MediaFile
		if !u.HasLibraryAccess(mf.LibraryID) {
			continue
		}
		if eliminateDuplicates {
			key := strings.ToLower(mf.Artist)
			if seenArtists[key] {
				continue
			}
			seenArtists[key] = true
		}
		tracks = append(tracks, audioMuseSimilarTrack{
			Author:   mf.Artist,
			Distance: m.Similarity,
			ItemID:   dto.EncodeID(mf.ID),
			Title:    mf.Title,
		})
	}
	api.ok(w, r, tracks)
}
