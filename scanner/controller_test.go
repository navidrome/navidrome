package scanner_test

import (
	"context"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller", func() {
	var ctx context.Context
	var ds model.DataStore
	var ctrl model.Scanner

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	Describe("Status", func() {
		BeforeEach(func() {
			ds = persistence.New(db.Db())
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
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		libs := ds.Library()
		Expect(libs.Put(ctx, &model.Library{ID: 2, Name: "Library 2", Path: "/music2"})).To(Succeed())
		Expect(libs.Put(ctx, &model.Library{ID: 3, Name: "Library 3", Path: "/music3", PIDAlbum: "folder"})).To(Succeed())
		pid := model.Library{}.EffectivePID()
		for _, id := range []int{1, 2, 3} {
			Expect(libs.SetScannedPID(ctx, id, pid)).To(Succeed())
		}
		Expect(libs.ScanEnd(ctx, 3)).To(Succeed())
		Expect(libs.ScanBegin(ctx, 1, true)).To(Succeed())
	})

	It("detects a library that needs a full rescan for a PID change", func() {
		targets := []model.ScanTarget{{LibraryID: 3, FolderPath: "."}}
		Expect(scanner.EffectiveFullScan(ctx, ds, false, targets)).To(BeTrue())
	})

	It("detects an interrupted full scan in a targeted library", func() {
		targets := []model.ScanTarget{{LibraryID: 1, FolderPath: "."}}
		Expect(scanner.EffectiveFullScan(ctx, ds, false, targets)).To(BeTrue())
	})

	It("detects an interrupted full scan when scanning all libraries", func() {
		Expect(scanner.EffectiveFullScan(ctx, ds, false, nil)).To(BeTrue())
	})

	It("ignores interrupted full scans in untargeted libraries", func() {
		targets := []model.ScanTarget{{LibraryID: 2, FolderPath: "."}}
		Expect(scanner.EffectiveFullScan(ctx, ds, false, targets)).To(BeFalse())
	})
})

var _ = Describe("GetInstance", func() {
	It("returns the same controller to every caller", func() {
		ds := persistence.New(db.Db())
		pls := playlists.NewPlaylists(ds, artwork.NewUploader(ds))
		ctx := context.WithoutCancel(GinkgoT().Context())
		a := scanner.GetInstance(ctx, ds, events.NoopBroker(), pls, metrics.NewNoopInstance())
		b := scanner.GetInstance(ctx, ds, events.NoopBroker(), pls, metrics.NewNoopInstance())
		Expect(a).To(BeIdenticalTo(b))
	})
})
