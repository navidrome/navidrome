package api

import (
	"github.com/deluan/gosonic/api/responses"
)

type GetLicenseController struct{ BaseAPIController }

func (c *GetLicenseController) Get() {
	response := c.NewEmpty()
	response.License = &responses.License{Valid: true}
	c.SendResponse(response)
}
