package controllers

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/scanner"
)

type SyncController struct{ beego.Controller }

func (c *SyncController) Get() {
	scanner.StartImport()
	c.Ctx.WriteString("Import started. Check logs")
}


