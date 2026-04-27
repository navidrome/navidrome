package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) GetSonicSimilarTracks(r *http.Request) (*responses.Subsonic, error) {
	if api.sonic == nil {
		return nil, newError(responses.ErrorDataNotFound, "sonicSimilarity not supported")
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

	response := newResponse()
	response.SonicMatches = make([]responses.SonicMatch, len(matches))
	for i, m := range matches {
		response.SonicMatches[i] = responses.SonicMatch{
			Entry:      childFromMediaFile(ctx, m.MediaFile),
			Similarity: m.Similarity,
		}
	}
	return response, nil
}

func (api *Router) FindSonicPath(r *http.Request) (*responses.Subsonic, error) {
	if api.sonic == nil {
		return nil, newError(responses.ErrorDataNotFound, "sonicSimilarity not supported")
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

	response := newResponse()
	response.SonicMatches = make([]responses.SonicMatch, len(matches))
	for i, m := range matches {
		response.SonicMatches[i] = responses.SonicMatch{
			Entry:      childFromMediaFile(ctx, m.MediaFile),
			Similarity: m.Similarity,
		}
	}
	return response, nil
}
