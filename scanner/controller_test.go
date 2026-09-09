package scanner_test

import (
	"context"

	"github.com/navidrome/navidrome/conf"
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
		libraries := &tests.MockLibraryRepo{}
		libraries.SetData(model.Libraries{
			{ID: 1, FullScanInProgress: true},
			{ID: 2},
		})
		ds = &tests.MockDataStore{MockedLibrary: libraries}
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

var _ = Describe("PIDConfChanged", func() {
	var ds *tests.MockDataStore
	ctx := context.Background()

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.PID.Album = "album_spec"
		conf.Server.PID.Track = "track_spec"
		ds = &tests.MockDataStore{}
	})

	It("reports no change when every library matches its effective specs", func() {
		repo := &tests.MockLibraryRepo{}
		repo.SetData(model.Libraries{
			{ID: 1, ScannedPIDAlbum: "album_spec", ScannedPIDTrack: "track_spec"},
		})
		ds.MockedLibrary = repo
		Expect(scanner.PIDConfChanged(ctx, ds)).To(BeFalse())
	})

	It("reports a change when the global album spec differs from what was scanned", func() {
		repo := &tests.MockLibraryRepo{}
		repo.SetData(model.Libraries{
			{ID: 1, ScannedPIDAlbum: "old_album", ScannedPIDTrack: "track_spec"},
		})
		ds.MockedLibrary = repo
		Expect(scanner.PIDConfChanged(ctx, ds)).To(BeTrue())
	})

	It("reports a change when only one library has an override that was never scanned", func() {
		repo := &tests.MockLibraryRepo{}
		repo.SetData(model.Libraries{
			{ID: 1, ScannedPIDAlbum: "album_spec", ScannedPIDTrack: "track_spec"},
			{ID: 2, PIDAlbum: "folder", ScannedPIDAlbum: "album_spec", ScannedPIDTrack: "track_spec"},
		})
		ds.MockedLibrary = repo
		Expect(scanner.PIDConfChanged(ctx, ds)).To(BeTrue())
	})

	It("reports a change when the track spec differs but the album spec matches", func() {
		repo := &tests.MockLibraryRepo{}
		repo.SetData(model.Libraries{
			{ID: 1, ScannedPIDAlbum: "album_spec", ScannedPIDTrack: "old_track"},
		})
		ds.MockedLibrary = repo
		Expect(scanner.PIDConfChanged(ctx, ds)).To(BeTrue())
	})

	It("ignores case differences", func() {
		repo := &tests.MockLibraryRepo{}
		repo.SetData(model.Libraries{
			{ID: 1, ScannedPIDAlbum: "ALBUM_SPEC", ScannedPIDTrack: "TRACK_SPEC"},
		})
		ds.MockedLibrary = repo
		Expect(scanner.PIDConfChanged(ctx, ds)).To(BeFalse())
	})
})
