package configtest

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/onsi/ginkgo/v2"
)

// RestoreAfterEachSpec restores the config after every spec. Call it at the top level of a suite.
// Ordered specs are skipped so a BeforeAll can set config for its whole container.
func RestoreAfterEachSpec() bool {
	return ginkgo.BeforeEach(func() {
		if !ginkgo.CurrentSpecReport().IsInOrderedContainer {
			ginkgo.DeferCleanup(conf.SnapshotConfig())
		}
	})
}
