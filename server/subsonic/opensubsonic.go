package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/server/subsonic/responses"
)

func (api *Router) GetOpenSubsonicExtensions(_ *http.Request) (*responses.Subsonic, error) {
	response := newResponse()
	response.OpenSubsonicExtensions = &responses.OpenSubsonicExtensions{
		{Name: "transcodeOffset", Versions: []int32{1}},
		{Name: "formPost", Versions: []int32{1}},
		{Name: "songLyrics", Versions: []int32{1}},
	}
	return response, nil
}
