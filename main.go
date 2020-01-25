package main

import (
	"github.com/deluan/navidrome/conf"
)

func main() {
	conf.Load()

	if !conf.Server.DevDisableBanner {
		ShowBanner()
	}

	a := CreateServer(conf.Server.MusicFolder)
	a.MountRouter("/rest", CreateSubsonicAPIRouter())
	a.MountRouter("/app", CreateAppRouter("/app"))
	a.Run(":" + conf.Server.Port)
}
