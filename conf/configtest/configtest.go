package configtest

import "github.com/SunilSherkhane/navidrome/conf"

// TODO Remove this redirection and call SnapshotConfig directly from tests
func SetupConfig() func() {
	return conf.SnapshotConfig()
}
