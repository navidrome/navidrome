package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/server/subsonic/responses"
)

func (api *Router) JukeboxControl(r *http.Request) (*responses.Subsonic, error) {
	response := newResponse()
	response.JukeboxStatus = &responses.JukeboxStatus{}
	response.JukeboxStatus.CurrentIndex = 0
	response.JukeboxStatus.Playing = false
	response.JukeboxStatus.Gain = 0
	response.JukeboxStatus.Position = 0
	return response, nil
}
