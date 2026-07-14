package jellyfin

import (
	"net/http"

	"github.com/navidrome/navidrome/consts"
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
	api.ok(w, r, audioMuseInfoResponse{
		Version:            consts.Version,
		AvailableEndpoints: audioMuseEndpoints,
	})
}
