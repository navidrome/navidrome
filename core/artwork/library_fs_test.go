package artwork

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("loadLibraryView", Ordered, func() {
	var ctx context.Context
	var ds model.DataStore

	BeforeAll(func() {
		DeferCleanup(conf.SnapshotConfig())
		storagetest.Register("fake", &storagetest.FakeFS{})
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = newTestDS(ctx)
	})

	It("returns a view for a library backed by registered storage", func() {
		persistencetest.SetLibraries(ctx, ds, "fake:///music")

		lib, err := loadLibraryView(ctx, ds, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(lib.FS).ToNot(BeNil())
		Expect(lib.absRoot).To(Equal("fake:///music"))
	})

	It("normalizes a library path to an OS root that Abs can join for os.Open/os.Stat", func() {
		// file:// URLs become their parsed OS path; bare paths and non-local schemes are unchanged.
		Expect(localOSRoot("file:///music/library")).To(Equal("/music/library"))
		Expect(localOSRoot("/music/library")).To(Equal("/music/library"))
		Expect(localOSRoot("fake:///music")).To(Equal("fake:///music"))
	})

	It("returns an error when the library does not exist", func() {
		_, err := loadLibraryView(ctx, ds, 999)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when the library path uses an unregistered scheme", func() {
		Expect(ds.Library().Put(ctx, &model.Library{ID: 2, Name: "Unsupported", Path: "unsupported:///music"})).To(Succeed())
		_, err := loadLibraryView(ctx, ds, 2)
		Expect(err).To(HaveOccurred())
	})
})
