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
})
