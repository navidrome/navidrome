package main

import (
	"fmt"

	"github.com/cloudsonic/sonic-server/conf"
)

func main() {
	conf.Load()

	fmt.Printf("\nCloudSonic Server v%s\n\n", "0.2")

	a := App{}
	a.Initialize()
	a.MountRouter("/rest/", initRouter().Routes())
	a.Run(":" + conf.Sonic.Port)
}
