package agents

import (
	"context"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SessionKeys", func() {
	var ctx context.Context
	var ds model.DataStore
	var sk SessionKeys
	user := model.User{ID: "u-1", UserName: "u-1"}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		Expect(ds.User().Put(ctx, &user)).To(Succeed())
		sk = SessionKeys{DataStore: ds, KeyName: "fakeSessionKey"}
	})

	It("uses the assigned key name", func() {
		other := SessionKeys{DataStore: ds, KeyName: "otherSessionKey"}
		Expect(sk.Put(ctx, user.ID, "mine")).To(Succeed())
		Expect(other.Put(ctx, user.ID, "theirs")).To(Succeed())

		Expect(sk.Get(ctx, user.ID)).To(Equal("mine"))
	})
	It("stores a value in the DB", func() {
		Expect(sk.Put(ctx, user.ID, "test-stored-value")).To(Succeed())

		Expect(ds.UserProps().Get(ctx, user.ID, "fakeSessionKey")).To(Equal("test-stored-value"))
	})
	It("fetches the stored value", func() {
		Expect(ds.UserProps().Put(ctx, user.ID, "fakeSessionKey", "test-stored-value")).To(Succeed())

		value, err := sk.Get(ctx, user.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal("test-stored-value"))
	})
	It("deletes the stored value", func() {
		Expect(ds.UserProps().Put(ctx, user.ID, "fakeSessionKey", "test-stored-value")).To(Succeed())

		Expect(sk.Delete(ctx, user.ID)).To(Succeed())

		_, err := ds.UserProps().Get(ctx, user.ID, "fakeSessionKey")
		Expect(err).To(MatchError(model.ErrNotFound))
	})
	It("handles a not found value", func() {
		_, err := sk.Get(ctx, "u-2")
		Expect(err).To(MatchError(model.ErrNotFound))
	})
})
