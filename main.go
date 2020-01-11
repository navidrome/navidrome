package main

import (
	"fmt"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/persistence"
)

func main() {
	conf.Load()

	fmt.Printf("\nCloudSonic Server v%s\n\n", "0.2")

	provider := persistence.ProviderIdentifier(conf.Sonic.DevPersistenceProvider)

	a := CreateApp(conf.Sonic.MusicFolder, provider)
	a.MountRouter("/rest/", CreateSubsonicAPIRouter(provider))
	a.Run(":" + conf.Sonic.Port)
}
