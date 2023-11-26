package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/server/subsonic/responses"
)

func (api *Router) GetOpenSubsonicExtensions(_ *http.Request) (*responses.Subsonic, error) {
	response := newResponse()
	response.OpenSubsonicExtensions = &responses.OpenSubsonicExtensions{
		OpenSubsonicExtensions: []responses.OpenSubsonicExtension{
			{
				Name: "songLyrics",
				Versions: []responses.Version{{
					Version: 1,
				}},
			},
		},
	}
	return response, nil
}
