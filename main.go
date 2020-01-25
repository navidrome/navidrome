package main

import (
	"fmt"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/server"
	"github.com/deluan/navidrome/static"
)

func ShowBanner() {
	banner, _ := static.Asset("banner.txt")
	fmt.Printf(string(banner), server.Version)
}

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
