package agents

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SessionKeys", func() {
	var ctx context.Context
	user := model.User{ID: "u-1"}
	ds := &tests.MockDataStore{MockedUserProps: &tests.MockedUserPropsRepo{}}
	sk := SessionKeys{DataStore: ds, KeyName: "fakeSessionKey"}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("uses the assigned key name", func() {
		Expect(sk.KeyName).To(Equal("fakeSessionKey"))
	})
	It("stores a value in the DB", func() {
		Expect(sk.Put(ctx, user.ID, "test-stored-value")).To(Succeed())
	})
	It("fetches the stored value", func() {
		value, err := sk.Get(ctx, user.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal("test-stored-value"))
	})
	It("deletes the stored value", func() {
		Expect(sk.Delete(ctx, user.ID)).To(Succeed())
	})
	It("handles a not found value", func() {
		_, err := sk.Get(ctx, "u-2")
		Expect(err).To(MatchError(model.ErrNotFound))
	})
})
