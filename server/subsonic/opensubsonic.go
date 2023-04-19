package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/server/subsonic/responses"
)

func (api *Router) GetOpenSubsonicExtensions(_ *http.Request) (*responses.Subsonic, error) {
	response := newResponse()
	response.OpenSubsonicExtensions = &responses.OpenSubsonicExtensions{}
	return response, nil
}
