package main

import (
	"fmt"

	"github.com/cloudsonic/sonic-server/conf"
)

func main() {
	conf.Load()

	fmt.Printf("\nCloudSonic Server v%s\n\n", "0.2")

	a := CreateApp(conf.Sonic.MusicFolder)
	a.MountRouter("/rest/", CreateSubsonicAPIRouter())
	a.Run(":" + conf.Sonic.Port)
}
