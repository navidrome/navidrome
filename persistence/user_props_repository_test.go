package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UserProps Repository", func() {
	var ctx context.Context
	var repo model.UserPropsRepository

	BeforeEach(func() {
		ctx = log.NewContext(GinkgoT().Context())
		user := &model.User{ID: "user-props-1", UserName: "user-props-1"}
		Expect(NewUserRepository(GetDBXBuilder()).Put(ctx, user)).To(Succeed())
		repo = NewUserPropsRepository(GetDBXBuilder())
	})

	It("inserts only when the row is absent", func() {
		swapped, err := repo.CompareAndSwap(ctx, "user-props-1", "key", "first", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(swapped).To(BeTrue())

		swapped, err = repo.CompareAndSwap(ctx, "user-props-1", "key", "second", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(swapped).To(BeFalse())
		value, err := repo.Get(ctx, "user-props-1", "key")
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal("first"))
	})

	It("updates only the exact value the caller read", func() {
		Expect(repo.Put(ctx, "user-props-1", "key", "first")).To(Succeed())
		expected := "first"
		swapped, err := repo.CompareAndSwap(ctx, "user-props-1", "key", "second", &expected)
		Expect(err).ToNot(HaveOccurred())
		Expect(swapped).To(BeTrue())

		stale := "first"
		swapped, err = repo.CompareAndSwap(ctx, "user-props-1", "key", "third", &stale)
		Expect(err).ToNot(HaveOccurred())
		Expect(swapped).To(BeFalse())
		value, err := repo.Get(ctx, "user-props-1", "key")
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal("second"))
	})

	It("keeps the value visible through a fresh repository instance", func() {
		Expect(repo.Put(ctx, "user-props-1", "key", "durable")).To(Succeed())
		fresh := NewUserPropsRepository(GetDBXBuilder())
		value, err := fresh.Get(ctx, "user-props-1", "key")
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal("durable"))
	})
})
