package main

import (
	"fmt"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/db"
)

func main() {
	println(consts.Banner())

	conf.Load()
	db.EnsureLatestVersion()

	subsonic, err := CreateSubsonicAPIRouter()
	if err != nil {
		panic(fmt.Sprintf("Could not create the Subsonic API router. Aborting! err=%v", err))
	}
	a := CreateServer(conf.Server.MusicFolder)
	a.MountRouter("/rest", subsonic)
	a.MountRouter("/app", CreateAppRouter("/app"))
	a.Run(":" + conf.Server.Port)
}
