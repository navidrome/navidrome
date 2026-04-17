package artwork

import (
	"context"

	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("libraryFS", func() {
	var ctx context.Context
	var ds *tests.MockDataStore

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = &tests.MockDataStore{MockedLibrary: &tests.MockLibraryRepo{}}
		storagetest.Register("fake", &storagetest.FakeFS{})
	})

	It("returns an FS for a library backed by file:// storage", func() {
		Expect(ds.Library(ctx).Put(&model.Library{ID: 1, Path: "fake:///music"})).To(Succeed())

		fs, err := libraryFS(ctx, ds, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(fs).ToNot(BeNil())
	})

	It("returns an error when the library does not exist", func() {
		_, err := libraryFS(ctx, ds, 999)
		Expect(err).To(HaveOccurred())
	})
})
