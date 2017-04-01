package api

import (
	"io"
	"os"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/utils"
)

type MediaRetrievalController struct {
	BaseAPIController
	cover engine.Cover
}

func (c *MediaRetrievalController) Prepare() {
	utils.ResolveDependencies(&c.cover)
}

func (c *MediaRetrievalController) GetAvatar() {
	var f *os.File
	f, err := os.Open("static/itunes.png")
	if err != nil {
		beego.Error(err, "Image not found")
		c.SendError(responses.ErrorDataNotFound, "Avatar image not found")
	}
	defer f.Close()
	io.Copy(c.Ctx.ResponseWriter, f)
}

func (c *MediaRetrievalController) GetCoverArt() {
	id := c.RequiredParamString("id", "id parameter required")
	size := c.ParamInt("size", 0)

	err := c.cover.Get(id, size, c.Ctx.ResponseWriter)

	switch {
	case err == domain.ErrNotFound:
		beego.Error(err, "Id:", id)
		c.SendError(responses.ErrorDataNotFound, "Cover not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}
}
