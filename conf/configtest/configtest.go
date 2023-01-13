package configtest

import "github.com/navidrome/navidrome/conf"

func SetupConfig() func() {
	oldValues := *conf.Server
	return func() {
		conf.Server = &oldValues
	}
}
