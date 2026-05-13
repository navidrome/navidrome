package configtest

import "github.com/navidrome/navidrome/conf"

func SetupConfig() func() {
	return conf.SnapshotConfig()
}
