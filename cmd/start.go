package cmd

import (
	"fmt"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/db"
)

func start() {
	println(consts.Banner())

	conf.Load()
	db.EnsureLatestVersion()

	subsonic, err := CreateSubsonicAPIRouter()
	if err != nil {
		panic(fmt.Sprintf("Could not create the Subsonic API router. Aborting! err=%v", err))
	}
	a := CreateServer(conf.Server.MusicFolder)
	a.MountRouter(consts.URLPathSubsonicAPI, subsonic)
	a.MountRouter(consts.URLPathUI, CreateAppRouter())
	a.Run(fmt.Sprintf(":%d", conf.Server.Port))
}
