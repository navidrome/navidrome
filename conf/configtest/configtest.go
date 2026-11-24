package configtest

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/onsi/ginkgo/v2"
)

// RestoreAfterEachSpec restores the config after every spec, and after each Ordered container as a
// whole, so its BeforeAll can set config for all its specs. Call it at the top level of a suite.
func RestoreAfterEachSpec() bool {
	return ginkgo.BeforeEach(ginkgo.OncePerOrdered, func() {
		ginkgo.DeferCleanup(conf.SnapshotConfig())
	})
}
