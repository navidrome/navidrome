package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/repositories"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type GetIndexesController struct {
	beego.Controller
	repo repositories.ArtistIndex
}

func (c *GetIndexesController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.repo)
}

func (c *GetIndexesController) Get() {
	if c.repo == nil {
		c.CustomAbort(500, "INJECTION NOT WORKING")
	}
}
