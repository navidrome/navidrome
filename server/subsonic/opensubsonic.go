package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/server/subsonic/responses"
)

func (api *Router) GetOpenSubsonicExtensions(_ *http.Request) (*responses.Subsonic, error) {
	response := newResponse()
	extensions := responses.OpenSubsonicExtensions{
		{Name: "transcodeOffset", Versions: []int32{1}},
		{Name: "formPost", Versions: []int32{1}},
		{Name: "songLyrics", Versions: []int32{1}},
		{Name: "indexBasedQueue", Versions: []int32{1}},
		{Name: "transcoding", Versions: []int32{1}},
	}
	if api.sonic != nil && api.sonic.HasProvider() {
		extensions = append(extensions, responses.OpenSubsonicExtension{
			Name: "sonicSimilarity", Versions: []int32{1},
		})
	}
	response.OpenSubsonicExtensions = &extensions
	return response, nil
}
