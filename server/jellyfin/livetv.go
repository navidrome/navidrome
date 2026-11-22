package jellyfin

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

// getLiveTvChannels handles GET /LiveTv/Channels, where Jellyfin radio clients list stations.
func (api *Router) getLiveTvChannels(w http.ResponseWriter, r *http.Request) {
	p := req.Params(r)
	offset, limit := pageParams(p)
	if strings.EqualFold(p.StringOr("type", ""), "tv") {
		api.ok(w, r, result(nil, 0, offset))
		return
	}
	radios, err := api.ds.Radio().GetAll(r.Context(), model.QueryOptions{Sort: "name"})
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	items := slice.Map(radios, dto.RadioToBaseItem)
	api.ok(w, r, result(paginate(items, offset, limit), len(items), offset))
}

// radioForRequest returns the station {itemId} names, if it names one.
func (api *Router) radioForRequest(r *http.Request) (*model.Radio, bool) {
	id, ok := dto.DecodeID(chi.URLParam(r, "itemId"))
	if !ok {
		return nil, false
	}
	rd, err := api.ds.Radio().Get(r.Context(), id)
	return rd, err == nil
}

func (api *Router) radioPlaybackInfo(w http.ResponseWriter, r *http.Request, rd model.Radio) {
	src := dto.MediaSourceFromRadio(rd)
	api.ok(w, r, dto.PlaybackInfoResponse{MediaSources: []dto.MediaSourceInfo{src}, PlaySessionId: src.Id})
}

// redirectToStation serves stream URLs clients build themselves instead of playing PlaybackInfo's
// Path. Navidrome doesn't proxy stations, so the client fetches the station directly.
func (api *Router) redirectToStation(w http.ResponseWriter, r *http.Request) bool {
	rd, ok := api.radioForRequest(r)
	if ok {
		http.Redirect(w, r, rd.StreamUrl, http.StatusFound)
	}
	return ok
}

// streamVideo handles /Videos/{itemId}/stream, where real Jellyfin plays Live TV channels, even
// radio ones. Only stations are served here.
func (api *Router) streamVideo(w http.ResponseWriter, r *http.Request) {
	if !api.redirectToStation(w, r) {
		api.notFound(w, r)
	}
}
