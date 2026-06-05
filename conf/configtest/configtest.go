package configtest

import "github.com/navidrome/navidrome/conf"

// TODO Remove this redirection and call SnapshotConfig directly from tests
func SetupConfig() func() {
	return conf.SnapshotConfig()
}
