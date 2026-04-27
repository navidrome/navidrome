package artwork

import (
	"context"

	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("loadLibraryView", Ordered, func() {
	var ctx context.Context
	var ds *tests.MockDataStore

	BeforeAll(func() {
		storagetest.Register("fake", &storagetest.FakeFS{})
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = &tests.MockDataStore{MockedLibrary: &tests.MockLibraryRepo{}}
	})

	It("returns a view for a library backed by registered storage", func() {
		Expect(ds.Library(ctx).Put(&model.Library{ID: 1, Path: "fake:///music"})).To(Succeed())

		lib, err := loadLibraryView(ctx, ds, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(lib.FS).ToNot(BeNil())
		Expect(lib.absRoot).To(Equal("fake:///music"))
	})

	It("returns an error when the library does not exist", func() {
		_, err := loadLibraryView(ctx, ds, 999)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when the library path uses an unregistered scheme", func() {
		Expect(ds.Library(ctx).Put(&model.Library{ID: 2, Path: "unsupported:///music"})).To(Succeed())
		_, err := loadLibraryView(ctx, ds, 2)
		Expect(err).To(HaveOccurred())
	})
})
