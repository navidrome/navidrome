package main

import (
	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/db"
)

func main() {
	if !conf.Server.DevDisableBanner {
		ShowBanner()
	}

	conf.Load()
	db.EnsureDB()

	a := CreateServer(conf.Server.MusicFolder)
	a.MountRouter("/rest", CreateSubsonicAPIRouter())
	a.MountRouter("/app", CreateAppRouter("/app"))
	a.Run(":" + conf.Server.Port)
}
