package main

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/api"
	"github.com/cloudsonic/sonic-server/conf"
)

func main() {
	conf.LoadFromLocalFile()
	conf.LoadFromFlags()

	fmt.Printf("\nCloudSonic Server v%s (%s mode)\n\n", "0.2", beego.BConfig.RunMode)

	a := App{}
	a.Initialize()
	a.MountRouter("/rest/", api.Router())
	a.Run(":" + conf.Sonic.Port)
}
