package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Property Repository", func() {
	var ctx context.Context
	var pr model.PropertyRepository

	BeforeEach(func() {
		ctx = log.NewContext(GinkgoT().Context())
		pr = NewPropertyRepository(GetDBXBuilder())
	})

	It("saves and restore a new property", func() {
		id := "1"
		value := "a_value"
		Expect(pr.Put(ctx, id, value)).To(BeNil())
		Expect(pr.Get(ctx, id)).To(Equal("a_value"))
	})

	It("updates a property", func() {
		Expect(pr.Put(ctx, "1", "another_value")).To(BeNil())
		Expect(pr.Get(ctx, "1")).To(Equal("another_value"))
	})

	It("returns a default value if property does not exist", func() {
		Expect(pr.DefaultGet(ctx, "2", "default")).To(Equal("default"))
	})

	It("PutIfAbsent inserts once and never overwrites", func() {
		Expect(pr.PutIfAbsent(ctx, "pia", "first")).To(Succeed())
		Expect(pr.PutIfAbsent(ctx, "pia", "second")).To(Succeed())
		Expect(pr.Get(ctx, "pia")).To(Equal("first"))
	})

	It("hides values marked as secrets from the SQL log, but still logs the property id", func() {
		logs := captureTraceLogs()
		insertCtx := log.WithSecrets(ctx, "inserted-secret")
		Expect(pr.Put(insertCtx, "secret-prop", "inserted-secret")).To(Succeed())
		updateCtx := log.WithSecrets(ctx, "updated-secret")
		Expect(pr.Put(updateCtx, "secret-prop", "updated-secret")).To(Succeed())
		absentCtx := log.WithSecrets(ctx, "absent-secret")
		Expect(pr.PutIfAbsent(absentCtx, "secret-prop-2", "absent-secret")).To(Succeed())

		Expect(logs.String()).To(ContainSubstring("INSERT INTO property"))
		Expect(logs.String()).To(ContainSubstring("UPDATE property"))
		Expect(logs.String()).To(ContainSubstring("secret-prop-2"))
		Expect(logs.String()).ToNot(ContainSubstring("inserted-secret"))
		Expect(logs.String()).ToNot(ContainSubstring("updated-secret"))
		Expect(logs.String()).ToNot(ContainSubstring("absent-secret"))
	})

	It("logs the values of unmarked property writes", func() {
		logs := captureTraceLogs()
		Expect(pr.Put(ctx, "plain-prop", "plain-value")).To(Succeed())

		Expect(logs.String()).To(ContainSubstring("plain-prop"))
		Expect(logs.String()).To(ContainSubstring("plain-value"))
	})
})
