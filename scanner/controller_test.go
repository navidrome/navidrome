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

var _ = Describe("ParseTargets", func() {
	It("parses multiple entries in slice", func() {
		targets, err := scanner.ParseTargets([]string{"1:Music/Rock", "1:Music/Jazz", "2:Classical"})
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(3))
		Expect(targets[0].LibraryID).To(Equal(1))
		Expect(targets[0].FolderPath).To(Equal("Music/Rock"))
		Expect(targets[1].LibraryID).To(Equal(1))
		Expect(targets[1].FolderPath).To(Equal("Music/Jazz"))
		Expect(targets[2].LibraryID).To(Equal(2))
		Expect(targets[2].FolderPath).To(Equal("Classical"))
	})

	It("handles empty folder paths", func() {
		targets, err := scanner.ParseTargets([]string{"1:", "2:"})
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
		Expect(targets[0].FolderPath).To(Equal(""))
		Expect(targets[1].FolderPath).To(Equal(""))
	})

	It("trims whitespace from entries", func() {
		targets, err := scanner.ParseTargets([]string{"  1:Music/Rock", " 2:Classical "})
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
		Expect(targets[0].LibraryID).To(Equal(1))
		Expect(targets[0].FolderPath).To(Equal("Music/Rock"))
		Expect(targets[1].LibraryID).To(Equal(2))
		Expect(targets[1].FolderPath).To(Equal("Classical"))
	})

	It("skips empty strings", func() {
		targets, err := scanner.ParseTargets([]string{"1:Music/Rock", "", "2:Classical"})
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
	})

	It("handles paths with colons", func() {
		targets, err := scanner.ParseTargets([]string{"1:C:/Music/Rock", "2:/path:with:colons"})
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
		Expect(targets[0].FolderPath).To(Equal("C:/Music/Rock"))
		Expect(targets[1].FolderPath).To(Equal("/path:with:colons"))
	})

	It("returns error for invalid format without colon", func() {
		_, err := scanner.ParseTargets([]string{"1Music/Rock"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid target format"))
	})

	It("returns error for non-numeric library ID", func() {
		_, err := scanner.ParseTargets([]string{"abc:Music/Rock"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid library ID"))
	})

	It("returns error for negative library ID", func() {
		_, err := scanner.ParseTargets([]string{"-1:Music/Rock"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid library ID"))
	})

	It("returns error for zero library ID", func() {
		_, err := scanner.ParseTargets([]string{"0:Music/Rock"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid library ID"))
	})

	It("returns error for empty input", func() {
		_, err := scanner.ParseTargets([]string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no valid targets found"))
	})

	It("returns error for all empty strings", func() {
		_, err := scanner.ParseTargets([]string{"", "  ", ""})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no valid targets found"))
	})
})
