package api

import "github.com/deluan/gosonic/api/responses"

type UsersController struct{ BaseAPIController }

// TODO This is a placeholder. The real one has to read this info from a config file or the database
func (c *UsersController) GetUser() {
	r := c.NewEmpty()
	r.User = &responses.User{}
	r.User.Username = c.RequiredParamString("username", "Required string parameter 'username' is not present")
	r.User.StreamRole = true
	r.User.DownloadRole = true
	c.SendResponse(r)
}
