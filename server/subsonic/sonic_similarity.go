package subsonic

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/core/sonic"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) GetSonicSimilarTracks(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	if api.sonic == nil || !api.sonic.HasProvider() {
		w.WriteHeader(http.StatusNotFound)
		return nil, nil
	}
	ctx := r.Context()
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	count := p.IntOr("count", 10)

	matches, err := api.sonic.GetSonicSimilarTracks(ctx, id, count)
	if err != nil {
		return nil, err
	}

	return sonicMatchResponse(ctx, matches), nil
}

func (api *Router) FindSonicPath(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	if api.sonic == nil || !api.sonic.HasProvider() {
		w.WriteHeader(http.StatusNotFound)
		return nil, nil
	}
	ctx := r.Context()
	p := req.Params(r)
	startSongID, err := p.String("startSongId")
	if err != nil {
		return nil, err
	}
	endSongID, err := p.String("endSongId")
	if err != nil {
		return nil, err
	}
	count := p.IntOr("count", 25)

	matches, err := api.sonic.FindSonicPath(ctx, startSongID, endSongID, count)
	if err != nil {
		return nil, err
	}

	return sonicMatchResponse(ctx, matches), nil
}

func sonicMatchResponse(ctx context.Context, matches []sonic.SimilarMatch) *responses.Subsonic {
	response := newResponse()
	resp := make(responses.Array[responses.SonicMatch], len(matches))
	for i, m := range matches {
		resp[i] = responses.SonicMatch{
			Entry:      childFromMediaFile(ctx, m.MediaFile),
			Similarity: m.Similarity,
		}
	}
	response.SonicMatches = &resp
	return response
}
