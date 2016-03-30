package main

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/conf"
	_ "github.com/deluan/gosonic/conf"
	_ "github.com/deluan/gosonic/init"
	_ "github.com/deluan/gosonic/tasks"
)

func main() {
	conf.LoadFromLocalFile()
	conf.LoadFromFlags()

	beego.BConfig.RunMode = conf.GoSonic.RunMode
	beego.BConfig.Listen.HTTPPort = conf.GoSonic.Port

	fmt.Printf("\nGoSonic v%s (%s mode)\n\n", "0.1", beego.BConfig.RunMode)
	if beego.BConfig.RunMode == "prod" {
		beego.SetLevel(beego.LevelInformational)
	}

	beego.Run()
}
