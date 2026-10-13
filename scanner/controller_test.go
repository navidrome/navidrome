package scanner_test

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
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
	var ctrl model.Scanner

	Describe("Status", func() {
		BeforeEach(func() {
			ctx = context.Background()
			db.Init(ctx)
			DeferCleanup(func() { Expect(tests.ClearDB()).To(Succeed()) })
			DeferCleanup(configtest.SetupConfig())
			ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}
			ds.MockedProperty = &tests.MockedPropertyRepo{}
			ctrl = scanner.New(ctx, ds, events.NoopBroker(), playlists.NewPlaylists(ds, artwork.NewUploader(ds)), metrics.NewNoopInstance())
		})

		It("includes last scan error", func() {
			Expect(ds.Property().Put(ctx, consts.LastScanErrorKey, "boom")).To(Succeed())
			status, err := ctrl.Status(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.LastError).To(Equal("boom"))
		})

		It("includes scan type and error in status", func() {
			// Set up test data in property repo
			Expect(ds.Property().Put(ctx, consts.LastScanErrorKey, "test error")).To(Succeed())
			Expect(ds.Property().Put(ctx, consts.LastScanTypeKey, "full")).To(Succeed())

			// Get status and verify basic info
			status, err := ctrl.Status(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.LastError).To(Equal("test error"))
			Expect(status.ScanType).To(Equal("full"))
		})
	})
})

var _ = Describe("LockForMaintenance", func() {
	It("allows only one database maintenance operation at a time", func() {
		release, ok := scanner.LockForMaintenance()
		Expect(ok).To(BeTrue())
		DeferCleanup(release)

		_, ok = scanner.LockForMaintenance()
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("EffectiveFullScan", func() {
	var ds *tests.MockDataStore

	BeforeEach(func() {
		pid := model.Library{}.EffectivePID()
		libraries := &tests.MockLibraryRepo{}
		libraries.SetData(model.Libraries{
			{ID: 1, FullScanInProgress: true, ScannedPIDAlbum: pid.Album, ScannedPIDTrack: pid.Track},
			{ID: 2, ScannedPIDAlbum: pid.Album, ScannedPIDTrack: pid.Track},
			{ID: 3, LastScanAt: time.Now(), PIDAlbum: "folder", ScannedPIDAlbum: pid.Album, ScannedPIDTrack: pid.Track},
		})
		ds = &tests.MockDataStore{MockedLibrary: libraries}
	})

	It("detects a library that needs a full rescan for a PID change", func() {
		targets := []model.ScanTarget{{LibraryID: 3, FolderPath: "."}}
		Expect(scanner.EffectiveFullScan(GinkgoT().Context(), ds, false, targets)).To(BeTrue())
	})

	It("detects an interrupted full scan in a targeted library", func() {
		targets := []model.ScanTarget{{LibraryID: 1, FolderPath: "."}}
		Expect(scanner.EffectiveFullScan(context.Background(), ds, false, targets)).To(BeTrue())
	})

	It("detects an interrupted full scan when scanning all libraries", func() {
		Expect(scanner.EffectiveFullScan(context.Background(), ds, false, nil)).To(BeTrue())
	})

	It("ignores interrupted full scans in untargeted libraries", func() {
		targets := []model.ScanTarget{{LibraryID: 2, FolderPath: "."}}
		Expect(scanner.EffectiveFullScan(context.Background(), ds, false, targets)).To(BeFalse())
	})
})

var _ = Describe("GetInstance", func() {
	It("returns the same controller to every caller", func() {
		ds := &tests.MockDataStore{}
		pls := playlists.NewPlaylists(ds, artwork.NewUploader(ds))
		a := scanner.GetInstance(context.Background(), ds, events.NoopBroker(), pls, metrics.NewNoopInstance())
		b := scanner.GetInstance(context.Background(), ds, events.NoopBroker(), pls, metrics.NewNoopInstance())
		Expect(a).To(BeIdenticalTo(b))
	})
})
