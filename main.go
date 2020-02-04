package main

import (
	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/db"
)

func main() {
	if !conf.Server.DevDisableBanner {
		println(consts.Banner())
	}

	conf.Load()
	db.EnsureLatestVersion()

	a := CreateServer(conf.Server.MusicFolder)
	a.MountRouter("/rest", CreateSubsonicAPIRouter())
	a.MountRouter("/app", CreateAppRouter("/app"))
	a.Run(":" + conf.Server.Port)
}
