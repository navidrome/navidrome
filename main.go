package main

import (
	"github.com/cloudsonic/sonic-server/conf"
)

func main() {
	conf.Load()

	a := CreateServer(conf.Sonic.MusicFolder)
	a.MountRouter("/rest", CreateSubsonicAPIRouter())
	a.MountRouter("/app", CreateAppRouter("/app"))
	a.Run(":" + conf.Sonic.Port)
}
