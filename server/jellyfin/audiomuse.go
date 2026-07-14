package jellyfin

import (
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
)

// audioMuseEndpoints is what /AudioMuseAI/info advertises; it omits info itself, like the plugin,
// and is sorted the same way (the plugin builds it with OrderBy).
var audioMuseEndpoints = []string{
	"GET /AudioMuseAI/find_path",
	"GET /AudioMuseAI/health",
	"GET /AudioMuseAI/similar_tracks",
}

type audioMuseInfoResponse struct {
	Version            string   `json:"Version"`
	AvailableEndpoints []string `json:"AvailableEndpoints"`
}

func (api *Router) audioMuseInfo(w http.ResponseWriter, r *http.Request) {
	endpoints := []string{} // non-nil so an empty list serializes as [], not null
	if api.sonic != nil && api.sonic.HasProvider() {
		endpoints = audioMuseEndpoints
	}
	api.ok(w, r, audioMuseInfoResponse{
		Version:            consts.Version,
		AvailableEndpoints: endpoints,
	})
}

// audioMuseHealth is a liveness probe: 200 with an empty body when a sonic provider is loaded, else
// 404 — mirroring the reference plugin, which returns 200 when its backend is reachable.
func (api *Router) audioMuseHealth(w http.ResponseWriter, r *http.Request) {
	if api.sonic == nil || !api.sonic.HasProvider() {
		api.notFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type audioMuseSimilarTrack struct {
	Author   string  `json:"author"`
	Distance float64 `json:"distance"`
	ItemID   string  `json:"item_id"`
	Title    string  `json:"title"`
}

func (api *Router) audioMuseSimilarTracks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// 404 without a provider, like the Subsonic sonicSimilarity handlers.
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
	n := min(p.IntOr("n", 10), maxSimilarLimit) // cap a user-controlled count, like clampLimit
	eliminateDuplicates := p.BoolOr("eliminate_duplicates", true)

	matches, err := api.sonic.GetSonicSimilarTracks(ctx, id, n)
	if err != nil {
		api.ok(w, r, tracks)
		return
	}

	u, _ := request.UserFrom(ctx)
	seenArtists := make(map[string]bool, len(matches))
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

type audioMusePathTrack struct {
	Author string   `json:"author"`
	ItemID string   `json:"item_id"`
	Title  string   `json:"title"`
	Tempo  *float64 `json:"tempo,omitempty"`
}

type audioMusePathResponse struct {
	Path          []audioMusePathTrack `json:"path"`
	TotalDistance float64              `json:"total_distance"`
}

func (api *Router) audioMuseFindPath(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if api.sonic == nil || !api.sonic.HasProvider() {
		api.notFound(w, r)
		return
	}
	p := req.Params(r)

	startID := p.StringOr("start_song_id", "")
	endID := p.StringOr("end_song_id", "")
	if startID == "" || endID == "" {
		http.Error(w, "start_song_id and end_song_id are required.", http.StatusBadRequest)
		return
	}

	resp := audioMusePathResponse{Path: []audioMusePathTrack{}}
	maxSteps := min(p.IntOr("max_steps", 25), maxSimilarLimit) // cap a user-controlled count
	matches, err := api.sonic.FindSonicPath(ctx,
		api.resolveItemID(ctx, dto.DecodeID(startID)),
		api.resolveItemID(ctx, dto.DecodeID(endID)),
		maxSteps)
	if err != nil {
		api.ok(w, r, resp)
		return
	}

	u, _ := request.UserFrom(ctx)
	for _, m := range matches {
		mf := m.MediaFile
		if !u.HasLibraryAccess(mf.LibraryID) {
			continue
		}
		track := audioMusePathTrack{
			Author: mf.Artist,
			ItemID: dto.EncodeID(mf.ID),
			Title:  mf.Title,
		}
		if mf.BPM != nil {
			tempo := float64(*mf.BPM)
			track.Tempo = &tempo
		}
		resp.Path = append(resp.Path, track)
		resp.TotalDistance += m.Similarity
	}
	api.ok(w, r, resp)
}
