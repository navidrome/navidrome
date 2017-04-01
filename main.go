package main

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/conf"
	_ "github.com/cloudsonic/sonic-server/init"
	_ "github.com/cloudsonic/sonic-server/tasks"
)

func main() {
	conf.LoadFromLocalFile()
	conf.LoadFromFlags()

	beego.BConfig.RunMode = conf.Sonic.RunMode
	beego.BConfig.Listen.HTTPPort = conf.Sonic.Port

	fmt.Printf("\nCloudSonic Server v%s (%s mode)\n\n", "0.1", beego.BConfig.RunMode)
	if beego.BConfig.RunMode == "prod" {
		beego.SetLevel(beego.LevelInformational)
	}

	beego.Run()
}
