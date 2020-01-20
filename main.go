package main

import (
	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/server"
)

func main() {
	conf.Load()

	if !conf.Sonic.DevDisableBanner {
		server.ShowBanner()
	}

	a := CreateServer(conf.Sonic.MusicFolder)
	a.MountRouter("/rest", CreateSubsonicAPIRouter())
	a.MountRouter("/app", CreateAppRouter("/app"))
	a.Run(":" + conf.Sonic.Port)
}
