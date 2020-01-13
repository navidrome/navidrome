package main

import (
	"github.com/cloudsonic/sonic-server/conf"
)

func main() {
	conf.Load()

	a := CreateApp(conf.Sonic.MusicFolder)
	a.MountRouter("/rest/", CreateSubsonicAPIRouter())
	a.Run(":" + conf.Sonic.Port)
}
