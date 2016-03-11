package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
)

type GetCoverArtController struct {
	BaseAPIController
	cover engine.Cover
}

func (c *GetCoverArtController) Prepare() {
	utils.ResolveDependencies(&c.cover)
}

func (c *GetCoverArtController) Get() {
	id := c.RequiredParamString("id", "id parameter required")
	size := c.ParamInt("size", 0)

	err := c.cover.Get(id, size, c.Ctx.ResponseWriter)

	switch {
	case err == engine.ErrDataNotFound:
		beego.Error(err, "Id:", id)
		c.SendError(responses.ERROR_DATA_NOT_FOUND, "Directory not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}
}
