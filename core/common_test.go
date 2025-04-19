package core

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
)

var _ = Describe("common.go", func() {
	Describe("userName", func() {
		It("returns the username from context", func() {
			ctx := request.WithUser(context.Background(), model.User{UserName: "testuser"})
			Expect(userName(ctx)).To(Equal("testuser"))
		})

		It("returns 'UNKNOWN' if no user in context", func() {
			ctx := context.Background()
			Expect(userName(ctx)).To(Equal("UNKNOWN"))
		})
	})

	Describe("AbsolutePath", func() {
		var (
			ds    *tests.MockDataStore
			libId int
			path  string
		)

		BeforeEach(func() {
			ds = &tests.MockDataStore{}
			libId = 1
			path = "music/file.mp3"
			mockLib := &tests.MockLibraryRepo{}
			mockLib.SetData(model.Libraries{{ID: libId, Path: "/library/root"}})
			ds.MockedLibrary = mockLib
		})

		It("returns the absolute path when library exists", func() {
			ctx := context.Background()
			abs := AbsolutePath(ctx, ds, libId, path)
			Expect(abs).To(Equal("/library/root/music/file.mp3"))
		})

		It("returns the original path if library not found", func() {
			ctx := context.Background()
			abs := AbsolutePath(ctx, ds, 999, path)
			Expect(abs).To(Equal(path))
		})
	})
})
