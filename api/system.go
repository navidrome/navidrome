package api

import "github.com/deluan/gosonic/api/responses"

type SystemController struct{ BaseAPIController }

func (c *SystemController) Ping() {
	c.SendEmptyResponse()
}

func (c *SystemController) GetLicense() {
	response := c.NewEmpty()
	response.License = &responses.License{Valid: true}
	c.SendResponse(response)
}
