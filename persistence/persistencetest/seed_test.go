package persistencetest_test

import (
	"context"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SetLibraries", func() {
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ctx = request.WithUser(GinkgoT().Context(), model.User{ID: "admin", UserName: "admin", IsAdmin: true})
		ds = persistence.New(db.Db())
		Expect(ds.User().Put(ctx, &model.User{ID: "u1", UserName: "u1"})).To(Succeed())
	})

	It("moves library 1 and creates the other libraries", func() {
		ids := persistencetest.SetLibraries(ctx, ds, "/music/one", "/music/two")

		Expect(ids).To(Equal([]int{1, 2}))
		Expect(ds.Library().GetPath(ctx, 1)).To(Equal("/music/one"))
		Expect(ds.Library().GetPath(ctx, 2)).To(Equal("/music/two"))
	})

	It("grants existing users access to every library", func() {
		persistencetest.SetLibraries(ctx, ds, "/music/one", "/music/two")

		u, err := ds.User().FindByUsername(ctx, "u1")
		Expect(err).ToNot(HaveOccurred())
		Expect(u.Libraries).To(HaveLen(2))
	})
})

var _ = Describe("Reset and the library path cache", func() {
	It("forgets library paths cached before the reset", func() {
		ctx := GinkgoT().Context()
		ds := persistence.New(db.Db())
		baseline, err := ds.Library().GetPath(ctx, model.DefaultLibraryID)
		Expect(err).ToNot(HaveOccurred())
		persistencetest.SetLibraries(ctx, ds, "/somewhere/else")
		Expect(ds.Library().GetPath(ctx, model.DefaultLibraryID)).To(Equal("/somewhere/else"))

		persistencetest.Reset()

		Expect(ds.Library().GetPath(ctx, model.DefaultLibraryID)).To(Equal(baseline))
	})
})
