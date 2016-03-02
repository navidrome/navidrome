package api

type PingController struct{ BaseAPIController }

func (c *PingController) Get() {
	c.SendResponse(c.NewEmpty())
}
