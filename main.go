package main

import (
	"github.com/astaxie/beego"
	_ "github.com/deluan/gosonic/conf"
)

func main() {
	//beego.BConfig.Log.FileLineNum = false
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}
	beego.Run()
}
