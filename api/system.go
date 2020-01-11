package api

import (
	"net/http"

	"github.com/cloudsonic/sonic-server/api/responses"
)

type SystemController struct{}

func NewSystemController() *SystemController {
	return &SystemController{}
}

func (c *SystemController) Ping(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	return NewResponse(), nil
}

func (c *SystemController) GetLicense(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	response := NewResponse()
	response.License = &responses.License{Valid: true}
	return response, nil
}
