package jellyfin

import (
	"net/http"
	"strings"

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
