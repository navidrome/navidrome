package main

import (
	"fmt"

	"github.com/cloudsonic/sonic-server/conf"
)

func main() {
	conf.Load()

	fmt.Printf("\nCloudSonic Server v%s\n\n", "0.2")

	a := createApp(conf.Sonic.MusicFolder)
	a.MountRouter("/rest/", initRouter())
	a.Run(":" + conf.Sonic.Port)
}
