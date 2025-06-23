package scanner_test

import (
	"context"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller", func() {
	var ctx context.Context
	var ds *tests.MockDataStore
	var ctrl scanner.Scanner

	Describe("Status", func() {
		BeforeEach(func() {
			ctx = context.Background()
			db.Init(ctx)
			DeferCleanup(func() { Expect(tests.ClearDB()).To(Succeed()) })
			DeferCleanup(configtest.SetupConfig())
			ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}
			ds.MockedProperty = &tests.MockedPropertyRepo{}
			ctrl = scanner.New(ctx, ds, artwork.NoopCacheWarmer(), events.NoopBroker(), core.NewPlaylists(ds), metrics.NewNoopInstance())
		})

		It("includes last scan error", func() {
			Expect(ds.Property(ctx).Put(consts.LastScanErrorKey, "boom")).To(Succeed())
			status, err := ctrl.Status(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.LastError).To(Equal("boom"))
		})

		It("includes scan type and error in status", func() {
			// Set up test data in property repo
			Expect(ds.Property(ctx).Put(consts.LastScanErrorKey, "test error")).To(Succeed())
			Expect(ds.Property(ctx).Put(consts.LastScanTypeKey, "full")).To(Succeed())

			// Get status and verify basic info
			status, err := ctrl.Status(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.LastError).To(Equal("test error"))
			Expect(status.ScanType).To(Equal("full"))
		})
	})
})
