package main

import (
	_ "github.com/deluan/gosonic/conf"
	"github.com/astaxie/beego"
)

func main() {
	beego.BConfig.Log.FileLineNum = false
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}
	beego.Run()
}
