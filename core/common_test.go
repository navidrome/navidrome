package core

import (
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
)

var _ = Describe("common.go", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	Describe("userName", func() {
		It("returns the username from context", func() {
			Expect(userName(request.WithUser(ctx, model.User{UserName: "testuser"}))).To(Equal("testuser"))
		})

		It("returns 'UNKNOWN' if no user in context", func() {
			Expect(userName(ctx)).To(Equal("UNKNOWN"))
		})
	})

	Describe("AbsolutePath", func() {
		var (
			ds    model.DataStore
			libId int
			path  string
		)

		BeforeEach(func() {
			ds = persistence.New(db.Db())
			path = "music/file.mp3"
			libId = persistencetest.SetLibraries(ctx, ds, "/other", "/library/root")[1]
		})

		It("returns the absolute path when library exists", func() {
			abs := AbsolutePath(ctx, ds, libId, path)
			Expect(abs).To(Equal(filepath.FromSlash("/library/root/music/file.mp3")))
		})

		It("returns the original path if library not found", func() {
			abs := AbsolutePath(ctx, ds, 999, path)
			Expect(abs).To(Equal(path))
		})
	})
})
