package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/server/subsonic/responses"
)

type SystemController struct{}

func NewSystemController() *SystemController {
	return &SystemController{}
}

func (c *SystemController) Ping(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	return newResponse(), nil
}

func (c *SystemController) GetLicense(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	response := newResponse()
	response.License = &responses.License{Valid: true}
	return response, nil
}
