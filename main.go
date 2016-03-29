package main

import (
	"fmt"

	"github.com/astaxie/beego"
	_ "github.com/deluan/gosonic/init"
	_ "github.com/deluan/gosonic/tasks"
)

func main() {
	fmt.Printf("\nGoSonic v%s (%s mode)\n\n", "0.1", beego.BConfig.RunMode)
	if beego.BConfig.RunMode == "prod" {
		beego.SetLevel(beego.LevelInformational)
	}
	//beego.BConfig.Log.FileLineNum = false
	beego.Run()
}
