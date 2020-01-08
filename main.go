package main

import (
	"fmt"

	"github.com/cloudsonic/sonic-server/api"
	"github.com/cloudsonic/sonic-server/conf"
)

func main() {
	conf.Load()

	fmt.Printf("\nCloudSonic Server v%s\n\n", "0.2")

	a := App{}
	a.Initialize()
	a.MountRouter("/rest/", api.Router())
	a.Run(":" + conf.Sonic.Port)
}
